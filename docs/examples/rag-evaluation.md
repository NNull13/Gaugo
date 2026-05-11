# RAG Evaluation

## When to use this

Use this example when evaluating a retrieval-augmented generation system with context documents, deterministic checks, and LLM-judged quality metrics.

## Complete test

```go
package yourpkg_test

import (
    "context"
    "os"
    "strings"
    "testing"
    "time"

    "github.com/nnull13/gaugo"
    "github.com/nnull13/gaugo/provider/openai"
)

func TestSupportRAG(t *testing.T) {
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
        gaugo.WithParallelism(8),
        gaugo.WithCaseTimeout(15*time.Second),
    )

    suite.Case("enterprise pricing",
        gaugo.Question("How does enterprise pricing work?"),
        gaugo.ContextDocs(
            gaugo.Document{ID: "pricing.md", Text: "Enterprise pricing is custom and requires contacting sales."},
        ),
        gaugo.ExpectedContains("sales"),
    )

    suite.Case("refund window",
        gaugo.Question("How long do I have to request a refund?"),
        gaugo.ContextDocs(
            gaugo.Document{ID: "refunds.md", Text: "Refunds are available within 30 days of purchase."},
        ),
        gaugo.ExpectedContains("30 days"),
    )

    suite.Assert(context.Background(),
        func(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
            answer, err := answerWithRAG(ctx, in.Question, in.Context)
            if err != nil {
                return gaugo.Output{}, err
            }
            return gaugo.Output{Answer: answer}, nil
        },
        gaugo.ContextRelevancy(gaugo.WithThreshold(0.75)),
        gaugo.Faithfulness(gaugo.WithThreshold(0.9)),
        gaugo.AnswerRelevancy(gaugo.WithThreshold(0.8)),
    )
}

func answerWithRAG(ctx context.Context, question string, docs []gaugo.Document) (string, error) {
    for _, doc := range docs {
        switch {
        case strings.Contains(doc.ID, "pricing"):
            return "Enterprise pricing is custom. Contact sales for a quote.", nil
        case strings.Contains(doc.ID, "refunds"):
            return "Refunds are available within 30 days of purchase.", nil
        }
    }
    return "", nil
}
```

## Why this shape works

- `ContextDocs` stores the retrieved evidence for each case.
- `ExpectedContains` catches hard product requirements cheaply.
- `ContextRelevancy` checks whether the retrieved evidence is useful for the question.
- `Faithfulness` checks whether the answer is supported by context.
- `AnswerRelevancy` checks whether the answer addresses the user question.
- `WithCaseTimeout` prevents one slow provider call from hanging the suite.

## Reading RAG failures

The RAG triad helps locate the broken part of the pipeline:

| Pattern | Likely issue |
| --- | --- |
| Low `ContextRelevancy` | Retriever, corpus coverage, chunking, or filters. |
| High `ContextRelevancy`, low `Faithfulness` | The generator ignored or distorted good evidence. |
| High `Faithfulness`, low `AnswerRelevancy` | The answer stayed grounded but did not satisfy the user task. |

Operational failures are different from quality failures. If your `RunFunc`
returns an error, Gaugo records it in `CaseResult.RunError`. If a judge request,
timeout, or parse failure prevents a metric from evaluating, the metric fails
with classified error details. A low metric score without classified error
details is a quality failure.

## Production tips

- Keep context short enough for the judge to evaluate reliably.
- Include document IDs that match your source system.
- Use `t.Skip` when provider credentials are missing in local development.
- Set thresholds explicitly once you understand baseline behavior.
- Start with a small high-signal suite before adding broad coverage.
- For large Anthropic-backed suites, start with low `WithParallelism`, bounded retries, and a case timeout that covers all metric calls.

For CI guidance, see [CI Integration](../guides/ci-integration.md). For provider setup, see [Provider](../provider/index.md).
