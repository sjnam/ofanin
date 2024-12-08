package oproc

import (
	"context"
	"runtime"
)

func OrderedProc[T, V any](
	ctx context.Context,
	inStream <-chan V,
	doWork func(V) T,
	size ...int,
) <-chan T {
	lvl := runtime.NumCPU()
	if len(size) > 0 {
		lvl = size[0]
	}

	orDone := func(c <-chan T) <-chan T {
		ch := make(chan T)
		go func() {
			defer close(ch)
			for {
				select {
				case <-ctx.Done():
					return
				case v, ok := <-c:
					if !ok {
						return
					}
					select {
					case ch <- v:
					case <-ctx.Done():
					}
				}
			}
		}()
		return ch
	}

	chanchan := func() <-chan <-chan T {
		chch := make(chan (<-chan T), lvl)
		go func() {
			defer close(chch)
			for v := range inStream {
				ch := make(chan T)
				chch <- ch

				go func() {
					defer close(ch)
					ch <- doWork(v)
				}()
			}
		}()
		return chch
	}

	// bridge-channel
	return func(chch <-chan <-chan T) <-chan T {
		vch := make(chan T)
		go func() {
			defer close(vch)
			for {
				var ch <-chan T
				select {
				case maybe, ok := <-chch:
					if !ok {
						return
					}
					ch = maybe
				case <-ctx.Done():
					return
				}
				for v := range orDone(ch) {
					select {
					case vch <- v:
					case <-ctx.Done():
					}
				}
			}
		}()
		return vch
	}(chanchan())
}
