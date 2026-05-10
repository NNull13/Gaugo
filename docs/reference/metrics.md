# Metrics

## When to use this

Use this reference when choosing built-in metrics, setting thresholds, or implementing custom metrics. For custom metric implementations, see [Custom Metrics](../extending/custom-metrics.md).

## Metric interface

```go
type Metric interface {
    Name() string
    Evaluate(ctx context.Context, in gaugo.EvalInput, j gaugo.Judge) (gaugo.MetricResult, error)
}
```

Metrics run after your `RunFunc` returns an `Output`. Built-in LLM metrics call the configured `Judge` and parse strictly structured JSON.

## ExpectedContains

`ExpectedContains` is a deterministic case check, not a value you pass as a `Metric`.

```go
suite.Case("pricing",
    gaugo.Question("What is enterprise pricing?"),
    gaugo.ExpectedContains("sales"),
)

suite.Assert(ctx, yourGaugoEvaluation)
```

Behavior:

- Case-insensitive substring matching.
- Empty or whitespace-only required substrings are invalid at case registration.
- One generated `MetricResult` named `ExpectedContains`.
- Score is the fraction of required substrings found.
- Passes only when all configured substrings are found.
- Does not require an LLM judge.

Use it for hard product contracts: required disclaimers, exact terms, expected support channels, or smoke tests.

## Faithfulness

```go
suite.Assert(ctx, yourGaugoEvaluation, gaugo.Faithfulness())
```

`Faithfulness` scores whether the answer is supported by the case context documents. It requires a configured `Judge`.

```go
judge, err := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4.1-mini",
})
if err != nil {
    t.Fatalf("openai judge config: %v", err)
}

suite := gaugo.New(t, gaugo.WithJudge(judge))
suite.Assert(ctx, yourGaugoEvaluation, gaugo.Faithfulness())
```

The metric asks the judge to extract claims and mark each one as supported or unsupported. The final score is the supported claim ratio.

Use it for RAG systems, support assistants, search answerers, and any product where unsupported claims are a release blocker.

## AnswerRelevancy

```go
suite.Assert(ctx, yourGaugoEvaluation, gaugo.AnswerRelevancy())
```

`AnswerRelevancy` scores whether the answer addresses the user question. It requires a configured `Judge`.

Use it when the answer might be factually grounded but incomplete, evasive, off-topic, or not useful for the user's task.

## Thresholds

Built-in LLM metrics default to a pass threshold of `0.7`.

```go
suite.Assert(ctx, yourGaugoEvaluation,
    gaugo.Faithfulness(gaugo.WithThreshold(0.9)),
    gaugo.AnswerRelevancy(gaugo.WithThreshold(0.8)),
)
```

Thresholds must be in `[0,1]`. Invalid thresholds produce an invalid metric; when used through a runner, that metric becomes a failing `MetricResult`.

## Combining checks

Combine deterministic and LLM-judged checks when you want both contract enforcement and quality scoring.

```go
suite.Case("refunds",
    gaugo.Question("Can I get a refund after 30 days?"),
    gaugo.ContextDocs(
        gaugo.Document{ID: "refunds.md", Text: "Refunds are available for 30 days after purchase."},
    ),
    gaugo.ExpectedContains("30 days"),
)

suite.Assert(ctx, yourGaugoEvaluation,
    gaugo.Faithfulness(gaugo.WithThreshold(0.95)),
    gaugo.AnswerRelevancy(),
)
```

## No vacuous runs

Gaugo rejects runs with no effective checks. At least one of these must be true:

- At least one case has `ExpectedContains`.
- At least one non-nil metric is passed to `Assert` or `Run`.

This prevents test suites that pass without evaluating anything.

## Tips

- Start with `ExpectedContains` for cheap smoke coverage.
- Add `Faithfulness` before shipping RAG changes.
- Add `AnswerRelevancy` for user-facing answer quality.
- Use stricter thresholds in CI than in exploratory local runs.
- Keep metric detail bytes bounded with `WithMetricDetailsLimit`; see [Configuration](configuration.md).
