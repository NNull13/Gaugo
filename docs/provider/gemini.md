# Gemini Provider

## When to use this

Use Gemini when you want Google Gemini models as your LLM judge with JSON schema structured output.

## Import

```go
import "github.com/nnull13/gaugo/provider/gemini"
```

## Minimal config

```go
judge, err := gemini.New(gemini.Config{
    APIKey: os.Getenv("GEMINI_API_KEY"),
})
if err != nil {
    return err
}
```

Default model: `gemini-2.5-flash`.

## Full config

```go
judge, err := gemini.New(gemini.Config{
    APIKey:          os.Getenv("GEMINI_API_KEY"),
    Model:           "gemini-2.5-flash",
    BaseURL:         "https://generativelanguage.googleapis.com",
    EndpointURL:     "",
    AllowUnsafeURL:  false,
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
GEMINI_API_KEY=...
```

Gaugo does not read environment variables automatically. Pass the value through `gemini.Config`.

## URL behavior

`BaseURL` is the API root. Gaugo appends `/v1beta/models/{model}:generateContent`.

Strict mode is enabled by default:

- `https` is required
- host must be `generativelanguage.googleapis.com`
- URL user info is rejected

```go
gemini.Config{
    APIKey:  os.Getenv("GEMINI_API_KEY"),
    BaseURL: "https://generativelanguage.googleapis.com",
}
```

`EndpointURL` is a full endpoint override and wins over `BaseURL`.

```go
gemini.Config{
    APIKey:         "test-key",
    EndpointURL:    server.URL,
    AllowUnsafeURL: true,
}
```

Set `AllowUnsafeURL: true` only for trusted local tests, stubs, and custom gateways.

## Structured output behavior

Gaugo sends Gemini requests with:

- `generationConfig.responseMimeType` set to `application/json`
- `generationConfig.responseJsonSchema` set to the metric schema
- temperature set to `0`

The returned text is parsed as strict metric JSON by Gaugo.

## Retry and body-limit example

```go
judge, err := gemini.New(gemini.Config{
    APIKey: os.Getenv("GEMINI_API_KEY"),
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

- `invalid gemini config: api key is required`: pass `APIKey`.
- `invalid gemini config: base URL`: check `BaseURL` or `EndpointURL` against strict URL policy.
- `invalid gemini config: endpoint URL`: use `AllowUnsafeURL: true` for trusted local stubs or custom gateways.
- `gemini judge request failed`: network, timeout, or retry exhaustion.
- `decode gemini response: output truncated by max tokens`: choose a model/config with enough output room.
- `decode gemini response: output blocked`: the provider blocked the output for safety or policy reasons.
- `provider response body too large`: increase `MaxResponseBody` or inspect the response.
