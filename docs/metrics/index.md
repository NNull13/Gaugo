# Metrics

Gaugo ships 24 evaluation checks across eight categories. Each check implements
the `Metric` interface or, in the case of `ExpectedContains`, runs as a
case-level assertion.

```go
type Metric interface {
    Name() string
    Evaluate(ctx context.Context, in gaugo.EvalInput, j gaugo.Judge) (gaugo.MetricResult, error)
}
```

## Catalog

| Category | Metrics | Type | Page |
| --- | --- | --- | --- |
| RAG | `ContextRelevancy`, `ContextPrecision`, `ContextRecall`, `Faithfulness` | LLM | [rag.md](rag.md) |
| Answer | `AnswerRelevancy`, `AnswerCorrectness`, `AnswerSimilarity` | LLM / Det | [answer.md](answer.md) |
| Safety | `Hallucination`, `Toxicity`, `Bias` | LLM | [safety.md](safety.md) |
| Generation quality | `Coherence`, `Conciseness`, `Completeness` | LLM | [generation-quality.md](generation-quality.md) |
| Structured output | `JSONValidity`, `SchemaCompliance`, `ExpectedJSON` | Det | [structured-output.md](structured-output.md) |
| Specialized | `CitationAccuracy`, `SummarizationQuality`, `InstructionAdherence`, `GEval` | LLM | [specialized.md](specialized.md) |
| Deterministic contracts | `ExpectedContains`, `ExpectedRegex`, `AnswerLength`, `Latency` | Det | [deterministic.md](deterministic.md) |

**Det** = deterministic, no network, no judge. **LLM** = requires a configured `Judge`.

## Deterministic vs LLM-judged

Deterministic metrics run in-process with no external calls. They produce hard
`0`/`1` scores or simple fractions and are suitable for every build.

LLM-judged metrics send a structured prompt and a JSON schema to the configured
`Judge`. The judge returns strict JSON that Gaugo parses into a typed output.
Scores are either computed from the parsed structure (claim ratios, document
averages) or taken directly from the judge response.

## EvalInput

Every metric receives the same evaluation context:

```go
type EvalInput struct {
    CaseName string
    Input    Input           // Question + Context
    Output   Output          // Answer from RunFunc
    Expected Expected        // Ground truth, instructions, contains checks
    Elapsed  time.Duration   // RunFunc wall time
}
```

| Field | Set by |
| --- | --- |
| `Input.Question` | `gaugo.Question(...)` |
| `Input.Context` | `gaugo.ContextDocs(...)` |
| `Output.Answer` | Return value of `RunFunc` |
| `Expected.Answer` | `gaugo.ExpectedAnswer(...)` |
| `Expected.Instructions` | `gaugo.ExpectedInstructions(...)` |
| `Expected.Contains` | `gaugo.ExpectedContains(...)` |
| `Elapsed` | Measured by the runner |

Metrics that require a field not present in the case fail with a descriptive
error rather than silently returning zero.

## Thresholds

All metrics default to a pass threshold of **0.7**. Override per metric:

```go
gaugo.Faithfulness(gaugo.WithThreshold(0.9))
```

Rules:
- Must be a finite number in `[0, 1]`.
- `NaN`, `+Inf`, `-Inf`, and out-of-range values produce a failing `MetricResult`.
- Deterministic metrics that produce hard `0`/`1` scores pass only on `1`
  regardless of threshold.

## Metric options

Options are passed as variadic arguments to metric constructors.

| Option | Applies to | Purpose |
| --- | --- | --- |
| `WithThreshold(float64)` | All metrics | Override pass/fail threshold |
| `WithSchema(json.RawMessage)` | `SchemaCompliance` | JSON Schema to validate against |
| `WithExpectedFields(map[string]any)` | `ExpectedJSON` | Expected field values with dot-path lookup |
| `WithMaxLatency(time.Duration)` | `Latency` | Maximum allowed wall time |
| `WithMinLength(int)` | `AnswerLength` | Minimum rune count |
| `WithMaxLength(int)` | `AnswerLength` | Maximum rune count |

The `metric` sub-package exports the same constructors and option functions as
the root `gaugo` package. Both forms are equivalent:

```go
gaugo.Faithfulness(gaugo.WithThreshold(0.8))
metric.Faithfulness(metric.WithThreshold(0.8))
```

## MetricResult

Every metric evaluation produces a `MetricResult`:

```go
type MetricResult struct {
    Name         string
    Score        float64       // [0, 1]
    Pass         bool          // Score >= threshold
    Reason       string        // Human-readable explanation
    Details      []byte        // JSON with metric-specific structure
    Provider     string        // Judge provider (LLM metrics only)
    Model        string        // Judge model (LLM metrics only)
    RequestID    string        // Judge request ID
    JudgeLatency time.Duration // Judge round-trip time
}
```

`Details` contains a JSON object whose schema varies per metric. Each metric
page documents its detail structure.

## No vacuous runs

Gaugo rejects runs with no effective checks. At least one of these must hold:

- Every case has at least one `ExpectedContains`.
- At least one non-nil `Metric` is passed to `Assert` or `Run`.
