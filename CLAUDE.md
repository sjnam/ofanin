# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run tests
go test -v

# Run a single test
go test -v -run ExampleOrderedFanIn

# Run examples
go run examples/ordered/main.go
go run examples/ctan/main.go
go run examples/fast_cancel/main.go
```

## Architecture

`ofanin` is a single-file Go generics library (`ofanin.go`) implementing an **ordered fan-in** concurrency pattern — parallel processing that preserves input order.

### Core type: `OrderedFanIn[IN, OUT any]`

Fields to set before calling `Process()`:
- `InputStream <-chan IN` — source of input items
- `DoWork func(IN) OUT` — function applied to each item in a goroutine
- `Size int` — max concurrent workers (defaults to `runtime.NumCPU()`)
- `Ctx context.Context` — for cancellation

### How `Process()` works

Uses three nested concurrency primitives:

1. **`chanchan()`** — reads from `InputStream`, spawns a goroutine per item calling `DoWork`, and sends each item's result channel into a buffered `chan (<-chan OUT)` of size `Size`. The buffer acts as the concurrency limiter.

2. **`orDone(ch)`** — wraps a `<-chan OUT` to respect context cancellation.

3. **Bridge channel** — iterates the channel-of-channels sequentially, draining each inner channel before moving to the next. This is what preserves order: items are processed in parallel but results are consumed in the original input sequence.

The pattern is: each input item gets its own channel; those channels are queued in order; the bridge reads them in order. Parallelism comes from goroutines running `DoWork` concurrently even as the bridge reads results sequentially.
