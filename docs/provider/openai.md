# OpenAI Provider

## When to use this

Use OpenAI when you want a hosted judge with strict structured output and simple CI setup.

## Import

```go
import "github.com/nnull13/gaugo/provider/openai"
```

## Minimal config

```go
judge, err := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})
if err != nil {
    return err
}
```

Default model: `gpt-4.1-mini`.

## Full config

```go
judge, err := openai.New(openai.Config{
    APIKey:          os.Getenv("OPENAI_API_KEY"),
    Model:           "gpt-4.1-mini",
    BaseURL:         "https://api.openai.com/v1",
    EndpointURL:     "",
    AllowUnsafeURL:  false,
    HTTPClient:      &http.Client{Timeout: 30 * time.Second},
    Retry:           gaugo.DefaultRetryConfig(),
    RateLimit:       gaugo.RateLimitConfig{},
    MaxResponseBody: 1 << 20,
})
if err != nil {
    return err
}
```

## Environment variable

Recommended variable:

```sh
OPENAI_API_KEY=...
```

Gaugo does not read environment variables automatically. Pass the value through `openai.Config`.

## URL behavior

`BaseURL` is the API root. Gaugo appends `/chat/completions`.

Strict mode is enabled by default:

- `https` is required
- host must be `api.openai.com`
- URL user info is rejected

```go
openai.Config{
    APIKey:  os.Getenv("OPENAI_API_KEY"),
    BaseURL: "https://api.openai.com/v1",
}
```

`EndpointURL` is a full endpoint override and wins over `BaseURL`.

```go
openai.Config{
    APIKey:         "test-key",
    EndpointURL:    server.URL,
    AllowUnsafeURL: true,
}
```

Set `AllowUnsafeURL: true` only for trusted local tests, stubs, and custom gateways.

## Retry and body-limit example

```go
judge, err := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Retry: gaugo.RetryConfig{
        MaxAttempts: 4,
        BaseDelay:   200 * time.Millisecond,
        MaxDelay:    3 * time.Second,
    },
    MaxResponseBody: 2 << 20,
})
if err != nil {
    return err
}
```

Retries apply to transient `429`/`5xx` responses and transient transport failures.

## Rate limiting example

Use `RateLimit` to prevent `429` errors when running large parallel suites.

```go
judge, err := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    RateLimit: gaugo.RateLimitConfig{
        RequestsPerMinute: 60,
    },
})
if err != nil {
    return err
}
```

The limiter is shared across all goroutines using this judge instance, so the limit applies per API key.

## Common errors

- `invalid openai config: api key is required`: pass `APIKey`.
- `invalid openai config: base URL`: check `BaseURL` or `EndpointURL` against strict URL policy.
- `invalid openai config: endpoint URL`: use `AllowUnsafeURL: true` for trusted local stubs or custom gateways.
- `openai judge request failed`: network, timeout, or retry exhaustion.
- `decode openai response: output truncated by token limit`: choose a model/config with enough output room.
- `decode openai response: model refusal`: the judge refused the metric request.
- `provider response body too large`: increase `MaxResponseBody` or inspect the provider response.
