# RAG Metrics

Four metrics designed to evaluate retrieval-augmented generation pipelines. They
separate retrieval quality from generation quality so failures point to the
right part of the system.

All four require a configured `Judge`.

## ContextRelevancy

Scores whether the retrieved context documents are useful for answering the
question. Ignores the generated answer entirely.

```go
gaugo.ContextRelevancy()
gaugo.ContextRelevancy(gaugo.WithThreshold(0.75))
```

**Required input:** `Question`, `ContextDocs`.

**Scoring:** The judge scores each non-empty context document independently
against the question on a `[0, 1]` scale. The final score is the arithmetic
mean of all document scores. An empty context scores `0`.

**Judge output schema:**

```json
{
  "documents": [
    { "id": "string", "score": 0.0, "reason": "string" }
  ],
  "reason": "string"
}
```

**Details structure:**

```json
{
  "documents": [
    { "id": "pricing.md", "score": 0.9, "reason": "Directly answers pricing question" }
  ],
  "reason": "Both documents are relevant to the query"
}
```

## ContextPrecision

Scores the fraction of retrieved documents that are actually useful. Targets
retrieval noise: too many irrelevant or redundant documents in the context
window.

```go
gaugo.ContextPrecision()
```

**Required input:** `Question`, `ContextDocs`.

**Scoring:** The judge marks each document as `useful` or not. The score is
`useful_count / total_documents`. Zero documents scores `0`.

**Judge output schema:**

```json
{
  "documents": [
    { "id": "string", "useful": true, "reason": "string" }
  ],
  "reason": "string"
}
```

**Details structure:**

```json
{
  "documents": [
    { "id": "pricing.md", "useful": true, "reason": "Contains enterprise pricing details" },
    { "id": "changelog.md", "useful": false, "reason": "Version history unrelated to pricing" }
  ],
  "reason": "One of two documents is relevant"
}
```

## ContextRecall

Scores whether the retrieved context covers the claims in the expected answer.
Requires ground truth.

```go
gaugo.ContextRecall()
```

**Required input:** `Question`, `ContextDocs`, `ExpectedAnswer`.

Fails with `"context recall metric requires Expected.Answer"` if ground truth
is missing.

**Scoring:** The judge breaks the expected answer into atomic claims and marks
each as `attributed` when at least one context document supports it. The score
is `attributed_claims / total_claims`. Zero claims scores `0`.

**Judge output schema:**

```json
{
  "claims": [
    { "text": "string", "attributed": true, "evidence": ["doc_id"] }
  ],
  "reason": "string"
}
```

**Details structure:**

```json
{
  "claims": [
    { "text": "Refunds available for 30 days", "attributed": true, "evidence": ["refunds.md"] },
    { "text": "Must contact support", "attributed": false, "evidence": [] }
  ],
  "reason": "Context covers duration but not the support requirement"
}
```

## Faithfulness

Scores whether the generated answer is supported by the context documents.
Targets hallucination at the generation layer: the model inventing, distorting,
or contradicting the evidence it received.

```go
gaugo.Faithfulness()
gaugo.Faithfulness(gaugo.WithThreshold(0.9))
```

**Required input:** `Question`, `Answer`, `ContextDocs` (optional but needed
for meaningful evaluation).

**Scoring:** The judge breaks the answer into atomic factual claims. Each claim
is marked `supported` only when the context directly entails it. The score is
`supported_claims / total_claims`. Zero claims scores `0`.

**Judge output schema:**

```json
{
  "claims": [
    { "text": "string", "supported": true, "evidence": ["doc_id"] }
  ],
  "reason": "string"
}
```

**Details structure:**

```json
{
  "claims": [
    { "text": "Pricing is custom", "supported": true, "evidence": ["pricing.md"] },
    { "text": "Starts at $500/month", "supported": false, "evidence": [] }
  ],
  "reason": "One claim fabricated pricing details not in context"
}
```

## Diagnostic patterns

Use multiple RAG metrics together to isolate the failure layer:

| Pattern | Likely cause |
| --- | --- |
| Low `ContextRelevancy` | Retriever returning irrelevant documents. Check embeddings, chunking, or filters. |
| Low `ContextPrecision`, normal `ContextRelevancy` | Retriever returning too many documents, diluting signal with noise. Reduce top-k or improve re-ranking. |
| Low `ContextRecall` | Retriever missing evidence needed for the answer. Expand corpus, adjust chunking, or relax filters. |
| High retrieval scores, low `Faithfulness` | Generator ignored or distorted good evidence. Prompt engineering or model issue. |
| High `Faithfulness`, low `AnswerRelevancy` | Answer is grounded but does not address the question. Generation is correct but off-topic. |

## Example

```go
func TestRAGPipeline(t *testing.T) {
    judge, err := openai.New(openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4.1-mini",
    })
    if err != nil {
        t.Fatalf("judge: %v", err)
    }

    suite := gaugo.New(t,
        gaugo.WithJudge(judge),
        gaugo.WithCaseTimeout(15*time.Second),
    )

    suite.Case("refund window",
        gaugo.Question("How long do I have to request a refund?"),
        gaugo.ContextDocs(
            gaugo.Document{ID: "refunds.md", Text: "Refunds are available within 30 days of purchase."},
        ),
        gaugo.ExpectedAnswer("Refunds are available within 30 days of purchase."),
    )

    suite.Assert(context.Background(), yourRAGFunc,
        gaugo.ContextRelevancy(gaugo.WithThreshold(0.75)),
        gaugo.ContextPrecision(),
        gaugo.ContextRecall(),
        gaugo.Faithfulness(gaugo.WithThreshold(0.9)),
    )
}
```
