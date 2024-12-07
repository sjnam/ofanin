# Ordered Process
You can use this `oproc` package when you want to speed up processing with goroutines
while guaranteeing ordering.

## Usage
See `ordered` and `ctan` of examples directory for examples of its use.
````go
package oproc

import (
	"context"
	"fmt"
)

func ExampleOrderedProc() {
	inputStream := func() <-chan string {
		ch := make(chan string)
		go func() {
			defer close(ch)
			for i := 0; i < 4; i++ {
				ch <- fmt.Sprintf("%d", i)
			}
		}()
		return ch
	}

	doWork := func(str string) string {
		return fmt.Sprintf("line:%s", str)
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	for s := range OrderedProc(ctx, inputStream(), doWork) {
		fmt.Println(s)
		// Output:
		// line:0
		// line:1
		// line:2
		// line:3
	}
}
````
