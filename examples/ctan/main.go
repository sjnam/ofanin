package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ofin := ofanin.NewOrderedFanIn[string, result](ctx)
	ofin.Size = 100
	ofin.Lookahead = 64
	
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

			for _, p := range list {
				// Exit without blocking on send when ctx is cancelled
				select {
				case valStream <- CtanAPIURL + "/pkg/" + p.Key:
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

		if resp.StatusCode != http.StatusOK {
			return result{err: fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)}
		}

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
