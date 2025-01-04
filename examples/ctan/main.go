package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"

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

func main() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ofin := ofanin.NewOrderedFanIn[string, *item](ctx)
	ofin.InputStream = func() <-chan string {
		valStream := make(chan string)
		go func() {
			defer close(valStream)

			resp, err := http.Get(CtanAPIURL + "/packages")
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			var list []item
			if err = json.NewDecoder(resp.Body).Decode(&list); err != nil {
				log.Fatal(err)
			}

			for p := range slices.Values(list) {
				valStream <- fmt.Sprintf("%s/%s/%s", CtanAPIURL, "pkg", p.Key)
			}
		}()
		return valStream
	}()
	ofin.DoWork = func(url string) *item {
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		var o item
		if err = json.NewDecoder(resp.Body).Decode(&o); err != nil {
			log.Fatal(o.ID, err)
		}
		return &o
	}
	ofin.Size = 20

	for p := range ofin.Process() {
		fmt.Println(p)
	}
}
