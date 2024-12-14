# Ordered Fan-In
You can use this `ofanin` package when you want to speed up processing with goroutines
while guaranteeing ordering.

## Usage
See [ordered](./examples/ordered) and [ctan](./examples/ctan) for examples of its use.
````go
package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/sjnam/ofanin"
)

func main() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ofin := ofanin.NewOrderedFanIn[string /*input param*/, string /*output param*/](ctx)
	ofin.InputStream = func() <-chan string {
		ch := make(chan string)
		go func() {
			defer close(ch)
			for i := 0; i < 1000; i++ {
				// sleep instread of reading a file
				time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
				ch <- fmt.Sprintf("line:%3d", i)
			}
		}()
		return ch
	}()
	ofin.DoWork = func(str string) string {
		// sleep instead of fetching
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
		return fmt.Sprintf("%s ... is fetched!", str)
	}

	start := time.Now()

	for s := range ofin.Process() {
		fmt.Println(s)
	}

	fmt.Println("done", time.Now().Sub(start))
}
````

## References
1. [chan chan は意外と美味しい](https://qiita.com/hogedigo/items/15af273176599307a2b2)
1. [Concurrency in Go](https://www.oreilly.com/library/view/concurrency-in-go/9781491941294/)
1. [Sudoku solver with dlx and ofanin](https://github.com/sjnam/dlx/tree/main/examples/sudoku)
