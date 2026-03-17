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

	// 버퍼 크기를 10으로 설정: 수신자가 없어도 모든 고루틴이 블록 없이 전송 가능
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

			// time.Tick() 대신 NewTicker 사용으로 누수 방지
			t := time.NewTicker(time.Duration(rand.Intn(10000)) * time.Millisecond)
			defer t.Stop()

			select {
			case <-t.C:
				// 에러 여부를 실제 작업 완료 후 결정
				var err error
				if rand.Intn(10000)%4 == 0 {
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
				return
			}
		}
	}()

	wg.Wait()
	close(errChan) // wg.Wait() 이후 모든 전송이 끝난 뒤 닫기
}
