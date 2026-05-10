package runner

import (
	"context"
	"sync"
)

// RunIndexed runs fn for indexes [0,total) with a bounded worker pool.
func RunIndexed(ctx context.Context, workers, total int, fn func(ctx context.Context, index int)) {
	if total <= 0 {
		return
	}
	if workers <= 0 {
		workers = 1
	}
	if workers > total {
		workers = total
	}

	jobs := make(chan int)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case idx, ok := <-jobs:
				if !ok {
					return
				}
				fn(ctx, idx)
			}
		}
	}

	for range workers {
		wg.Add(1)
		go worker()
	}

	for i := range total {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return
		case jobs <- i:
		}
	}

	close(jobs)
	wg.Wait()
}
