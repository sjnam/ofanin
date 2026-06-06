# ctan — Parallel HTTP Fetch in Listing Order

Fetches detailed metadata for every package in the [CTAN](https://ctan.org) registry using the CTAN JSON API, with results printed in the **same order as the package listing** despite being fetched concurrently.

## What It Demonstrates

CTAN hosts thousands of TeX packages. Fetching them one by one is prohibitively slow. This example uses `ofanin` to fan out HTTP requests across 50 workers while preserving the original listing order in the output.

```text
Step 1 — InputStream goroutine:
  GET /json/2.0/packages  →  [pkg-a, pkg-b, pkg-c, ...]
  Streams each package URL into the input channel in listing order.

Step 2 — DoWork (50 concurrent workers):
  GET /json/2.0/pkg/{key}  →  detailed item struct
  Responses arrive out of order depending on network latency.

Step 3 — bridge:
  Reorders results back to listing order before printing.
```

## Key Parameters

| Parameter | Value | Purpose |
| --------- | ----- | ------- |
| `Size` | 50 | Up to 50 concurrent HTTP requests |
| `MaxIdleConnsPerHost` | 50 | Matches `Size` to avoid connection churn |
| HTTP timeout | 30 s | Per-request deadline |

## Run

```bash
go run main.go
```

A network connection is required. Output lines arrive as soon as each package and all preceding packages have been fetched.

## Expected Output

```text
{amsmath  ...}
{babel    ...}
{pgf      ...}
...
```

Each line is a `item` struct printed in the order CTAN lists packages — not the order responses arrived.
