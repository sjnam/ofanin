package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup

	errChan := make(chan error)
	defer close(errChan)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			eStr := "finished"
			start := time.Now()

			var err error
			if rand.Intn(10000)%4 == 0 {
				err = fmt.Errorf("ERROR[%d]", i)
			}

			defer func() {
				log.Printf("goroutine[%d] %v %s", i, time.Now().Sub(start), eStr)
				wg.Done()
			}()

			select {
			case <-time.Tick(time.Duration(rand.Intn(10000)) * time.Millisecond):
				errChan <- err
			case <-ctx.Done():
				eStr = ctx.Err().Error()
			}
		}()
	}

	go func() {
		for err := range errChan {
			if err != nil {
				cancel()
				return
			}
		}
	}()

	wg.Wait()
}
