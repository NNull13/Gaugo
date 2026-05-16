# NVIDIA NIM Provider

## When to use this

Use NVIDIA NIM when you want a free hosted judge without a credit card. NVIDIA NIM exposes 100+ models (Llama 3.x, Mistral, DeepSeek, Nemotron) through an OpenAI-compatible API. Get a free API key at [build.nvidia.com](https://build.nvidia.com) — no credit card required.

Free tier limits: 1,000–5,000 inference credits and 40 requests per minute. Suitable for evaluation pipelines and CI.

## Import

```go
import "github.com/nnull13/gaugo/provider/nvidia"
```

## Minimal config

```go
judge, err := nvidia.New(nvidia.Config{
    APIKey: os.Getenv("NVIDIA_API_KEY"),
})
if err != nil {
    return err
}
```

Default model: `meta/llama-3.3-70b-instruct`.

## Full config

```go
judge, err := nvidia.New(nvidia.Config{
    APIKey:          os.Getenv("NVIDIA_API_KEY"),
    Model:           "meta/llama-3.3-70b-instruct",
    BaseURL:         "https://integrate.api.nvidia.com/v1",
    EndpointURL:     "",
    AllowUnsafeURL:  false,
    HTTPClient:      &http.Client{Timeout: 60 * time.Second},
    Retry:           gaugo.DefaultRetryConfig(),
    RateLimit:       gaugo.RateLimitConfig{},
    MaxResponseBody: 1 << 20,
})
if err != nil {
    return err
}
```

A longer timeout is recommended because some large models have higher latency.

## Environment variable

Recommended variable:

```sh
NVIDIA_API_KEY=nvapi-...
```

Gaugo does not read environment variables automatically. Pass the value through `nvidia.Config`.

## Available models

A subset of the most capable free-tier models:

| Model | Parameters | Context |
| --- | --- | --- |
| `meta/llama-3.3-70b-instruct` | 70B | 128K |
| `meta/llama-3.1-405b-instruct` | 405B | 128K |
| `nvidia/llama-3.1-nemotron-70b-instruct` | 70B | 128K |
| `mistralai/mistral-large` | 123B | 128K |
| `deepseek-ai/deepseek-r1` | — | 128K |
| `meta/llama-3.1-8b-instruct` | 8B | 128K |

See the full catalog at [build.nvidia.com](https://build.nvidia.com).

## URL behavior

`BaseURL` is the API root. Gaugo appends `/chat/completions`.

Strict mode is enabled by default:

- `https` is required
- host must be `integrate.api.nvidia.com`
- URL user info is rejected

```go
nvidia.Config{
    APIKey:  os.Getenv("NVIDIA_API_KEY"),
    BaseURL: "https://integrate.api.nvidia.com/v1",
}
```

`EndpointURL` is a full endpoint override and wins over `BaseURL`.

```go
nvidia.Config{
    APIKey:         "nvapi-test",
    EndpointURL:    server.URL,
    AllowUnsafeURL: true,
}
```

Set `AllowUnsafeURL: true` only for trusted local tests, stubs, and custom gateways.

## Rate limit and retry

NVIDIA NIM free tier is capped at 40 requests per minute. Use `RateLimitConfig` to throttle requests proactively and avoid `429` errors corrupting metric scores when running large suites:

```go
judge, err := nvidia.New(nvidia.Config{
    APIKey: os.Getenv("NVIDIA_API_KEY"),
    RateLimit: gaugo.RateLimitConfig{
        RequestsPerMinute: 40,
    },
    Retry: gaugo.RetryConfig{
        MaxAttempts: 5,
        BaseDelay:   500 * time.Millisecond,
        MaxDelay:    10 * time.Second,
    },
    MaxResponseBody: 2 << 20,
})
if err != nil {
    return err
}
```

Use both together: `RateLimit` prevents hitting the quota, `Retry` handles occasional bursts and transient `5xx` errors.

## Common errors

- `invalid nvidia config: api key is required`: pass `APIKey` with your `nvapi-...` key.
- `invalid nvidia config: base URL`: check `BaseURL` or `EndpointURL` against strict URL policy.
- `invalid nvidia config: endpoint URL`: use `AllowUnsafeURL: true` for trusted local stubs or custom gateways.
- `nvidia judge request failed`: network, timeout, or retry exhaustion (often rate limiting on free tier).
- `decode nvidia response: output truncated by token limit`: choose a model with higher output capacity or reduce prompt size.
- `decode nvidia response: model refusal`: the judge refused the metric request.
- `provider response body too large`: increase `MaxResponseBody` or inspect the provider response.
