# Specialized Metrics

Four metrics for domain-specific evaluation and custom criteria. All require a
configured `Judge`.

## CitationAccuracy

Verifies that citations or references in the answer point to the correct
context documents.

```go
gaugo.CitationAccuracy()
```

**Required input:** `Question`, `Answer`, `ContextDocs`.

**Scoring:** The judge identifies every citation-like reference in the answer,
maps it to a document ID, and marks it `accurate` when the cited document
supports the statement. The score is `accurate_count / total_citations`. Zero
citations scores `0`.

**Judge output schema:**

```json
{
  "citations": [
    { "text": "string", "doc_id": "string", "accurate": true, "reason": "string" }
  ],
  "reason": "string"
}
```

Use this for systems that generate answers with source attribution (footnotes,
inline references, document links).

## SummarizationQuality

Evaluates summaries across three axes: coverage, fidelity, and conciseness.

```go
gaugo.SummarizationQuality()
```

**Required input:** `Question`, `Answer`, `ContextDocs` (the context is the
source material being summarized).

**Scoring:** The judge returns three independent sub-scores in `[0, 1]`:

| Sub-score | Measures |
| --- | --- |
| `coverage_score` | Whether important source information is included |
| `fidelity_score` | Whether the summary is faithful and avoids unsupported additions |
| `conciseness_score` | Whether the summary avoids repetition and unnecessary detail |

Final score = `(coverage + fidelity + conciseness) / 3`.

**Judge output schema:**

```json
{
  "coverage_score": 0.0,
  "fidelity_score": 0.0,
  "conciseness_score": 0.0,
  "reason": "string"
}
```

The three sub-scores appear in `Details`, making it possible to diagnose which
axis of summarization quality is weakest.

## InstructionAdherence

Checks whether the answer follows a set of explicit instructions.

```go
gaugo.InstructionAdherence()
```

**Required input:** `Question`, `Answer`, `ExpectedInstructions`.

Fails with `"instruction adherence metric requires Expected.Instructions"` if
instructions are missing.

**Scoring:** The judge breaks the instructions into checkable requirements and
marks each `followed` or not. It evaluates compliance only, not factual
correctness (unless the instructions require it). The score is
`followed_count / total_instructions`. Zero instructions scores `0`.

**Judge output schema:**

```json
{
  "instructions": [
    { "text": "string", "followed": true, "reason": "string" }
  ],
  "reason": "string"
}
```

**Details structure:**

```json
{
  "instructions": [
    { "text": "Return valid JSON", "followed": true, "reason": "Output is valid JSON" },
    { "text": "No prose before or after", "followed": false, "reason": "Output begins with explanatory text" }
  ],
  "reason": "Follows JSON format but includes preamble"
}
```

## GEval

General-purpose evaluation with user-defined criteria. Use it when no built-in
metric matches your evaluation needs.

```go
gaugo.GEval("Prefer answers that are directly actionable and include next steps.")
gaugo.GEval("Penalize answers that use jargon without defining it.", gaugo.WithThreshold(0.8))
```

**Required input:** `Question`, `Answer`. The criteria string is the first
argument to the constructor and must be non-empty.

Fails with `"g-eval metric requires non-empty criteria"` if criteria is empty
or whitespace.

**Scoring:** The judge evaluates the answer against the provided criteria and
returns a score in `[0, 1]`.

**Judge output schema:**

```json
{
  "score": 0.0,
  "reason": "string",
  "issues": ["string"]
}
```

`GEval` is the escape hatch for evaluation dimensions that Gaugo does not cover
with a built-in metric. Criteria should be specific and testable. Vague
criteria like "be good" produce inconsistent scores.

## Example

```go
suite.Case("article summary",
    gaugo.Question("Summarize the key findings"),
    gaugo.ContextDocs(
        gaugo.Document{ID: "paper.md", Text: "The study found a 40% reduction in latency..."},
    ),
    gaugo.ExpectedInstructions("Return 3 bullet points. No introductory sentence."),
)

suite.Assert(ctx, yourFunc,
    gaugo.SummarizationQuality(),
    gaugo.CitationAccuracy(),
    gaugo.InstructionAdherence(gaugo.WithThreshold(0.9)),
    gaugo.GEval("Each bullet point should start with a quantitative finding."),
)
```
