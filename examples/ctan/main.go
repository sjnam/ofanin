package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/sjnam/ofanin"
)

const CtanAPIURL = "https://ctan.org/json/2.0"

type item struct {
	ID      string `json:"id,omitempty"`
	Key     string `json:"key,omitempty"`
	Name    string `json:"name,omitempty"`
	Caption string `json:"caption,omitempty"`
	Authors []struct {
		ID     string `json:"id"`
		Active bool   `json:"active"`
	} `json:"authors,omitempty"`
	Topics []string `json:"topics,omitempty"`
}

type result struct {
	item *item
	err  error
}

func main() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ofin := ofanin.NewOrderedFanIn[string, result](ctx)
	ofin.Size = 50

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: ofin.Size,
		},
	}

	ofin.InputStream = func() <-chan string {
		valStream := make(chan string)
		go func() {
			defer close(valStream)

			resp, err := client.Get(CtanAPIURL + "/packages")
			if err != nil {
				log.Println(err)
				return
			}
			defer resp.Body.Close()

			var list []item
			if err = json.NewDecoder(resp.Body).Decode(&list); err != nil {
				log.Println(err)
				return
			}

			for p := range slices.Values(list) {
				// ctx 취소 시 전송 블록 없이 종료
				select {
				case valStream <- fmt.Sprintf("%s/%s/%s", CtanAPIURL, "pkg", p.Key):
				case <-ctx.Done():
					return
				}
			}
		}()
		return valStream
	}()

	ofin.DoWork = func(url string) result {
		resp, err := client.Get(url)
		if err != nil {
			return result{err: err}
		}
		defer resp.Body.Close()

		var o item
		if err = json.NewDecoder(resp.Body).Decode(&o); err != nil {
			return result{err: fmt.Errorf("%s: %w", url, err)}
		}
		return result{item: &o}
	}

	for r := range ofin.Process() {
		if r.err != nil {
			log.Println(r.err)
			continue
		}
		fmt.Println(*r.item)
	}
}
