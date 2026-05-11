# CI Integration

## When to use this

Use this guide when Gaugo evaluations should run on pull requests, nightly jobs, release branches, or deployment gates.

## Recommended CI split

Run deterministic checks on every pull request. Run LLM-backed checks on a slower or credentialed lane.

| Lane | Checks | Cost | Stability | Suggested trigger |
| --- | --- | --- | --- | --- |
| Fast PR | `ExpectedContains`, unit tests | Low | High | Every push |
| Quality PR | `ContextRelevancy`, `Faithfulness`, `AnswerRelevancy` on a small suite | Medium | Medium | Label, protected branch, or merge queue |
| Nightly | Larger semantic suite | Higher | Medium | Scheduled |
| Release | Critical eval suite | Higher | High signal | Before tag or deploy |

## Minimal GitHub Actions workflow

```yaml
name: ci

on:
  pull_request:
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - run: go test ./...
```

This works best for deterministic Gaugo checks that do not need provider credentials.

## Add LLM-backed evaluations

Use repository or organization secrets for provider keys. Do not hard-code keys in tests.

```yaml
name: evals

on:
  workflow_dispatch:
  schedule:
    - cron: "0 3 * * *"

jobs:
  gaugo:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - run: go test ./...
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

In tests, skip LLM-backed suites when credentials are absent:

```go
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
		gaugo.WithCaseTimeout(20*time.Second),
		gaugo.WithMetricDetailsLimit(4096),
	)

	suite.Case("answer is grounded",
		gaugo.Question("What is enterprise pricing?"),
		gaugo.ContextDocs(gaugo.Document{
			ID:   "pricing.md",
			Text: "Enterprise pricing is custom and handled by sales.",
		}),
		gaugo.ExpectedContains("sales"),
	)

	suite.Assert(context.Background(), yourGaugoEvaluation,
		gaugo.ContextRelevancy(gaugo.WithThreshold(0.75)),
		gaugo.Faithfulness(gaugo.WithThreshold(0.8)),
		gaugo.AnswerRelevancy(gaugo.WithThreshold(0.7)),
	)
}
```

## Control rate limits

Provider-backed metrics multiply quickly:

```text
provider calls = cases * LLM-backed metrics
```

For 50 cases with the full RAG triad (`ContextRelevancy`, `Faithfulness`, and
`AnswerRelevancy`), expect 150 judge requests.

Use:

- `WithParallelism` to cap concurrent cases.
- `WithCaseTimeout` to cap per-case runtime.
- Provider `RetryConfig` to handle transient `429`/`5xx` responses and transient transport failures.
- Smaller PR suites for fast feedback.
- Nightly jobs for broad coverage.

For large Anthropic-backed suites, start with `WithParallelism(1)` or
`WithParallelism(2)`, bounded retries, and an explicit provider HTTP timeout.
Make the case timeout large enough for your application call plus every metric
call in that case.

```go
judge, err := anthropic.New(anthropic.Config{
	APIKey: os.Getenv("ANTHROPIC_API_KEY"),
	HTTPClient: &http.Client{
		Timeout: 45 * time.Second,
	},
	Retry: gaugo.RetryConfig{
		MaxAttempts: 4,
		BaseDelay:   250 * time.Millisecond,
		MaxDelay:    8 * time.Second,
	},
})
if err != nil {
	t.Fatalf("anthropic judge config: %v", err)
}

suite := gaugo.New(t,
	gaugo.WithJudge(judge),
	gaugo.WithParallelism(1),
	gaugo.WithCaseTimeout(90*time.Second),
)
```

## Use tags for expensive tests

Go build tags are a simple way to keep expensive evals out of the default lane.

```go
//go:build llm

package rag_test
```

Run tagged evals explicitly:

```sh
go test -tags=llm ./...
```

## Programmatic CI reporting

Use `NewRunner` when CI needs JSON artifacts or custom pass/fail rules.

```go
func runEvals(ctx context.Context, runner *gaugo.Runner, yourGaugoEvaluation gaugo.RunFunc) error {
	result, err := runner.Run(ctx, yourGaugoEvaluation,
		gaugo.ContextRelevancy(gaugo.WithThreshold(0.75)),
		gaugo.Faithfulness(gaugo.WithThreshold(0.8)),
		gaugo.AnswerRelevancy(gaugo.WithThreshold(0.7)),
	)
	if err != nil {
		return err
	}

	if hasFailures(result) {
		return fmt.Errorf("gaugo evaluations failed")
	}

	return nil
}

func hasFailures(result gaugo.RunResult) bool {
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

CI usually benefits from separating operational failures from quality failures.
Retry or quarantine provider outages, rate limits, and parse failures; fail the
quality gate when the application returned an answer and a metric score missed
its threshold. Use `CaseResult.RunError` and `gaugo.MetricErrorInfo` for that
split. See [Results and Reporting](../reference/results-and-reporting.md).

See [Programmatic Runner](programmatic-runner.md) for a full example.

## CI checklist

- Keep deterministic checks in the default `go test ./...` lane.
- Put provider keys in CI secrets only.
- Skip LLM tests when secrets are absent unless the job explicitly requires them.
- Cap parallelism to stay under provider limits.
- Separate operational failures from quality failures in dashboards and rerun policies.
- Use stable case names so failures can be tracked over time.
- Store full results from `Runner` when you need trend analysis.
- Keep metric details bounded with `WithMetricDetailsLimit`.

For provider configuration, see [Provider](../provider/index.md). For production reliability patterns, see [Production](production.md).
