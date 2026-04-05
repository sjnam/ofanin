package ofanin

import (
	"context"
	"runtime"
)

type OrderedFanIn[IN, OUT any] struct {
	Ctx         context.Context
	InputStream <-chan IN
	DoWork      func(IN) OUT
	Size        int
}

func NewOrderedFanIn[IN, OUT any](ctx context.Context) *OrderedFanIn[IN, OUT] {
	return &OrderedFanIn[IN, OUT]{
		Ctx:  ctx,
		Size: runtime.NumCPU(),
	}
}

func (o *OrderedFanIn[IN, OUT]) Process() <-chan OUT {
	if o.InputStream == nil {
		panic("ofanin: InputStream must be set before calling Process()")
	}
	if o.DoWork == nil {
		panic("ofanin: DoWork must be set before calling Process()")
	}
	if o.Size <= 0 {
		o.Size = runtime.NumCPU()
	}
	return o.bridge(o.fanOut())
}

// fanOut reads from InputStream and spawns a goroutine per item.
// A semaphore (sem) limits concurrent DoWork goroutines to exactly Size.
func (o *OrderedFanIn[IN, OUT]) fanOut() <-chan (<-chan OUT) {
	chch := make(chan (<-chan OUT), o.Size)
	sem := make(chan struct{}, o.Size)
	go func() {
		defer close(chch)
		for {
			select {
			case v, ok := <-o.InputStream:
				if !ok {
					return
				}
				// 세마포어 획득: Size개 초과 시 블록
				select {
				case sem <- struct{}{}:
				case <-o.Ctx.Done():
					return
				}
				ch := make(chan OUT, 1)
				select {
				case chch <- ch:
				case <-o.Ctx.Done():
					<-sem
					return
				}
				go func() {
					defer func() { <-sem }() // 세마포어 반환
					defer close(ch)
					ch <- o.DoWork(v)
				}()
			case <-o.Ctx.Done():
				return
			}
		}
	}()
	return chch
}

// bridge iterates the channel-of-channels sequentially, preserving input order.
func (o *OrderedFanIn[IN, OUT]) bridge(chch <-chan (<-chan OUT)) <-chan OUT {
	out := make(chan OUT)
	go func() {
		defer close(out)
		for {
			var ch <-chan OUT
			select {
			case maybe, ok := <-chch:
				if !ok {
					return
				}
				ch = maybe
			case <-o.Ctx.Done():
				return
			}
			// 각 내부 채널은 값을 1개만 전송하므로 orDone 없이 직접 읽음
			select {
			case v, ok := <-ch:
				if !ok {
					continue
				}
				select {
				case out <- v:
				case <-o.Ctx.Done():
					return
				}
			case <-o.Ctx.Done():
				return
			}
		}
	}()
	return out
}
