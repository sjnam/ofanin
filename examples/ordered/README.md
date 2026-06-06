# ordered — Core Pattern Demo

The canonical `ofanin` example. It feeds 100 items through 20 concurrent workers where each worker takes a **random** amount of time (50–200 ms), so items finish in unpredictable internal order — yet the output always arrives in the original input sequence.

## What It Demonstrates

```text
InputStream:  line:  0   line:  1   line:  2  ...  line: 99
                  ↓          ↓          ↓
             [20 workers, each sleeping 50–200 ms at random]
                  ↓          ↓          ↓
Output:       line:  0   line:  1   line:  2  ...  line: 99  ← always ordered
```

Because `DoWork` duration varies, a later item may finish before an earlier one. `ofanin`'s bridge goroutine holds back any out-of-order result until all preceding results have been delivered.

## Key Parameters

| Parameter | Value | Purpose |
| --------- | ----- | ------- |
| `Size` | 20 | Up to 20 items processed concurrently |
| InputStream delay | 1–5 ms per item | Simulates a slow data source |
| `DoWork` duration | 50–200 ms per item | Simulates variable-length work |

## Run

```bash
go run main.go
```

## Expected Output

```text
line:  0 ... is fetched!
line:  1 ... is fetched!
line:  2 ... is fetched!
...
line: 99 ... is fetched!
done ~1s
```

Total time is roughly `ceil(100 / 20) × 200 ms ≈ 1 s`, compared to `100 × 125 ms ≈ 12.5 s` if processed sequentially.
