package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/sjnam/ofanin"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ofin := ofanin.NewOrderedFanIn[string, string](ctx)

	ofin.InputStream = func() <-chan string {
		ch := make(chan string)
		go func() {
			defer close(ch)
			for i := range 100 {
				time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)
				ch <- fmt.Sprintf("line:%3d", i)
			}
		}()
		return ch
	}()

	ofin.DoWork = func(str string) string {
		// Variable duration (50~200ms) causes items to finish out of input order,
		// making the ordering guarantee visible in the output.
		time.Sleep(time.Duration(50+rand.Intn(151)) * time.Millisecond)
		return fmt.Sprintf("%s ... is fetched!", str)
	}

	ofin.Size = 20

	start := time.Now()

	for s := range ofin.Process() {
		fmt.Println(s)
	}

	fmt.Println("done", time.Since(start))
}
