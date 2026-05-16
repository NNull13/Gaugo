# BYO LLM Judge

## When to use this

Use this pattern when you want to run Gaugo metrics against your own LLM without giving Gaugo access to an API key. You manage the HTTP call; Gaugo handles everything else (metric evaluation, scoring, reporting).

Typical scenarios:

- Your organization routes all LLM traffic through an internal gateway.
- You run an on-premises model and do not want to configure a reverse proxy.
- You prefer not to trust any third-party library with your credentials.

For cases where you're comfortable letting Gaugo handle the HTTP call, prefer the built-in [providers](../provider/index.md) instead.

## Quick start with FuncJudge

[`gaugo.FuncJudge`](https://pkg.go.dev/github.com/nnull13/gaugo#FuncJudge) lets you implement a judge with a single function literal, without defining a named type.

```go
package yourpkg_test

import (
    "context"
    "testing"

    "github.com/nnull13/gaugo"
)

func TestWithMyLLM(t *testing.T) {
    suite := gaugo.New(t, gaugo.WithJudge(gaugo.FuncJudge(
        func(ctx context.Context, req gaugo.JudgeRequest) (gaugo.JudgeResponse, error) {
            // req.Instructions — system prompt for the metric (e.g. faithfulness evaluator)
            // req.Schema       — JSON schema the response must match
            // req.Question, req.Answer, req.ContextDocs — the data to evaluate

            raw, err := myLLM.Call(ctx, req.Instructions, formatUserPrompt(req))
            if err != nil {
                return gaugo.JudgeResponse{}, err
            }
            return gaugo.JudgeResponse{RawJSON: raw}, nil
        },
    )))

    suite.Case("my case",
        gaugo.Question("What is the capital of France?"),
        gaugo.Answer("Paris"),
    )
    suite.Assert(context.Background(),
        func(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
            return gaugo.Output{Answer: "Paris"}, nil
        },
        gaugo.AnswerRelevancy(),
        gaugo.Faithfulness(),
    )
}
```

`req.Instructions` already contains the compiled system prompt for whichever metric is being evaluated. `req.Schema` contains the expected JSON schema. You do not need to construct these yourself.

## Reusing Gaugo's prompts

If you build your own evaluation pipeline outside of Gaugo's suite, import [`github.com/nnull13/gaugo/prompt`](https://pkg.go.dev/github.com/nnull13/gaugo/prompt) to get the exact same instructions and schemas that Gaugo uses internally.

```go
import "github.com/nnull13/gaugo/prompt"

systemPrompt := prompt.FaithfulnessInstructions()
schema       := prompt.FaithfulnessSchema()
```

Available functions follow the pattern `<MetricName>Instructions()` and `<MetricName>Schema()` for every built-in metric. For `GEval`, pass your criteria string:

```go
systemPrompt := prompt.GEvalInstructions("Is the answer polite and professional?")
```

## Complete gateway example

This example shows a judge that forwards evaluation to an internal HTTP gateway. The gateway is responsible for calling the LLM and returning structured JSON.

```go
package yourpkg

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/nnull13/gaugo"
    "github.com/nnull13/gaugo/prompt"
)

type GatewayJudge struct {
    Endpoint string
    Client   *http.Client
}

type gatewayRequest struct {
    System   string          `json:"system"`
    User     string          `json:"user"`
    Schema   json.RawMessage `json:"schema"`
}

func (j GatewayJudge) EvaluateJSON(ctx context.Context, req gaugo.JudgeRequest) (gaugo.JudgeResponse, error) {
    client := j.Client
    if client == nil {
        client = http.DefaultClient
    }

    body, err := json.Marshal(gatewayRequest{
        System: req.Instructions,
        User:   buildUserMessage(req),
        Schema: req.Schema,
    })
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
        return gaugo.JudgeResponse{}, fmt.Errorf("gateway status %d: %s", resp.StatusCode, raw)
    }

    return gaugo.JudgeResponse{
        RawJSON:   raw,
        Provider:  "gateway",
        Model:     resp.Header.Get("x-model"),
        RequestID: resp.Header.Get("x-request-id"),
        Latency:   time.Since(start),
    }, nil
}

func buildUserMessage(req gaugo.JudgeRequest) string {
    // Build a simple text prompt. You can format this however your gateway expects.
    var b strings.Builder
    b.WriteString("Question: ")
    b.WriteString(req.Question)
    b.WriteString("\nAnswer: ")
    b.WriteString(req.Answer)
    if len(req.ContextDocs) > 0 {
        b.WriteString("\nContext:")
        for _, d := range req.ContextDocs {
            b.WriteString("\n- [")
            b.WriteString(d.ID)
            b.WriteString("] ")
            b.WriteString(d.Text)
        }
    }
    return b.String()
}
```

Use it the same way as any other judge:

```go
suite := gaugo.New(t, gaugo.WithJudge(GatewayJudge{
    Endpoint: "https://llm.internal/evaluate",
}))
```

## What Gaugo provides; what you provide

| Gaugo handles | You handle |
|---|---|
| Metric selection and scoring | HTTP call to your LLM |
| `req.Instructions` (system prompt) | Authentication / API key |
| `req.Schema` (expected JSON shape) | Rate limiting and retries |
| Parallelism, timeouts, reporting | Response parsing (return raw JSON) |

## Requirements

- Honor `ctx`; Gaugo uses it for case timeouts and cancellation.
- Return raw JSON matching `req.Schema` in `JudgeResponse.RawJSON`.
- Populate `Provider`, `Model`, `RequestID`, and `Latency` when available; Gaugo copies them into `MetricResult`.
- Keep errors concise and free of secrets.
