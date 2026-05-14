# Generation Quality Metrics

Three metrics that evaluate the structural and communicative quality of
generated answers independent of factual correctness. All three require a
configured `Judge`.

Scores use `1` as highest quality.

## Coherence

Evaluates logical structure, natural flow, and internal consistency.

```go
gaugo.Coherence()
```

**Required input:** `Question`, `Answer`.

**Scoring:** The judge returns a score in `[0, 1]` and a list of specific
issues found. Rewards answers that are easy to follow and internally
consistent. Penalizes contradictions, abrupt topic jumps, confusing ordering,
broken references, and disorganized structure.

**Judge output schema:**

```json
{
  "score": 0.0,
  "issues": [
    { "text": "string", "reason": "string" }
  ],
  "reason": "string"
}
```

## Conciseness

Evaluates whether the answer avoids unnecessary verbosity while preserving
essential information.

```go
gaugo.Conciseness()
```

**Required input:** `Question`, `Answer`.

**Scoring:** The judge returns a score in `[0, 1]` and a list of specific
issues found. Rewards compact answers that preserve necessary information.
Penalizes duplicated ideas, rambling, irrelevant expansions, boilerplate, and
excessive hedging.

**Judge output schema:** Same structure as `Coherence`.

## Completeness

Evaluates whether the answer covers all important aspects of the question.

```go
gaugo.Completeness()
```

**Required input:** `Question`, `Answer`. `ContextDocs` used as additional
signal when present.

**Scoring:** The judge returns a score in `[0, 1]` and a list of missing
aspects. Rewards answers that address the full request without material
omissions.

**Judge output schema:**

```json
{
  "score": 0.0,
  "missing": [
    { "aspect": "string", "reason": "string" }
  ],
  "reason": "string"
}
```

**Details structure:**

```json
{
  "score": 0.6,
  "missing": [
    { "aspect": "Return shipping cost", "reason": "Question asks about full refund process but shipping is not mentioned" }
  ],
  "reason": "Covers main refund steps but omits shipping details"
}
```

## Trade-offs

`Conciseness` and `Completeness` pull in opposite directions. A perfectly
concise answer may omit relevant details; a perfectly complete answer may be
verbose. Set thresholds that reflect the product requirement. For user-facing
chat, favor conciseness. For knowledge base answers, favor completeness.

## Example

```go
suite.Assert(ctx, yourFunc,
    gaugo.Coherence(),
    gaugo.Conciseness(gaugo.WithThreshold(0.6)),
    gaugo.Completeness(gaugo.WithThreshold(0.8)),
)
```
