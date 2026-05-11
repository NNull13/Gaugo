# Local Provider

## When to use this

Use the local provider for local evaluation, offline development, private model experiments, or CI environments with a local model service such as Ollama, LM Studio, or another OpenAI-compatible server.

## Import

```go
import "github.com/nnull13/gaugo/provider/local"
```

## Minimal config

```go
judge, err := local.New(local.Config{
    Model: "llama3.1",
})
if err != nil {
    return err
}
```

Default model: `llama3.1`.

Default base URL: `http://127.0.0.1:11434`, which matches Ollama. Set `BaseURL` for LM Studio or other local servers.

## Full config

```go
judge, err := local.New(local.Config{
    APIKey:               "",
    Model:                "llama3.1",
    BaseURL:              "http://127.0.0.1:11434",
    EndpointURL:          "",
    HTTPClient:           &http.Client{Timeout: 60 * time.Second},
    Mode:                 local.ModeNative,
    OpenAICompatEndpoint: "",
    AnthropicAPIVersion:  "2023-06-01",
    AnthropicMaxTokens:   1024,
    Retry:                gaugo.DefaultRetryConfig(),
    MaxResponseBody:      1 << 20,
})
if err != nil {
    return err
}
```

## Environment variable

Local servers often do not require an API key. The local provider supplies a placeholder key for compatibility modes when you leave `APIKey` empty.

If your gateway requires auth, pass it explicitly:

```go
judge, err := local.New(local.Config{
    APIKey: os.Getenv("LOCAL_LLM_API_KEY"),
    Model:  "llama3.1",
})
```

## URL behavior

`BaseURL` is the API root. For native mode, Gaugo appends `/api/chat`.

```go
local.Config{
    BaseURL: "http://127.0.0.1:11434",
    Mode:    local.ModeNative,
}
```

For OpenAI-compatible mode, Gaugo appends `/v1` before the selected OpenAI-compatible endpoint. For example, LM Studio commonly uses `BaseURL: "http://127.0.0.1:1234"`.

`EndpointURL` is a full endpoint override and wins over `BaseURL`.

```go
local.Config{
    EndpointURL: server.URL,
    Mode:        local.ModeNative,
}
```

## Modes

Native mode:

```go
judge, err := local.New(local.Config{
    Mode:  local.ModeNative,
    Model: "llama3.1",
})
```

OpenAI-compatible Chat Completions:

```go
judge, err := local.New(local.Config{
    BaseURL:              "http://127.0.0.1:1234",
    Mode:                 local.ModeOpenAI,
    OpenAICompatEndpoint: local.OpenAIEndpointChat,
    Model:                "local-model",
})
```

OpenAI-compatible Responses:

```go
judge, err := local.New(local.Config{
    Mode:                 local.ModeOpenAI,
    OpenAICompatEndpoint: local.OpenAIEndpointResponses,
    Model:                "llama3.1",
})
```

Anthropic-compatible mode:

```go
judge, err := local.New(local.Config{
    Mode:  local.ModeAnthropic,
    Model: "llama3.1",
})
```

## Retry and body-limit example

```go
judge, err := local.New(local.Config{
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

## Common errors

- `invalid local config: unsupported mode`: use `ModeNative`, `ModeOpenAI`, or `ModeAnthropic`.
- `invalid local config: openai endpoint requires mode=openai`: set `Mode: local.ModeOpenAI`.
- `invalid local config: unsupported openai endpoint`: use `OpenAIEndpointChat` or `OpenAIEndpointResponses`.
- `ollama judge request failed`: local server unavailable, model missing, timeout, or retry exhaustion.
- `decode ollama response: empty message content`: model returned no parseable JSON.
- `provider response body too large`: increase `MaxResponseBody` or inspect the response.
