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
	orDone := func(c <-chan OUT) <-chan OUT {
		ch := make(chan OUT)
		go func() {
			defer close(ch)
			for {
				select {
				case <-o.Ctx.Done():
					return
				case v, ok := <-c:
					if !ok {
						return
					}
					select {
					case ch <- v:
					case <-o.Ctx.Done():
					}
				}
			}
		}()
		return ch
	}

	chanchan := func() <-chan <-chan OUT {
		chch := make(chan (<-chan OUT), o.Size)
		go func() {
			defer close(chch)
			for v := range o.InputStream {
				ch := make(chan OUT)
				chch <- ch

				go func() {
					defer close(ch)
					ch <- o.DoWork(v)
				}()
			}
		}()
		return chch
	}

	// bridge-channel
	return func(chch <-chan <-chan OUT) <-chan OUT {
		vch := make(chan OUT)
		go func() {
			defer close(vch)
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
				for v := range orDone(ch) {
					select {
					case vch <- v:
					case <-o.Ctx.Done():
					}
				}
			}
		}()
		return vch
	}(chanchan())
}
