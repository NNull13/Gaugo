# xAI Provider

## When to use this

Use xAI when you want Grok models as your LLM judge. Gaugo uses the Responses-compatible path by default and can switch to Chat Completions when needed.

## Import

```go
import "github.com/nnull13/gaugo/provider/xai"
```

## Minimal config

```go
judge, err := xai.New(xai.Config{
    APIKey: os.Getenv("XAI_API_KEY"),
})
if err != nil {
    return err
}
```

Default model: `grok-4.3`.

## Full config

```go
judge, err := xai.New(xai.Config{
    APIKey:             os.Getenv("XAI_API_KEY"),
    Model:              "grok-4.3",
    BaseURL:            "https://api.x.ai/v1",
    EndpointURL:        "",
    AllowUnsafeURL:     false,
    UseChatCompletions: false,
    HTTPClient:         &http.Client{Timeout: 30 * time.Second},
    Retry:              gaugo.DefaultRetryConfig(),
    MaxResponseBody:    1 << 20,
})
if err != nil {
    return err
}
```

## Environment variable

Recommended variable:

```sh
XAI_API_KEY=...
```

Gaugo does not read environment variables automatically. Pass the value through `xai.Config`.

## URL behavior

`BaseURL` is the API root. Gaugo appends `/responses` by default or `/chat/completions` when `UseChatCompletions` is true.

Strict mode is enabled by default:

- `https` is required
- host must be `api.x.ai`
- URL user info is rejected

```go
xai.Config{
    APIKey:  os.Getenv("XAI_API_KEY"),
    BaseURL: "https://api.x.ai/v1",
}
```

`EndpointURL` is a full endpoint override and wins over `BaseURL`.

```go
xai.Config{
    APIKey:         "test-key",
    EndpointURL:    server.URL,
    AllowUnsafeURL: true,
}
```

Set `AllowUnsafeURL: true` only for trusted local tests, stubs, and custom gateways.

## Chat Completions mode

```go
judge, err := xai.New(xai.Config{
    APIKey:             os.Getenv("XAI_API_KEY"),
    UseChatCompletions: true,
})
```

Use this only when your gateway or selected model requires the Chat Completions-style protocol.

## Retry and body-limit example

```go
judge, err := xai.New(xai.Config{
    APIKey: os.Getenv("XAI_API_KEY"),
    Retry: gaugo.RetryConfig{
        MaxAttempts: 4,
        BaseDelay:   250 * time.Millisecond,
        MaxDelay:    5 * time.Second,
    },
    MaxResponseBody: 2 << 20,
})
if err != nil {
    return err
}
```

Retries apply to transient `429`/`5xx` responses and transient transport failures.

## Common errors

- `invalid xai config: api key is required`: pass `APIKey`.
- `invalid xai config: endpoint URL`: check URL syntax and strict URL policy.
- `invalid xai config: base URL`: use `AllowUnsafeURL: true` for trusted local stubs or custom gateways.
- `responses judge request failed` or `openai judge request failed`: network, timeout, or retry exhaustion.
- `decode responses response: incomplete output`: the Responses API returned an incomplete result.
- `decode openai response: output truncated by token limit`: Chat Completions output hit a limit.
- `provider response body too large`: increase `MaxResponseBody` or inspect the response.
