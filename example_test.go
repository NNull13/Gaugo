package gaugo_test

import (
	"context"
	"fmt"
	"time"

	"github.com/nnull13/gaugo"
)

func ExampleNew() {
	// In a real test file, use the *testing.T from the test function.
	// suite := gaugo.New(t, gaugo.WithParallelism(2))
	// suite.Case("basic", gaugo.Question("What is Go?"), gaugo.ExpectedContains("programming"))
	// suite.Assert(ctx, runFunc)

	// This example shows the Doc helper and case registration pattern.
	fmt.Println("gaugo.New(t, opts...) creates a testing Suite")
	// Output:
	// gaugo.New(t, opts...) creates a testing Suite
}

func ExampleNewRunner() {
	r, err := gaugo.NewRunner(gaugo.WithParallelism(2))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	err = r.Case("capital-check",
		gaugo.Question("What is the capital of France?"),
		gaugo.ContextDocs(gaugo.Doc("wiki", "Paris is the capital of France.")),
		gaugo.ExpectedContains("paris"),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	result, err := r.Run(context.Background(),
		func(_ context.Context, in gaugo.Input) (gaugo.Output, error) {
			return gaugo.Output{Answer: "The capital of France is Paris."}, nil
		},
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("failed:", result.Failed())
	// Output:
	// failed: false
}

func ExampleDoc() {
	doc := gaugo.Doc("doc-1", "Go is a statically typed language.")
	fmt.Println("ID:", doc.ID)
	fmt.Println("Text:", doc.Text)
	// Output:
	// ID: doc-1
	// Text: Go is a statically typed language.
}

func ExampleRunResult_Summary() {
	result := gaugo.RunResult{
		Cases: []gaugo.CaseResult{
			{
				Name: "case-1",
				Metrics: []gaugo.MetricResult{
					{Name: "ExpectedContains", Score: 1, Pass: true, Reason: "all required substrings present"},
					{Name: "Faithfulness", Score: 0.8, Pass: true, Reason: "ok"},
				},
				Elapsed: 50 * time.Millisecond,
			},
			{
				Name: "case-2",
				Metrics: []gaugo.MetricResult{
					{Name: "ExpectedContains", Score: 0, Pass: false, Reason: "missing"},
				},
				Elapsed: 30 * time.Millisecond,
			},
		},
	}
	fmt.Println(result.Summary())
	// Output:
	// cases=2 failed_cases=0 metrics=3 failed_metrics=1 pass_rate=66.7%
}

func ExampleRunResult_Failed() {
	passing := gaugo.RunResult{
		Cases: []gaugo.CaseResult{
			{Metrics: []gaugo.MetricResult{{Pass: true}}},
		},
	}
	fmt.Println("passing:", passing.Failed())

	failing := gaugo.RunResult{
		Cases: []gaugo.CaseResult{
			{Metrics: []gaugo.MetricResult{{Pass: false}}},
		},
	}
	fmt.Println("failing:", failing.Failed())
	// Output:
	// passing: false
	// failing: true
}

func ExampleWithThreshold() {
	// Create a Faithfulness metric that requires at least 90% score to pass.
	m := gaugo.Faithfulness(gaugo.WithThreshold(0.9))
	fmt.Println("metric:", m.Name())
	// Output:
	// metric: Faithfulness
}
