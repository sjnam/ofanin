# ofanin — Ordered Fan-In for Go

A small Go generics library that processes a stream of items **concurrently** while delivering results in the **original input order**.

## The Problem

Suppose you have a sequence of inputs — lines from a file, URLs to fetch, records to transform — and each item takes time to process. Running them sequentially is safe but slow. Running them concurrently with goroutines is fast but scrambles the order.

```text
Sequential (ordered, slow):
  [0]→W→[0'] [1]→W→[1'] [2]→W→[2'] [3]→W→[3']

Naive concurrent (fast, but order lost):
  [0]→W↘         ↗[2'] ← finishes first
  [1]→W→ mixed →[0']
  [2]→W↗         ↘[1']
  [3]→W            [3']
```

`ofanin` solves this with the **ordered fan-in** pattern: items are processed in parallel, but results flow out in exactly the same order they came in.

## How It Works

```text
InputStream:   0   1   2   3   4   5  ...
               ↓   ↓   ↓   ↓   ↓   ↓
              [fixed pool of Size workers] ← parallel DoWork
               ↓   ↓   ↓   ↓   ↓   ↓
result slots: ch0 ch1 ch2 ch3 ch4 ch5  ← queued in input order
               ↓
             bridge (drains each slot in turn)
               ↓
Output:        0'  1'  2'  3'  4'  5' ...  ← always in input order
```

The key insight is a **channel-of-channels** (`chan (<-chan OUT)`):

1. For each input item a buffered result channel `ch` is created.
2. A work item `{input, ch}` is sent to a fixed worker pool — workers call `DoWork` and write the result into `ch`.
3. `ch` is also placed, **in input order**, into a queue of channels (`chch`).
4. A bridge goroutine reads `chch` sequentially, waiting for each `ch` to have a value before moving on.

Workers run in parallel (fast), but the bridge enforces order (correct).

## Installation

```bash
go get github.com/sjnam/ofanin
```

Requires Go 1.23 or later.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/sjnam/ofanin"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    ofin := ofanin.NewOrderedFanIn[string, string](ctx)
    ofin.Size = 20 // concurrent workers

    ofin.InputStream = func() <-chan string {
        ch := make(chan string)
        go func() {
            defer close(ch)
            lines := []string{"alpha", "beta", "gamma", "delta"}
            for _, l := range lines {
                ch <- l
            }
        }()
        return ch
    }()

    ofin.DoWork = func(line string) string {
        time.Sleep(100 * time.Millisecond) // simulate work
        return "[processed] " + line
    }

    for result := range ofin.Process() {
        fmt.Println(result)
    }
}
// Output (always in input order despite concurrent processing):
// [processed] alpha
// [processed] beta
// [processed] gamma
// [processed] delta
```

## API

### `NewOrderedFanIn[IN, OUT any](ctx) *OrderedFanIn[IN, OUT]`

Creates a new instance with `Size` defaulting to `runtime.NumCPU()`.

### Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| `Ctx` | `context.Context` | Cancellation control. Do not modify after calling `Process`. |
| `InputStream` | `<-chan IN` | Source of input items. Must be set before `Process`. |
| `DoWork` | `func(IN) OUT` | Applied to each item in a worker goroutine. Must be concurrency-safe. |
| `Size` | `int` | Number of concurrent workers. Defaults to `runtime.NumCPU()` if ≤ 0. |

### `Process() <-chan OUT`

Starts the worker pool and returns a channel of ordered results. The channel is closed when `InputStream` is exhausted or `Ctx` is cancelled.

**Important:**

- Call `Process` at most once per instance.
- Always drain the returned channel **or** cancel `Ctx`; otherwise internal goroutines will leak.

## Examples

| Example | Description |
| ------- | ----------- |
| [ordered](./examples/ordered) | Basic ordered processing with variable-duration work |
| [ctan](./examples/ctan) | Parallel HTTP fetch of CTAN packages in listing order |
| [fast_cancel](./examples/fast_cancel) | Context cancellation on first error across multiple goroutines |

## When to Use

- Processing a file or stream **line by line**, where downstream expects ordered output.
- Parallel HTTP / RPC calls where **response order must match request order**.
- Any CPU- or I/O-bound pipeline stage that is a bottleneck and must remain ordered.

## When Not to Use

- Order doesn't matter — a plain `errgroup` or `sync.WaitGroup` fan-out is simpler.
- Items are so lightweight that goroutine overhead dominates — process sequentially.

## References

1. [chan chan は意外と美味しい](https://qiita.com/hogedigo/items/15af273176599307a2b2)
2. [Concurrency in Go — O'Reilly](https://www.oreilly.com/library/view/concurrency-in-go/9781491941294/)
3. [Sudoku solver using dlx and ofanin](https://github.com/sjnam/dlx/tree/main/examples/sudoku)
