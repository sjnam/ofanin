package ofanin

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"slices"
	"sync/atomic"
	"testing"
	"time"
)

func ExampleOrderedFanIn() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ofin := NewOrderedFanIn[string, string](ctx)
	ofin.InputStream = func() <-chan string {
		ch := make(chan string)
		go func() {
			defer close(ch)
			for i := range 4 {
				ch <- fmt.Sprintf("%d", i)
			}
		}()
		return ch
	}()
	ofin.DoWork = func(s string) string {
		return "line:" + s
	}

	for s := range ofin.Process() {
		fmt.Println(s)
	}
	// Output:
	// line:0
	// line:1
	// line:2
	// line:3
}

// inputChan은 슬라이스를 채널로 변환하는 헬퍼입니다.
func inputChan[T any](vals []T) <-chan T {
	ch := make(chan T)
	go func() {
		defer close(ch)
		for _, v := range vals {
			ch <- v
		}
	}()
	return ch
}

// TestOrderPreserved: 병렬 실행에도 입력 순서가 출력에 보장되는지 확인합니다.
// 입력을 셔플하여 "출력 순서 == 입력 순서"임을 타이밍 의존 없이 검증합니다.
func TestOrderPreserved(t *testing.T) {
	ctx := context.Background()
	ofin := NewOrderedFanIn[int, int](ctx)
	ofin.Size = 8

	n := 100
	input := make([]int, n)
	for i := range n {
		input[i] = i * 10
	}
	rand.Shuffle(n, func(i, j int) { input[i], input[j] = input[j], input[i] })

	ofin.InputStream = inputChan(input)
	ofin.DoWork = func(v int) int { return v }

	i := 0
	for got := range ofin.Process() {
		if got != input[i] {
			t.Fatalf("index %d: want %d, got %d", i, input[i], got)
		}
		i++
	}
	if i != n {
		t.Fatalf("expected %d results, got %d", n, i)
	}
}

// TestConcurrency: Size 만큼 고루틴이 동시에 실행되는지 확인합니다.
func TestConcurrency(t *testing.T) {
	ctx := context.Background()
	ofin := NewOrderedFanIn[int, int](ctx)
	ofin.Size = 8

	var active atomic.Int64
	var maxActive atomic.Int64

	ofin.InputStream = inputChan(slices.Collect(func(yield func(int) bool) {
		for i := range 50 {
			if !yield(i) {
				return
			}
		}
	}))
	ofin.DoWork = func(v int) int {
		cur := active.Add(1)
		for {
			m := maxActive.Load()
			if cur <= m || maxActive.CompareAndSwap(m, cur) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		active.Add(-1)
		return v
	}

	for range ofin.Process() {
	}

	if maxActive.Load() < 2 {
		t.Fatalf("expected concurrent execution, maxActive=%d", maxActive.Load())
	}
	if maxActive.Load() > int64(ofin.Size) {
		t.Fatalf("exceeded Size limit: maxActive=%d, Size=%d", maxActive.Load(), ofin.Size)
	}
}

// TestEmptyInput: 빈 입력 채널에서 즉시 종료되는지 확인합니다.
func TestEmptyInput(t *testing.T) {
	ctx := context.Background()
	ofin := NewOrderedFanIn[int, int](ctx)
	ofin.InputStream = inputChan([]int{})
	ofin.DoWork = func(v int) int { return v }

	count := 0
	for range ofin.Process() {
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 results, got %d", count)
	}
}

// TestContextCancellation: ctx 취소 시 조기 종료되는지 확인합니다.
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ofin := NewOrderedFanIn[int, int](ctx)
	ofin.Size = 4

	// 무한 스트림
	infinite := make(chan int)
	go func() {
		defer close(infinite)
		i := 0
		for {
			select {
			case infinite <- i:
				i++
			case <-ctx.Done():
				return
			}
		}
	}()
	ofin.InputStream = infinite
	ofin.DoWork = func(v int) int {
		time.Sleep(time.Millisecond)
		return v
	}

	count := 0
	for range ofin.Process() {
		count++
		if count == 10 {
			cancel()
		}
	}

	if count < 10 {
		t.Fatalf("expected at least 10 results before cancel, got %d", count)
	}
}

// TestSizeZeroDefaultsToCPU: Size=0 이면 NumCPU로 대체되는지 확인합니다.
func TestSizeZeroDefaultsToCPU(t *testing.T) {
	ctx := context.Background()
	ofin := NewOrderedFanIn[int, int](ctx)
	ofin.Size = 0
	ofin.InputStream = inputChan([]int{1, 2, 3})
	ofin.DoWork = func(v int) int { return v * 2 }

	var results []int
	for v := range ofin.Process() {
		results = append(results, v)
	}

	if ofin.Size != runtime.NumCPU() {
		t.Fatalf("expected Size=%d, got %d", runtime.NumCPU(), ofin.Size)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

// TestSingleItem: 아이템이 1개인 경우를 확인합니다.
func TestSingleItem(t *testing.T) {
	ctx := context.Background()
	ofin := NewOrderedFanIn[string, string](ctx)
	ofin.InputStream = inputChan([]string{"hello"})
	ofin.DoWork = func(s string) string { return s + "!" }

	var results []string
	for v := range ofin.Process() {
		results = append(results, v)
	}
	if len(results) != 1 || results[0] != "hello!" {
		t.Fatalf("unexpected results: %v", results)
	}
}
