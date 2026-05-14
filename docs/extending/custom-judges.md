# Custom Judges

## When to use this

Implement a custom judge when you want Gaugo metrics to call an internal model gateway, a recorded fixture, a deterministic fake, or a provider that is not bundled with Gaugo.

For built-in providers, see [Provider](../provider/index.md).

## Interface

```go
type Judge interface {
    EvaluateJSON(ctx context.Context, req gaugo.JudgeRequest) (gaugo.JudgeResponse, error)
}
```

The judge must return JSON that matches the metric schema in `req.Schema`.

## Complete fake judge example

Copy this into a test file when you want deterministic LLM metric tests without network calls.

```go
package yourpkg_test

import (
    "context"
    "fmt"
    "testing"

    "github.com/nnull13/gaugo"
)

type staticJudge struct{}

func (staticJudge) EvaluateJSON(ctx context.Context, req gaugo.JudgeRequest) (gaugo.JudgeResponse, error) {
    switch req.Metric {
    case "Faithfulness":
        return gaugo.JudgeResponse{
            RawJSON:   []byte(`{"claims":[{"text":"answer is supported","supported":true}],"reason":"supported by fixture"}`),
            Provider:  "static",
            Model:     "fixture",
            RequestID: "fixture-faithfulness",
        }, nil
    case "AnswerRelevancy":
        return gaugo.JudgeResponse{
            RawJSON:   []byte(`{"score":1,"reason":"answers the question","issues":[]}`),
            Provider:  "static",
            Model:     "fixture",
            RequestID: "fixture-answer-relevancy",
        }, nil
    default:
        return gaugo.JudgeResponse{}, fmt.Errorf("unsupported metric %q", req.Metric)
    }
}

func TestWithStaticJudge(t *testing.T) {
    suite := gaugo.New(t, gaugo.WithJudge(staticJudge{}))
    suite.Case("pricing",
        gaugo.Question("How does enterprise pricing work?"),
        gaugo.ContextDocs(
            gaugo.Document{ID: "pricing.md", Text: "Enterprise pricing is custom and requires sales."},
        ),
    )

    suite.Assert(context.Background(),
        func(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
            return gaugo.Output{Answer: "Enterprise pricing is custom and requires sales."}, nil
        },
        gaugo.Faithfulness(),
        gaugo.AnswerRelevancy(),
    )
}
```

## Gateway judge sketch

Use this shape for internal HTTP gateways. The response body must be the raw structured JSON expected by the metric.

```go
type GatewayJudge struct {
    Endpoint string
    Client   *http.Client
}

func (j GatewayJudge) EvaluateJSON(ctx context.Context, req gaugo.JudgeRequest) (gaugo.JudgeResponse, error) {
    client := j.Client
    if client == nil {
        client = http.DefaultClient
    }

    body, err := json.Marshal(req)
    if err != nil {
        return gaugo.JudgeResponse{}, err
    }

    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, j.Endpoint, bytes.NewReader(body))
    if err != nil {
        return gaugo.JudgeResponse{}, err
    }
    httpReq.Header.Set("Content-Type", "application/json")

    start := time.Now()
    resp, err := client.Do(httpReq)
    if err != nil {
        return gaugo.JudgeResponse{}, err
    }
    defer resp.Body.Close()

    raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
    if err != nil {
        return gaugo.JudgeResponse{}, err
    }
    if resp.StatusCode >= http.StatusBadRequest {
        return gaugo.JudgeResponse{}, fmt.Errorf("gateway status %d", resp.StatusCode)
    }

    return gaugo.JudgeResponse{
        RawJSON:   raw,
        Provider:  "gateway",
        Model:     "internal",
        RequestID: resp.Header.Get("x-request-id"),
        Latency:   time.Since(start),
    }, nil
}
```

## Requirements

- Honor `ctx`; Gaugo uses it for case timeouts and cancellation.
- Return raw JSON only in `JudgeResponse.RawJSON`.
- Populate `Provider`, `Model`, `RequestID`, and `Latency` when known; Gaugo copies them into `MetricResult`.
- Keep provider errors concise and safe for test logs.
- Do not include secrets in errors.
- Use `req.Schema` if your gateway supports structured output.
- Keep temperature low or deterministic for CI.
