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

	ofin := ofanin.NewOrderedFanIn[item /*input param*/, item /*output param*/](ctx)
	ofin.InputStream = func() <-chan item {
		valStream := make(chan item)
		go func() {
			defer close(valStream)
			resp, err := http.Get("https://ctan.org/json/2.0/packages")
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			var list []item
			if err = json.NewDecoder(resp.Body).Decode(&list); err != nil {
				log.Fatal(err)
			}

			for o := range slices.Values(list) {
				valStream <- o
			}
		}()
		return valStream
	}()
	ofin.DoWork = func(o item) item {
		resp, err := http.Get("https://ctan.org/json/2.0/pkg/" + o.Key)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		if err = json.NewDecoder(resp.Body).Decode(&o); err != nil {
			log.Fatal(o.ID, err)
		}
		return o
	}
	ofin.Size = 20

	for s := range ofin.Process() {
		b, err := json.Marshal(s)
		if err == nil {
			fmt.Println(string(b))
		}
	}
}
