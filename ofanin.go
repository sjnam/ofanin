package ofanin

import (
	"context"
	"runtime"
	"sync"
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

type workItem[IN, OUT any] struct {
	in  IN
	out chan OUT
}

// fanOut distributes input items to a fixed pool of Size workers and enqueues
// result channels into chch in input order, preserving the ordered guarantee.
func (o *OrderedFanIn[IN, OUT]) fanOut() <-chan (<-chan OUT) {
	work := make(chan workItem[IN, OUT], o.Size)
	chch := make(chan (<-chan OUT), o.Size)

	var wg sync.WaitGroup
	wg.Add(o.Size)
	for range o.Size {
		go func() {
			defer wg.Done()
			for item := range work {
				item.out <- o.DoWork(item.in)
				close(item.out)
			}
		}()
	}

	go func() {
		defer func() {
			close(work)
			wg.Wait()
			close(chch)
		}()
		for {
			select {
			case v, ok := <-o.InputStream:
				if !ok {
					return
				}
				ch := make(chan OUT, 1)
				select {
				case work <- workItem[IN, OUT]{in: v, out: ch}:
				case <-o.Ctx.Done():
					return
				}
				select {
				case chch <- ch:
				case <-o.Ctx.Done():
					return
				}
			case <-o.Ctx.Done():
				return
			}
		}
	}()

	return chch
}

// bridge iterates the channel-of-channels sequentially, preserving input order.
func (o *OrderedFanIn[IN, OUT]) bridge(chch <-chan (<-chan OUT)) <-chan OUT {
	out := make(chan OUT, 1)
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
			// Each inner channel sends exactly one value, so read directly without orDone
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
