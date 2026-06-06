// Package ofanin implements an ordered fan-in concurrency pattern: input items
// are processed in parallel by a fixed worker pool, and results are delivered
// in the same order as the input.
package ofanin

import (
	"context"
	"runtime"
	"sync"
)

// OrderedFanIn processes items from InputStream concurrently while preserving
// the original input order in its output channel.
//
// Set InputStream, DoWork, and optionally Size before calling [OrderedFanIn.Process].
// Process must be called at most once per instance.
type OrderedFanIn[IN, OUT any] struct {
	// Ctx controls cancellation. Cancelling it stops Process early.
	// Do not modify Ctx after calling Process.
	Ctx context.Context

	// InputStream is the source of input items.
	// It must be set before calling Process.
	InputStream <-chan IN

	// DoWork is applied to each input item inside a worker goroutine.
	// It must be set before calling Process and must be safe to call concurrently.
	DoWork func(IN) OUT

	// Size is the number of concurrent workers.
	// Defaults to runtime.NumCPU() when zero or negative.
	Size int
}

// NewOrderedFanIn returns a new OrderedFanIn with Ctx set to ctx and Size
// initialised to runtime.NumCPU(). Set InputStream and DoWork before calling Process.
func NewOrderedFanIn[IN, OUT any](ctx context.Context) *OrderedFanIn[IN, OUT] {
	return &OrderedFanIn[IN, OUT]{
		Ctx:  ctx,
		Size: runtime.NumCPU(),
	}
}

// Process starts the worker pool and returns a channel that delivers results
// in the same order as InputStream. The returned channel is closed when all
// items have been processed or Ctx is cancelled.
//
// Process must be called at most once per OrderedFanIn instance.
// The caller must either drain the returned channel or cancel Ctx; otherwise
// the internal goroutines will leak.
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
			// Each inner channel carries exactly one value written by a worker.
			select {
			case v := <-ch:
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
