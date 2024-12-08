package oproc

import (
	"context"
	"runtime"
)

type OrderedProc[TI /*input param type*/, TO /*output param type*/ any] struct {
	Ctx         context.Context
	InputStream <-chan TI
	DoWork      func(TI) TO
	Size        int
}

func NewOrderedProc[TI, TO any](ctx context.Context) *OrderedProc[TI, TO] {
	return &OrderedProc[TI, TO]{
		Ctx:  ctx,
		Size: runtime.NumCPU(),
	}
}

func (o *OrderedProc[TI, TO]) Process() <-chan TO {
	orDone := func(c <-chan TO) <-chan TO {
		ch := make(chan TO)
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

	chanchan := func() <-chan <-chan TO {
		chch := make(chan (<-chan TO), o.Size)
		go func() {
			defer close(chch)
			for v := range o.InputStream {
				ch := make(chan TO)
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
	return func(chch <-chan <-chan TO) <-chan TO {
		vch := make(chan TO)
		go func() {
			defer close(vch)
			for {
				var ch <-chan TO
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
