package oproc

import (
	"context"
	"fmt"
)

func ExampleOrderedProc() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	my := NewOrderedProc[string /*input param*/, string /*output param*/](ctx)
	my.InputStream = func() <-chan string {
		ch := make(chan string)
		go func() {
			defer close(ch)
			for i := 0; i < 4; i++ {
				ch <- fmt.Sprintf("%d", i)
			}
		}()
		return ch
	}()
	my.DoWork = func(str string) string {
		return fmt.Sprintf("line:%s", str)
	}

	for s := range my.Process() {
		fmt.Println(s)
		// Output:
		// line:0
		// line:1
		// line:2
		// line:3
	}
}
