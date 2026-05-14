# Metrics

## When to use this

Use this page as a quick reference for the metric catalog, thresholds, and
options. For detailed documentation on each metric including scoring algorithms,
judge schemas, and diagnostic patterns, see the
[metrics section](../metrics/index.md).

For custom metric implementations, see
[Custom Metrics](../extending/custom-metrics.md).

## Metric interface

```go
type Metric interface {
    Name() string
    Evaluate(ctx context.Context, in gaugo.EvalInput, j gaugo.Judge) (gaugo.MetricResult, error)
}
```

Metrics run after your `RunFunc` returns an `Output`. Built-in LLM metrics call the configured `Judge` and parse strictly structured JSON. Deterministic metrics use only the Go standard library.

The public metric package lives in `github.com/nnull13/gaugo/metric`. The root
`gaugo` package re-exports the same constructors and types, so both forms are
equivalent:

```go
gaugo.Faithfulness(gaugo.WithThreshold(0.8))
metric.Faithfulness(metric.WithThreshold(0.8))
```

## Built-in metric catalog

| Category | Metrics |
| --- | --- |
| RAG | `ContextRelevancy`, `ContextPrecision`, `ContextRecall`, `Faithfulness` |
| Answer quality | `AnswerRelevancy`, `AnswerCorrectness`, `AnswerSimilarity` |
| Safety | `Hallucination`, `Toxicity`, `Bias` |
| Generation quality | `Coherence`, `Conciseness`, `Completeness` |
| Structured output | `JSONValidity`, `SchemaCompliance`, `ExpectedJSON` |
| Instructions and custom | `InstructionAdherence`, `GEval` |
| Domain-specific | `CitationAccuracy`, `SummarizationQuality` |
| Deterministic contracts | `ExpectedContains`, `Latency`, `AnswerLength`, `ExpectedRegex` |

Metrics that compare against ground truth use `ExpectedAnswer(...)`. `InstructionAdherence` uses `ExpectedInstructions(...)`.

## RAG quality

For retrieval-augmented generation, use multiple metrics to isolate retrieval
problems from generation problems:

| Metric | What it checks | Typical failure it isolates |
| --- | --- | --- |
| `ContextRelevancy` | Are the retrieved context documents useful for the question? | Retrieval returned irrelevant, noisy, or incomplete evidence. |
| `ContextPrecision` | Of the retrieved documents, how many are actually useful? | Retriever returned too much irrelevant or redundant context. |
| `ContextRecall` | Does the retrieved context cover the expected answer? | Retriever missed evidence needed for the ground truth. |
| `Faithfulness` | Is the answer supported by the context documents? | The model added unsupported claims or contradicted evidence. |
| `AnswerRelevancy` | Does the answer address the user's question? | The answer is grounded but incomplete, evasive, or off-topic. |
| `AnswerCorrectness` | Does the answer match `ExpectedAnswer` factually? | The model produced a grounded but wrong answer. |

For example, low `ContextPrecision` points at noisy retrieval, low
`ContextRecall` points at missing evidence, and high retrieval scores with low
`Faithfulness` usually points at answer generation.

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

## ContextRelevancy

```go
suite.Assert(ctx, yourGaugoEvaluation, gaugo.ContextRelevancy())
```

`ContextRelevancy` scores whether the case context documents are relevant to
the input question. It requires a configured `Judge`.

The metric asks the judge to score each non-empty context document against the
question while ignoring the generated answer. The final score is the average
document relevance score; an empty context scores `0`.

Use it for RAG systems when you want to evaluate retrieval quality separately
from answer generation. A low score means the generated answer may be poor even
if the model behaved reasonably with the evidence it received.

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

## Ground-truth metrics

`ContextRecall`, `AnswerCorrectness`, and `AnswerSimilarity` require `ExpectedAnswer`.

```go
suite.Case("refunds",
    gaugo.Question("How long are refunds available?"),
    gaugo.ContextDocs(gaugo.Doc("refunds.md", "Refunds are available for 30 days.")),
    gaugo.ExpectedAnswer("Refunds are available for 30 days."),
)

suite.Assert(ctx, yourGaugoEvaluation,
    gaugo.ContextRecall(),
    gaugo.AnswerCorrectness(),
    gaugo.AnswerSimilarity(gaugo.WithThreshold(0.6)),
)
```

