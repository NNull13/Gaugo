# Anthropic Provider

## When to use this

Use Anthropic when you want Claude models as your LLM judge through the Messages API.

## Import

```go
import "github.com/nnull13/gaugo/provider/anthropic"
```

## Minimal config

```go
judge, err := anthropic.New(anthropic.Config{
    APIKey: os.Getenv("ANTHROPIC_API_KEY"),
})
if err != nil {
    return err
}
```

Default model: `claude-sonnet-4-5`.

## Full config

```go
judge, err := anthropic.New(anthropic.Config{
    APIKey:          os.Getenv("ANTHROPIC_API_KEY"),
    Model:           "claude-sonnet-4-5",
    BaseURL:         "https://api.anthropic.com",
    EndpointURL:     "",
    AllowUnsafeURL:  false,
    APIVersion:      "2023-06-01",
    MaxTokens:       1024,
    HTTPClient:      &http.Client{Timeout: 30 * time.Second},
    Retry:           gaugo.DefaultRetryConfig(),
    MaxResponseBody: 1 << 20,
})
if err != nil {
    return err
}
```

## Environment variable

Recommended variable:

```sh
ANTHROPIC_API_KEY=...
```

Gaugo does not read environment variables automatically. Pass the value through `anthropic.Config`.

## URL behavior

`BaseURL` is the API root. Gaugo appends `/v1/messages`.

Strict mode is enabled by default:

- `https` is required
- host must be `api.anthropic.com`
- URL user info is rejected

```go
anthropic.Config{
    APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
    BaseURL: "https://api.anthropic.com",
}
```

`EndpointURL` is a full endpoint override and wins over `BaseURL`.

```go
anthropic.Config{
    APIKey:         "test-key",
    EndpointURL:    server.URL,
    AllowUnsafeURL: true,
}
```

Set `AllowUnsafeURL: true` only for trusted local tests, stubs, and custom gateways.

## Retry and body-limit example

```go
judge, err := anthropic.New(anthropic.Config{
    APIKey: os.Getenv("ANTHROPIC_API_KEY"),
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

Retries apply to transient `429`/`5xx` responses and transient transport failures. `Retry-After` is honored when returned by the provider.

## Common errors

- `invalid anthropic config: api key is required`: pass `APIKey`.
- `invalid anthropic config: endpoint URL`: check URL syntax and strict URL policy.
- `invalid anthropic config: base URL`: use `AllowUnsafeURL: true` for trusted local stubs or custom gateways.
- `anthropic judge request failed`: network, timeout, or retry exhaustion.
- `decode anthropic response: output truncated by max_tokens`: increase `MaxTokens`.
- `decode anthropic response: model refusal`: the judge refused the metric request.
- `provider response body too large`: increase `MaxResponseBody` or inspect the response.
