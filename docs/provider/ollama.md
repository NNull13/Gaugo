# Ollama Provider

## When to use this

Use Ollama for local evaluation, offline development, private model experiments, or CI environments with a local model service.

## Import

```go
import "github.com/nnull13/gaugo/provider/ollama"
```

## Minimal config

```go
judge, err := ollama.New(ollama.Config{
    Model: "llama3.1",
})
if err != nil {
    return err
}
```

Default model: `llama3.1`.

Default base URL: `http://127.0.0.1:11434`.

## Full config

```go
judge, err := ollama.New(ollama.Config{
    APIKey:               "",
    Model:                "llama3.1",
    BaseURL:              "http://127.0.0.1:11434",
    EndpointURL:          "",
    HTTPClient:           &http.Client{Timeout: 60 * time.Second},
    Mode:                 ollama.ModeNative,
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

Ollama does not require an API key for the default local server.

If your gateway requires auth, pass it explicitly:

```go
judge, err := ollama.New(ollama.Config{
    APIKey: os.Getenv("OLLAMA_API_KEY"),
    Model:  "llama3.1",
})
```

## URL behavior

`BaseURL` is the API root. For native mode, Gaugo appends `/api/chat`.

```go
ollama.Config{
    BaseURL: "http://127.0.0.1:11434",
    Mode:    ollama.ModeNative,
}
```

For OpenAI-compatible mode, Gaugo appends `/v1` before the selected OpenAI-compatible endpoint.

`EndpointURL` is a full endpoint override and wins over `BaseURL`.

```go
ollama.Config{
    EndpointURL: server.URL,
    Mode:        ollama.ModeNative,
}
```

## Modes

Native mode:

```go
judge, err := ollama.New(ollama.Config{
    Mode:  ollama.ModeNative,
    Model: "llama3.1",
})
```

OpenAI-compatible Chat Completions:

```go
judge, err := ollama.New(ollama.Config{
    Mode:                 ollama.ModeOpenAI,
    OpenAICompatEndpoint: ollama.OpenAIEndpointChat,
    Model:                "llama3.1",
})
```

OpenAI-compatible Responses:

```go
judge, err := ollama.New(ollama.Config{
    Mode:                 ollama.ModeOpenAI,
    OpenAICompatEndpoint: ollama.OpenAIEndpointResponses,
    Model:                "llama3.1",
})
```

Anthropic-compatible mode:

```go
judge, err := ollama.New(ollama.Config{
    Mode:  ollama.ModeAnthropic,
    Model: "llama3.1",
})
```

## Retry and body-limit example

```go
judge, err := ollama.New(ollama.Config{
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

- `invalid ollama config: unsupported mode`: use `ModeNative`, `ModeOpenAI`, or `ModeAnthropic`.
- `invalid ollama config: openai endpoint requires mode=openai`: set `Mode: ollama.ModeOpenAI`.
- `invalid ollama config: unsupported openai endpoint`: use `OpenAIEndpointChat` or `OpenAIEndpointResponses`.
- `ollama judge request failed`: local server unavailable, model missing, timeout, or retry exhaustion.
- `decode ollama response: empty message content`: model returned no parseable JSON.
- `provider response body too large`: increase `MaxResponseBody` or inspect the response.
