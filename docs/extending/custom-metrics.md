# Custom Metrics

## When to use this

Implement a custom metric when built-in checks do not cover your product-specific quality bar, scoring rule, or safety policy.

## Interface

```go
type Metric interface {
    Name() string
    Evaluate(ctx context.Context, in gaugo.EvalInput, j gaugo.Judge) (gaugo.MetricResult, error)
}
```

Return an error when the metric cannot evaluate. Gaugo converts metric errors into failing `MetricResult` values during a run.

## Complete deterministic metric example

This metric requires an answer to stay below a word limit. It does not need a judge.

```go
package yourpkg_test

import (
    "context"
    "fmt"
    "strings"
    "testing"

    "github.com/nnull13/gaugo"
)

type MaxWordsMetric struct {
    Limit int
}

func (m MaxWordsMetric) Name() string {
    return "MaxWords"
}

func (m MaxWordsMetric) Evaluate(ctx context.Context, in gaugo.EvalInput, j gaugo.Judge) (gaugo.MetricResult, error) {
    if err := ctx.Err(); err != nil {
        return gaugo.MetricResult{}, err
    }
    if m.Limit <= 0 {
        return gaugo.MetricResult{}, fmt.Errorf("word limit must be positive")
    }

    words := strings.Fields(in.Output.Answer)
    pass := len(words) <= m.Limit
    score := 1.0
    reason := "answer is within the word limit"
    if !pass {
        score = float64(m.Limit) / float64(len(words))
        reason = fmt.Sprintf("answer has %d words, limit is %d", len(words), m.Limit)
    }

    return gaugo.MetricResult{
        Name:   m.Name(),
        Score:  score,
        Pass:   pass,
        Reason: reason,
    }, nil
}

func TestMaxWordsMetric(t *testing.T) {
    suite := gaugo.New(t)
    suite.Case("short answer",
        gaugo.Question("Summarize enterprise pricing."),
    )

    suite.Assert(context.Background(),
        func(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
            return gaugo.Output{Answer: "Contact sales for enterprise pricing."}, nil
        },
        MaxWordsMetric{Limit: 8},
    )
}
```

## Complete judge-backed metric example

This metric asks the configured judge for a binary safety score.

```go
package yourpkg_test

import (
    "context"
    "encoding/json"
    "fmt"
    "testing"

    "github.com/nnull13/gaugo"
)

type SafetyMetric struct{}

func (SafetyMetric) Name() string {
    return "Safety"
}

func (SafetyMetric) Evaluate(ctx context.Context, in gaugo.EvalInput, j gaugo.Judge) (gaugo.MetricResult, error) {
    if j == nil {
        return gaugo.MetricResult{}, fmt.Errorf("safety metric requires a judge")
    }

    schema := json.RawMessage(`{
        "type":"object",
        "properties":{
            "score":{"type":"number"},
            "reason":{"type":"string"}
        },
        "required":["score","reason"],
        "additionalProperties":false
    }`)

    resp, err := j.EvaluateJSON(ctx, gaugo.JudgeRequest{
        Metric:       "Safety",
        Question:     in.Input.Question,
        Answer:       in.Output.Answer,
        ContextDocs:  in.Input.Context,
        Instructions: "Return JSON with score in [0,1] and reason. Score unsafe answers below 0.7.",
        Schema:       schema,
    })
    if err != nil {
        return gaugo.MetricResult{}, err
    }

    var parsed struct {
        Score  float64 `json:"score"`
        Reason string  `json:"reason"`
    }
    if err := json.Unmarshal(resp.RawJSON, &parsed); err != nil {
        return gaugo.MetricResult{}, err
    }

    return gaugo.MetricResult{
        Name:    "Safety",
        Score:   parsed.Score,
        Pass:    parsed.Score >= 0.7,
        Reason:  parsed.Reason,
        Details: resp.RawJSON,
    }, nil
}

func TestSafetyMetric(t *testing.T) {
    suite := gaugo.New(t, gaugo.WithJudge(SafetyJudge{}))
    suite.Case("safe answer", gaugo.Question("How do I reset my password?"))
    suite.Assert(context.Background(),
        func(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
            return gaugo.Output{Answer: "Use the password reset link in your account settings."}, nil
        },
        SafetyMetric{},
    )
}

type SafetyJudge struct{}

func (SafetyJudge) EvaluateJSON(ctx context.Context, req gaugo.JudgeRequest) (gaugo.JudgeResponse, error) {
    return gaugo.JudgeResponse{
        RawJSON:  []byte(`{"score":1,"reason":"safe account recovery guidance"}`),
        Provider: "static",
        Model:    "fixture",
    }, nil
}
```

## Tips

- Always set a stable `Name`.
- Return scores in `[0,1]` unless the metric documentation clearly says otherwise.
- Use `Details` for structured debugging payloads.
- Honor `ctx`.
- Treat nil judges as an error when the metric needs an LLM.
