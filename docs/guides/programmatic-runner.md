# Programmatic Runner

## When to use this

Use `Runner` when you want Gaugo results as data instead of direct `testing.T` failures. This is the right entrypoint for CLIs, scheduled eval jobs, dashboards, pipelines, notebooks, and custom CI reporting.

## Create a runner

```go
runner, err := gaugo.NewRunner(
	gaugo.WithParallelism(8),
	gaugo.WithCaseTimeout(20*time.Second),
	gaugo.WithMetricDetailsLimit(8*1024),
)
if err != nil {
	return err
}
```

`NewRunner` validates options and returns an error instead of failing a test.

## Register cases

`Runner.Case` returns an error for invalid cases or duplicate names.

```go
if err := runner.Case("enterprise pricing",
	gaugo.Question("What is enterprise pricing?"),
	gaugo.ContextDocs(gaugo.Document{
		ID:   "pricing.md",
		Text: "Enterprise pricing is custom and handled by sales.",
	}),
	gaugo.ExpectedContains("sales"),
); err != nil {
	return err
}
```

## Run deterministic checks

```go
result, err := runner.Run(ctx, func(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
	return gaugo.Output{
		Answer: "Enterprise pricing is custom. Contact sales for a quote.",
	}, nil
})
if err != nil {
	return err
}

for _, c := range result.Cases {
	if c.RunError != nil {
		log.Printf("%s run failed: %v", c.Name, c.RunError)
		continue
	}
	for _, m := range c.Metrics {
		log.Printf("%s %s pass=%t score=%.2f", c.Name, m.Name, m.Pass, m.Score)
	}
}
```

If a run has no `ExpectedContains` checks and no metrics, `Run` returns an error. Gaugo rejects vacuous evaluations.

## Run semantic metrics

LLM-backed metrics require `WithJudge`.

```go
judge, err := openai.New(openai.Config{
	APIKey: os.Getenv("OPENAI_API_KEY"),
	Model:  "gpt-4.1-mini",
})
if err != nil {
	return err
}

runner, err := gaugo.NewRunner(
	gaugo.WithJudge(judge),
	gaugo.WithParallelism(4),
	gaugo.WithCaseTimeout(15*time.Second),
)
if err != nil {
	return err
}

if err := runner.Case("grounded pricing answer",
	gaugo.Question("What is enterprise pricing?"),
	gaugo.ContextDocs(gaugo.Document{
		ID:   "pricing.md",
		Text: "Enterprise pricing is custom and handled by sales.",
	}),
); err != nil {
	return err
}

result, err := runner.Run(ctx, yourGaugoEvaluation,
	gaugo.Faithfulness(gaugo.WithThreshold(0.8)),
	gaugo.AnswerRelevancy(gaugo.WithThreshold(0.7)),
)
if err != nil {
	return err
}
```

## Complete CLI-style example

```go
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/nnull13/gaugo"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	runner, err := gaugo.NewRunner(
		gaugo.WithParallelism(4),
		gaugo.WithCaseTimeout(10*time.Second),
		gaugo.WithMetricDetailsLimit(4096),
	)
	if err != nil {
		return err
	}

	if err := runner.Case("enterprise pricing",
		gaugo.Question("What is enterprise pricing?"),
		gaugo.ExpectedContains("sales"),
	); err != nil {
		return err
	}

	result, err := runner.Run(ctx, yourGaugoEvaluation)
	if err != nil {
		return err
	}

	return json.NewEncoder(os.Stdout).Encode(result)
}

func yourGaugoEvaluation(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
	return gaugo.Output{Answer: "Contact sales for enterprise pricing."}, nil
}
```

## Turn results into pass/fail

`Runner.Run` returns a `RunResult` even when cases or metrics fail. Operational setup errors are returned as `error`; evaluation failures are recorded inside the result.

```go
func failed(result gaugo.RunResult) bool {
	for _, c := range result.Cases {
		if c.RunError != nil {
			return true
		}
		for _, m := range c.Metrics {
			if !m.Pass {
				return true
			}
		}
	}
	return false
}
```

This separation lets pipelines decide whether to fail immediately, upload partial results, compare against a baseline, or apply custom thresholds.

## Runner tips

- Register all cases before calling `Run`.
- Preserve the order of cases if downstream tools compare results over time.
- Use `WithReporter` only for side effects; the main result is still returned by `Run`.
- Use `WithMetricDetailsLimit(0)` when results are stored in small CI artifacts or logs.
- Use context cancellation to stop long-running eval batches from external orchestration.

For result fields, see [Results and Reporting](../reference/results-and-reporting.md). For custom reporters, see [Custom Reporters](../extending/custom-reporters.md).
