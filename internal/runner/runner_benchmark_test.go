package runner

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
)

func BenchmarkRunIndexedCapacity(b *testing.B) {
	caseSizes := []int{100, 1_000, 10_000}
	workers := []int{1, 8, 64}

	for _, total := range caseSizes {
		total := total
		for _, w := range workers {
			w := w
			b.Run(fmt.Sprintf("cases=%d/workers=%d", total, w), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					var count int64
					RunIndexed(context.Background(), w, total, func(_ context.Context, _ int) {
						atomic.AddInt64(&count, 1)
					})
					if got := int(count); got != total {
						b.Fatalf("processed=%d want=%d", got, total)
					}
				}
			})
		}
	}
}