`ContextRecall` is LLM-judged and checks whether the context supports each ground-truth claim. `AnswerCorrectness` is LLM-judged and compares the answer to ground truth. `AnswerSimilarity` is deterministic Jaccard similarity over normalized answer tokens.

## Safety and generation quality

```go
suite.Assert(ctx, yourGaugoEvaluation,
    gaugo.Hallucination(),
    gaugo.Toxicity(),
    gaugo.Bias(),
    gaugo.Coherence(),
    gaugo.Conciseness(),
    gaugo.Completeness(),
)
```

These metrics require a configured `Judge`. Safety scores use `1` as safest or least problematic. Quality scores use `1` as highest quality.

## Structured output

```go
suite.Assert(ctx, yourGaugoEvaluation,
    gaugo.JSONValidity(),
    gaugo.SchemaCompliance(gaugo.WithSchema(json.RawMessage(`{
        "type":"object",
        "required":["status"],
        "properties":{"status":{"type":"string"}}
    }`))),
    gaugo.ExpectedJSON(gaugo.WithExpectedFields(map[string]any{
        "status": "ok",
    })),
)
```

`JSONValidity`, `SchemaCompliance`, and `ExpectedJSON` are deterministic and do not require a judge. `SchemaCompliance` supports a basic JSON Schema subset: object properties, required fields, primitive types, arrays, nested structures, and `additionalProperties:false`.

## Instructions and custom criteria

```go
suite.Case("format",
    gaugo.Question("Return a compact answer"),
    gaugo.ExpectedInstructions("Return valid JSON with no prose."),
)

suite.Assert(ctx, yourGaugoEvaluation,
    gaugo.InstructionAdherence(),
    gaugo.GEval("Prefer answers that are directly actionable."),
)
```

`InstructionAdherence` requires `ExpectedInstructions`. `GEval` accepts custom criteria and uses the configured judge.

## Domain-specific and deterministic contracts

```go
suite.Assert(ctx, yourGaugoEvaluation,
    gaugo.CitationAccuracy(),
    gaugo.SummarizationQuality(),
    gaugo.Latency(gaugo.WithMaxLatency(500*time.Millisecond)),
    gaugo.AnswerLength(gaugo.WithMinLength(20), gaugo.WithMaxLength(600)),
    gaugo.ExpectedRegex(`(?i)contact sales`),
)
```

`CitationAccuracy` and `SummarizationQuality` require a judge. `Latency`, `AnswerLength`, and `ExpectedRegex` are deterministic.

## Thresholds

Built-in metrics default to a pass threshold of `0.7` unless a deterministic metric produces a hard `0` or `1` score. Override with `WithThreshold`.

```go
suite.Assert(ctx, yourGaugoEvaluation,
    gaugo.ContextRelevancy(gaugo.WithThreshold(0.75)),
    gaugo.Faithfulness(gaugo.WithThreshold(0.9)),
    gaugo.AnswerRelevancy(gaugo.WithThreshold(0.8)),
)
```

Thresholds must be finite numbers in `[0,1]`. `NaN`, `+Inf`, `-Inf`, and out-of-range values produce an invalid metric; when used through a runner, that metric becomes a failing `MetricResult`.

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
    gaugo.ContextRelevancy(),
    gaugo.Faithfulness(gaugo.WithThreshold(0.95)),
    gaugo.AnswerRelevancy(),
)
```

## No vacuous runs

Gaugo rejects runs with no effective checks. At least one of these must be true:

- Every case has `ExpectedContains`.
- At least one non-nil metric is passed to `Assert` or `Run`.

This prevents test suites that pass without evaluating anything.

## Tips

- Start with `ExpectedContains` for cheap smoke coverage.
- Add `ContextRelevancy` when retrieval quality is part of the release risk.
- Add `Faithfulness` before shipping RAG changes.
- Add `AnswerRelevancy` for user-facing answer quality.
- Use stricter thresholds in CI than in exploratory local runs.
- Keep metric detail bytes bounded with `WithMetricDetailsLimit`; see [Configuration](configuration.md).
