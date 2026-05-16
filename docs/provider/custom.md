# Custom Provider

## When to use this

Use the custom provider for any model service that is not one of the hosted providers — local or remote. This includes local runtimes (Ollama, LM Studio), private inference endpoints, self-hosted NIM instances, and any gateway that exposes an OpenAI-compatible or Anthropic-compatible API.

## Import

```go
import "github.com/nnull13/gaugo/provider/custom"
```

## Minimal config

```go
judge, err := custom.New(custom.Config{
    Model: "llama3.1",
})
if err != nil {
    return err
}
```

Default model: `llama3.1`.

Default base URL: `http://127.0.0.1:11434`, which matches Ollama. Set `BaseURL` for LM Studio, remote endpoints, or other servers.

## Full config

```go
judge, err := custom.New(custom.Config{
    APIKey:               "",
    Model:                "llama3.1",
    BaseURL:              "http://127.0.0.1:11434",
    EndpointURL:          "",
    HTTPClient:           &http.Client{Timeout: 60 * time.Second},
    Mode:                 custom.ModeNative,
    OpenAICompatEndpoint: "",
    AnthropicAPIVersion:  "2023-06-01",
    AnthropicMaxTokens:   1024,
    Retry:                gaugo.DefaultRetryConfig(),
    RateLimit:            gaugo.RateLimitConfig{},
    MaxResponseBody:      1 << 20,
})
if err != nil {
    return err
}
```

## Environment variable

Custom endpoints often do not require an API key. The custom provider supplies a placeholder key for compatibility modes when you leave `APIKey` empty.

If your gateway requires auth, pass it explicitly:

```go
judge, err := custom.New(custom.Config{
    APIKey: os.Getenv("CUSTOM_LLM_API_KEY"),
    Model:  "llama3.1",
})
```

## URL behavior

`BaseURL` is the API root. For native mode, Gaugo appends `/api/chat`.

```go
custom.Config{
    BaseURL: "http://127.0.0.1:11434",
    Mode:    custom.ModeNative,
}
```

For OpenAI-compatible mode, Gaugo appends `/v1` before the selected endpoint. For example, LM Studio commonly uses `BaseURL: "http://127.0.0.1:1234"`. For a remote server use its full base URL.

`EndpointURL` is a full endpoint override and wins over `BaseURL`.

```go
custom.Config{
    EndpointURL: server.URL,
    Mode:        custom.ModeNative,
}
```

## Modes

Native mode (Ollama):

```go
judge, err := custom.New(custom.Config{
    Mode:  custom.ModeNative,
    Model: "llama3.1",
})
```

OpenAI-compatible Chat Completions:

```go
judge, err := custom.New(custom.Config{
    BaseURL:              "http://127.0.0.1:1234",
    Mode:                 custom.ModeOpenAI,
    OpenAICompatEndpoint: custom.OpenAIEndpointChat,
    Model:                "local-model",
})
```

OpenAI-compatible Responses:

```go
judge, err := custom.New(custom.Config{
    Mode:                 custom.ModeOpenAI,
    OpenAICompatEndpoint: custom.OpenAIEndpointResponses,
    Model:                "llama3.1",
})
```

Anthropic-compatible mode:

```go
judge, err := custom.New(custom.Config{
    Mode:  custom.ModeAnthropic,
    Model: "llama3.1",
})
```

## Remote custom endpoints

Any OpenAI-compatible remote server works. For example, a self-hosted NIM or a private vLLM instance:

```go
judge, err := custom.New(custom.Config{
    APIKey:  os.Getenv("REMOTE_API_KEY"),
    BaseURL: "https://my-inference-server.internal/v1",
    Mode:    custom.ModeOpenAI,
    Model:   "my-model",
})
```

## Retry and body-limit example

```go
judge, err := custom.New(custom.Config{
    Model: "llama3.1",
    Retry: gaugo.RetryConfig{
        MaxAttempts: 2,
        BaseDelay:   100 * time.Millisecond,
        MaxDelay:    1 * time.Second,
    },
    MaxResponseBody: 2 << 20,
})
if err != nil {
    return err
}
```

Retries apply to transient `429`/`5xx` responses and transient transport failures.

## Rate limiting example

For remote custom endpoints with per-minute quotas:

```go
judge, err := custom.New(custom.Config{
    APIKey:  os.Getenv("REMOTE_API_KEY"),
    BaseURL: "https://my-inference-server.internal/v1",
    Mode:    custom.ModeOpenAI,
    Model:   "my-model",
    RateLimit: gaugo.RateLimitConfig{
        RequestsPerMinute: 30,
    },
})
if err != nil {
    return err
}
```

## Common errors

- `invalid custom config: unsupported mode`: use `ModeNative`, `ModeOpenAI`, or `ModeAnthropic`.
- `invalid custom config: openai endpoint requires mode=openai`: set `Mode: custom.ModeOpenAI`.
- `invalid custom config: unsupported openai endpoint`: use `OpenAIEndpointChat` or `OpenAIEndpointResponses`.
- `custom judge request failed`: server unavailable, model missing, timeout, or retry exhaustion.
- `decode custom response: empty message content`: model returned no parseable JSON.
- `provider response body too large`: increase `MaxResponseBody` or inspect the response.
