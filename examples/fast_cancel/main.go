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

	// Buffer size 10: all goroutines can send without blocking even with no receiver
	errChan := make(chan error, 10)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	wg.Add(10)
	for i := range 10 {
		go func() {
			eStr := "finished"
			start := time.Now()

			defer func() {
				log.Printf("goroutine[%d] %v %s", i, time.Since(start), eStr)
				wg.Done()
			}()

			// Use NewTicker instead of time.Tick() to prevent goroutine leak
			t := time.NewTicker(time.Duration(rand.Intn(10000)) * time.Millisecond)
			defer t.Stop()

			select {
			case <-t.C:
				// Decide whether to report an error after the actual work completes
				var err error
				if rand.Intn(4) == 0 {
					err = fmt.Errorf("ERROR[%d]", i)
				}
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
				// Drain the channel fully instead of returning early to avoid missing errors
			}
		}
	}()

	wg.Wait()
	close(errChan) // Close after wg.Wait() ensures all sends are done
}
