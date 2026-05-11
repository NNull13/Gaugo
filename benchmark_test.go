package gaugo

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func BenchmarkExpectedContains(b *testing.B) {
	answer := "Paris is the capital of France. It is known for the Eiffel Tower, " +
		"the Louvre Museum, and its rich cultural heritage spanning centuries of history."
	expected := Expected{
		Contains: []string{"paris", "capital", "eiffel tower", "louvre", "heritage"},
	}

	b.ResetTimer()
	for b.Loop() {
		expectedContainsResult(answer, expected)
	}
}

func BenchmarkLimitDetails(b *testing.B) {
	data := []byte(strings.Repeat("a", 8*1024))

	b.ResetTimer()
	for b.Loop() {
		limitDetails(data, defaultMetricDetailsLimit)
	}
}

func BenchmarkRunnerSmall(b *testing.B) {
	benchmarkRunner(b, 10)
}

func BenchmarkRunnerLarge(b *testing.B) {
	benchmarkRunner(b, 100)
}

func benchmarkRunner(b *testing.B, numCases int) {
	b.Helper()

	for b.Loop() {
		b.StopTimer()

		r, err := NewRunner(WithParallelism(4))
		if err != nil {
			b.Fatalf("NewRunner error: %v", err)
		}

		for i := range numCases {
			name := fmt.Sprintf("case-%d", i)
			if err := r.Case(name,
				Question("What is the capital of France?"),
				ContextDocs(Doc("doc", "Paris is the capital of France.")),
				ExpectedContains("paris"),
			); err != nil {
				b.Fatalf("Case error: %v", err)
			}
		}

		b.StartTimer()

		_, err = r.Run(context.Background(), func(_ context.Context, _ Input) (Output, error) {
			return Output{Answer: "The capital of France is Paris."}, nil
		})
		if err != nil {
			b.Fatalf("Run error: %v", err)
		}
	}
}

func BenchmarkCompactMetrics(b *testing.B) {
	metrics := []Metric{
		nil,
		passMetric{name: "A"},
		nil,
		passMetric{name: "B"},
		nil,
		passMetric{name: "C"},
	}

	b.ResetTimer()
	for b.Loop() {
		compactMetrics(metrics)
	}
}
