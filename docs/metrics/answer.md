# Answer Metrics

Three metrics that evaluate the generated answer against the question and,
optionally, a ground-truth reference.

## AnswerRelevancy

Scores whether the answer addresses the user's question. Requires a `Judge`.

```go
gaugo.AnswerRelevancy()
gaugo.AnswerRelevancy(gaugo.WithThreshold(0.8))
```

**Required input:** `Question`, `Answer`.

**Scoring:** The judge returns a single score in `[0, 1]` based on directness,
completeness, specificity, and usefulness. Penalizes omissions, generic filler,
evasion, contradictions, unsupported refusals, and unrelated details.

**Judge output schema:**

```json
{
  "score": 0.0,
  "reason": "string",
  "issues": ["string"]
}
```

`issues` is optional and lists specific problems found (evasion, off-topic
content, missing aspects). When present, it appears in `Details`.

## AnswerCorrectness

Compares the answer against a ground-truth expected answer. Requires a `Judge`
and `ExpectedAnswer`.

```go
gaugo.AnswerCorrectness()
```

**Required input:** `Question`, `Answer`, `ExpectedAnswer`.

Fails with `"answer correctness metric requires Expected.Answer"` if ground
truth is missing.

**Scoring:** The judge breaks the answer into factual statements, marks each
`correct` against the expected answer, and returns an overall score. The score
reflects factual alignment: contradictions, omissions, and invented details
lower it.

**Judge output schema:**

```json
{
  "statements": [
    { "text": "string", "correct": true, "reason": "string" }
  ],
  "score": 0.0,
  "reason": "string"
}
```

**Details structure:**

```json
{
  "statements": [
    { "text": "Refunds last 30 days", "correct": true, "reason": "Matches expected answer" },
    { "text": "No exceptions apply", "correct": false, "reason": "Expected answer does not mention exceptions" }
  ],
  "score": 0.65,
  "reason": "Mostly correct but adds unsupported claim about exceptions"
}
```

## AnswerSimilarity

Deterministic token overlap between the answer and expected answer. Does not
require a `Judge`.

```go
gaugo.AnswerSimilarity()
gaugo.AnswerSimilarity(gaugo.WithThreshold(0.6))
```

**Required input:** `Answer`, `ExpectedAnswer`.

**Scoring:** Jaccard similarity over normalized tokens. Both strings are
lowercased and split on non-alphanumeric boundaries. The score is
`|intersection| / |union|`.

**Details structure:**

```json
{
  "intersection_tokens": 8,
  "union_tokens": 12
}
```

Use `AnswerSimilarity` as a fast smoke check. For semantic comparison, use
`AnswerCorrectness`.

## Choosing between them

| Situation | Metric |
| --- | --- |
| No ground truth available | `AnswerRelevancy` |
| Ground truth available, need semantic comparison | `AnswerCorrectness` |
| Ground truth available, need fast deterministic check | `AnswerSimilarity` |
| Full answer evaluation | All three combined |

## Example

```go
suite.Case("refund policy",
    gaugo.Question("What is the refund policy?"),
    gaugo.ExpectedAnswer("Refunds are available within 30 days of purchase."),
)

suite.Assert(ctx, yourFunc,
    gaugo.AnswerRelevancy(gaugo.WithThreshold(0.8)),
    gaugo.AnswerCorrectness(),
    gaugo.AnswerSimilarity(gaugo.WithThreshold(0.5)),
)
```
