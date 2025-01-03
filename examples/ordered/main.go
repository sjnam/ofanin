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

	ofin := ofanin.NewOrderedFanIn[string, string](ctx)
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
