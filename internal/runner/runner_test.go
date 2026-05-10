package runner

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRunIndexedExecutesAllOnce(t *testing.T) {
	t.Parallel()

	const total = 200
	seen := make([]int, total)
	var mu sync.Mutex

	RunIndexed(context.Background(), 32, total, func(_ context.Context, idx int) {
		mu.Lock()
		seen[idx]++
		mu.Unlock()
	})

	for i, c := range seen {
		if c != 1 {
			t.Fatalf("index %d seen=%d want=1", i, c)
		}
	}
}

func TestRunIndexedCancellationStopsEarly(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		mu    sync.Mutex
		count int
	)
	RunIndexed(ctx, 8, 10_000, func(_ context.Context, _ int) {
		mu.Lock()
		count++
		if count == 50 {
			cancel()
		}
		mu.Unlock()
		time.Sleep(200 * time.Microsecond)
	})

	if count >= 10_000 {
		t.Fatalf("expected early stop on cancellation, got count=%d", count)
	}
}

func TestRunIndexedZeroTotalNoop(t *testing.T) {
	t.Parallel()

	called := false
	RunIndexed(context.Background(), 4, 0, func(_ context.Context, _ int) {
		called = true
	})
	if called {
		t.Fatalf("callback should not be called")
	}
}
