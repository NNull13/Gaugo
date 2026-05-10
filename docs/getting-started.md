# Getting Started

## When to use this

Use this guide when you want the fastest path from a Go test file to a working Gaugo evaluation. It covers a deterministic test that needs no LLM provider, then adds an LLM judge for semantic metrics.

## Install

```sh
go get github.com/nnull13/gaugo
```

Gaugo is designed to run inside normal Go projects. You usually add it to a `_test.go` file and run it with `go test`.

## Your first test without an LLM

Start with deterministic checks. They are fast, cheap, stable, and do not need credentials.

```go
package rag_test

import (
	"context"
	"testing"

	"github.com/nnull13/gaugo"
)

func TestEnterprisePricingAnswer(t *testing.T) {
	suite := gaugo.New(t)

	suite.Case("enterprise pricing mentions sales",
		gaugo.Question("What is enterprise pricing?"),
		gaugo.ContextDocs(
			gaugo.Document{
				ID:   "pricing.md",
				Text: "Enterprise plans are custom and sold through the sales team.",
			},
		),
		gaugo.ExpectedContains("sales"),
	)

	suite.Assert(context.Background(), func(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
		return gaugo.Output{
			Answer: "Enterprise pricing is custom. Contact sales for a quote.",
		}, nil
	})
}
```

Run it:

```sh
go test ./...
```

`ExpectedContains` produces a normal Gaugo metric result named `ExpectedContains`. If the answer does not contain the required substring, the test fails through Go's `testing` package.

## Add an LLM judge

LLM-backed metrics need a `gaugo.Judge`. Bundled providers live under `github.com/nnull13/gaugo/provider/...`.

```go
package rag_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nnull13/gaugo"
	"github.com/nnull13/gaugo/provider/openai"
)

func TestRAGQuality(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY is not set")
	}

	judge, err := openai.New(openai.Config{
		APIKey: apiKey,
		Model:  "gpt-4.1-mini",
	})
	if err != nil {
		t.Fatalf("openai judge config: %v", err)
	}

	suite := gaugo.New(t,
		gaugo.WithJudge(judge),
		gaugo.WithParallelism(4),
		gaugo.WithCaseTimeout(15*time.Second),
		gaugo.WithMetricDetailsLimit(8*1024),
	)

	suite.Case("pricing answer is grounded",
		gaugo.Question("What is enterprise pricing?"),
		gaugo.ContextDocs(
			gaugo.Document{
				ID:   "pricing.md",
				Text: "Enterprise pricing is custom and handled by sales.",
			},
		),
		gaugo.ExpectedContains("sales"),
	)

	suite.Assert(context.Background(), yourGaugoEvaluation,
		gaugo.Faithfulness(gaugo.WithThreshold(0.8)),
		gaugo.AnswerRelevancy(gaugo.WithThreshold(0.7)),
	)
}

func yourGaugoEvaluation(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
	answer := "Enterprise pricing is custom and handled by sales."
	return gaugo.Output{Answer: answer}, nil
}
```

`Faithfulness` checks whether the answer is supported by the provided context. `AnswerRelevancy` checks whether the answer addresses the input question. Both require a configured judge.

## Connect Gaugo to your application

The adapter between Gaugo and your product is a `gaugo.RunFunc`:

```go
func(ctx context.Context, in gaugo.Input) (gaugo.Output, error)
```

Keep this adapter thin. Let it call your real application code, return `gaugo.Output{Answer: ...}` on success, and return an error only when the system under test failed to run.

```go
func runSearchAnswer(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
	answer := "Contact sales for enterprise pricing."
	return gaugo.Output{Answer: answer}, nil
}
```

Bad answers should usually return `gaugo.Output`, not `error`. Metrics decide whether the answer passes.

## Choose your next step

- Learn the core model: [Concepts](concepts.md)
- Write test-first evaluations: [Testing with Suite](guides/testing-with-suite.md)
- Collect structured results: [Programmatic Runner](guides/programmatic-runner.md)
- Choose a provider: [Provider](provider/index.md)
- Harden CI usage: [CI Integration](guides/ci-integration.md)
- Prepare production evals: [Production](guides/production.md)
