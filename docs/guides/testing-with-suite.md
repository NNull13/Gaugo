# Testing with Suite

## When to use this

Use `Suite` when Gaugo evaluations should behave like normal Go tests: failures are reported through `testing.TB`, `go test ./...` is the runner, and CI already understands the result.

## Basic structure

```go
package rag_test

import (
	"context"
	"testing"

	"github.com/nnull13/gaugo"
)

func TestAnswerContract(t *testing.T) {
	suite := gaugo.New(t)

	suite.Case("enterprise pricing",
		gaugo.Question("What is enterprise pricing?"),
		gaugo.ExpectedContains("sales"),
	)

	suite.Assert(context.Background(), func(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
		return gaugo.Output{Answer: "Contact sales for enterprise pricing."}, nil
	})
}
```

This test has no LLM dependency. It is a good smoke test for response contracts that should never regress.

## Add context documents

Context documents are passed to your `RunFunc` and to LLM-backed metrics.

```go
suite.Case("refund window",
	gaugo.Question("How long do I have to request a refund?"),
	gaugo.ContextDocs(
		gaugo.Document{
			ID:   "refunds.md",
			Text: "Customers can request refunds within 30 days.",
		},
	),
	gaugo.ExpectedContains("30 days"),
)
```

Use stable document IDs. They make failures easier to debug when metric details include context.

## Add an LLM judge

Use `WithJudge` when you add semantic metrics such as `Faithfulness` and `AnswerRelevancy`.

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

func TestRAGWithLLMJudge(t *testing.T) {
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
		gaugo.WithMetricDetailsLimit(4096),
	)

	suite.Case("grounded pricing answer",
		gaugo.Question("What is enterprise pricing?"),
		gaugo.ContextDocs(gaugo.Document{
			ID:   "pricing.md",
			Text: "Enterprise pricing is custom and handled by sales.",
		}),
		gaugo.ExpectedContains("sales"),
	)

	suite.Assert(context.Background(), yourGaugoEvaluation,
		gaugo.Faithfulness(gaugo.WithThreshold(0.8)),
		gaugo.AnswerRelevancy(gaugo.WithThreshold(0.7)),
	)
}

func yourGaugoEvaluation(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
	return gaugo.Output{
		Answer: "Enterprise pricing is custom and handled by sales.",
	}, nil
}
```

## How failures are reported

`suite.Assert` runs all registered cases and then reports failures.

- Invalid setup fails immediately: invalid options, invalid cases, duplicate names, missing run function, or no effective checks.
- A `RunFunc` error or panic is reported as a case run failure.
- A failed metric is reported with its score and reason.
- Default assertion logs include safe detail metadata (`details_bytes`) rather than raw detail payloads.

If you configure `WithReporter`, Gaugo sends the `RunResult` to your reporter and does not use the default testing reporter for that suite. This is useful when you want to inspect results without failing the test automatically.

```go
type captureReporter struct {
	result gaugo.RunResult
}

func (r *captureReporter) Report(ctx context.Context, result gaugo.RunResult) {
	r.result = result
}

func TestCaptureResults(t *testing.T) {
	reporter := &captureReporter{}
	suite := gaugo.New(t, gaugo.WithReporter(reporter))

	suite.Case("contains check",
		gaugo.Question("Where do I buy enterprise?"),
		gaugo.ExpectedContains("sales"),
	)

	suite.Assert(context.Background(), func(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
		return gaugo.Output{Answer: "Contact sales."}, nil
	})

	if len(reporter.result.Cases) != 1 {
		t.Fatalf("case count = %d, want 1", len(reporter.result.Cases))
	}
}
```

## Suite tips

- Start with deterministic `ExpectedContains` checks before adding LLM metrics.
- Keep case names stable; they are the primary handle in failure output and dashboards.
- Keep `RunFunc` small; call your application code rather than rewriting product logic in the test.
- Use `t.Skip` when provider credentials are intentionally absent in local development.
- Use `WithCaseTimeout` to prevent one slow case from blocking the whole test.
- Tune `WithParallelism` to match provider rate limits and test environment capacity.

For the programmatic API, see [Programmatic Runner](programmatic-runner.md). For production hardening, see [Production](production.md).
