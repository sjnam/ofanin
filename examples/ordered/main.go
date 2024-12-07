package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/sjnam/oproc"
)

func main() {
	inputStream := func() <-chan string {
		ch := make(chan string)
		go func() {
			defer close(ch)
			for i := 0; i < 1000; i++ {
				time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
				ch <- fmt.Sprintf("%3d", i)
			}
		}()
		return ch
	}

	doWork := func(str string) string {
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
		return fmt.Sprintf("line:%s", str)
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	start := time.Now()

	for s := range oproc.OrderedProc(ctx, inputStream(), doWork) {
		fmt.Println(s)
	}

	fmt.Println("done", time.Now().Sub(start))
}
