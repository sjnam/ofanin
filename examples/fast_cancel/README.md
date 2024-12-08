# Fast Cancellation

Several goroutines with different lifetimes are executed, and if an error occurs in one of them,
all goroutines are terminated.

- The lifetime of each goroutine is less than 10 seconds.
- An error occurs randomly in some goroutine.
- When the goroutine completes execution normally, `nil` is sent to the error channel.
- If an error occurs in a goroutine, the error is sent to the error channel.
