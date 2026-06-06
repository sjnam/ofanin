# fast_cancel — First-Error Cancellation

Runs 10 goroutines in parallel where each finishes after a random delay (1–9999 ms). If any goroutine encounters an error, it signals all others to stop immediately via context cancellation.

## What It Demonstrates

This example is **independent of `ofanin`** — it shows a complementary pattern useful in any concurrent Go program: how to propagate a first-error signal to all sibling goroutines cleanly.

```text
goroutine[0] ─────────────────────────────── done (9 s)
goroutine[1] ──── ERROR (2 s) → cancel()
goroutine[2] ────────── cancelled
goroutine[3] ────────── cancelled
...
```

## Key Design Points

| Concern | Solution |
| ------- | -------- |
| Propagate cancellation | `context.WithCancel`; consumer calls `cancel()` on first non-nil error |
| Avoid goroutine leak from blocking send | Buffered `errChan` sized to goroutine count (10) |
| Avoid missing errors if returning early | Consumer drains `errChan` fully instead of returning after first error |
| Ensure consumer finishes before `main` exits | `done` channel: `close(errChan)` → consumer exits → `<-done` unblocks |
| Prevent ticker goroutine leak | `time.NewTicker` + `defer t.Stop()` instead of `time.Tick` |

## Goroutine Lifecycle

```text
main
 ├─ goroutine[0..9]: wait on ticker or ctx.Done, send err/nil to errChan, wg.Done()
 ├─ consumer:        range errChan → cancel() on error; exits when errChan closes
 │
 wg.Wait()          ← all goroutines done; all sends to errChan complete
 close(errChan)     ← consumer's range loop exits
 <-done             ← main waits for consumer to finish
```

## Run

```bash
go run main.go
```

## Expected Output

```text
2009/11/10 23:00:01 goroutine[3] 2.341s context canceled
2009/11/10 23:00:01 goroutine[7] 2.341s context canceled
2009/11/10 23:00:02 goroutine[1] 2.887s ERROR[1]
2009/11/10 23:00:04 goroutine[5] 4.102s finished
...
```

Goroutines that had not yet fired their ticker print `context canceled`; those that completed normally before the error print `finished`; the goroutine that caused the cancellation prints its error.
