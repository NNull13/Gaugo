# Production

## When to use this

Use this guide when Gaugo evaluations influence release decisions, deployment gates, quality dashboards, or customer-facing AI reliability work.

## Start with deterministic guarantees

Use deterministic checks for behavior that must always hold.

```go
suite.Case("enterprise pricing",
	gaugo.Question("What is enterprise pricing?"),
	gaugo.ContextDocs(gaugo.Document{
		ID:   "pricing.md",
		Text: "Enterprise pricing is custom and handled by sales.",
	}),
	gaugo.ExpectedContains("sales"),
)
```

Good deterministic checks include required escalation paths, support contacts, legal disclaimers, product names, URLs, and policy limits.

## Add semantic metrics for quality

Use LLM-backed metrics for properties that are hard to express as exact string checks.

```go
suite.Assert(context.Background(), yourGaugoEvaluation,
	gaugo.Faithfulness(gaugo.WithThreshold(0.8)),
	gaugo.AnswerRelevancy(gaugo.WithThreshold(0.7)),
)
```

`Faithfulness` is most useful for RAG and grounded generation. `AnswerRelevancy` is useful when the answer must address the user's question directly.

## Configure execution deliberately

```go
suite := gaugo.New(t,
	gaugo.WithJudge(judge),
	gaugo.WithParallelism(4),
	gaugo.WithCaseTimeout(20*time.Second),
	gaugo.WithMetricDetailsLimit(8*1024),
)
```

Production defaults should be explicit:

- Use `WithParallelism` to match provider rate limits and your application's capacity.
- Use `WithCaseTimeout` so one case cannot stall the suite indefinitely.
- Use `WithMetricDetailsLimit` to keep CI logs and result artifacts bounded.
- Use `WithMetricDetailsLimit(0)` when details may contain sensitive data and should not be stored.

## Configure provider resilience

Bundled providers support retry and body-size controls through their config.

```go
judge, err := openai.New(openai.Config{
	APIKey: os.Getenv("OPENAI_API_KEY"),
	Model:  "gpt-4.1-mini",
	Retry: gaugo.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   200 * time.Millisecond,
		MaxDelay:    2 * time.Second,
	},
	MaxResponseBody: 1 << 20,
})
if err != nil {
	return err
}
```

Use `BaseURL` for the official provider API root in production.

```go
judge, err := openai.New(openai.Config{
	APIKey:  os.Getenv("OPENAI_API_KEY"),
	Model:   "gpt-4.1-mini",
	BaseURL: "https://api.openai.com/v1",
})
```

For local stubs, test servers, or custom gateways, set `AllowUnsafeURL: true` explicitly.

## Keep eval data stable

Production evals should change intentionally.

- Keep case names stable.
- Keep documents small enough for the judge model and easy to inspect.
- Prefer one behavior per case.
- Add cases for past incidents and user-visible regressions.
- Avoid hidden network calls inside test fixtures unless the test is explicitly integration-level.
- Keep thresholds documented near the test.

## Separate operational failures from quality failures

`RunFunc` errors mean your system did not complete the case. Metric failures mean the system answered but did not meet the quality bar.

```go
func yourGaugoEvaluation(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
	answer := "Enterprise pricing is custom and handled by sales."
	return gaugo.Output{Answer: answer}, nil
}
```

Do not return an error for a bad answer. Return the answer and let metrics fail it.

## Use Runner for dashboards and pipelines

When results are part of a quality system, use `NewRunner`.

```go
func runProductionEvals(ctx context.Context, judge gaugo.Judge) error {
	runner, err := gaugo.NewRunner(
		gaugo.WithJudge(judge),
		gaugo.WithParallelism(4),
		gaugo.WithCaseTimeout(20*time.Second),
		gaugo.WithMetricDetailsLimit(4096),
	)
	if err != nil {
		return err
	}

	if err := runner.Case("pricing",
		gaugo.Question("What is enterprise pricing?"),
		gaugo.ContextDocs(gaugo.Document{
			ID:   "pricing.md",
			Text: "Enterprise pricing is custom and handled by sales.",
		}),
		gaugo.ExpectedContains("sales"),
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

	for _, c := range result.Cases {
		if c.RunError != nil {
			return fmt.Errorf("%s run failed: %w", c.Name, c.RunError)
		}
		for _, m := range c.Metrics {
			if !m.Pass {
				return fmt.Errorf("%s %s failed: %s", c.Name, m.Name, m.Reason)
			}
		}
	}

	return nil
}
```

Store the full `RunResult` when you need trend charts, case-level failure analysis, or release comparisons.

## Security and privacy

- Treat questions, context documents, answers, reasons, and metric details as potentially sensitive.
- Do not print provider API keys or full request payloads in CI logs.
- The default `gaugo.Assert` reporter logs safe detail metadata (`details_bytes`), not raw detail payloads.
- Use `WithMetricDetailsLimit(0)` for high-sensitivity test suites.
- Use a custom `Judge` or provider gateway when evaluation data must stay inside your infrastructure.
- Use provider `MaxResponseBody` to limit memory exposure from unexpected responses.

## Cost and latency

Estimate judge usage before enabling a suite on every pull request.

```text
judge requests = case count * LLM-backed metric count
```

Reduce cost by keeping PR suites small, using deterministic checks first, running broad semantic suites nightly, and tuning `WithParallelism` to avoid retries from rate limits.

## Recommended gates

- Pull request: deterministic checks plus a small semantic suite for critical workflows.
- Main branch: full unit suite and representative Gaugo suite.
- Nightly: broad semantic suite, dashboards, and trend comparison.
- Release: critical cases with strict thresholds and stored artifacts.

For CI examples, see [CI Integration](ci-integration.md). For configuration details, see [Configuration](../reference/configuration.md). For common failures, see [Troubleshooting](../troubleshooting.md).
