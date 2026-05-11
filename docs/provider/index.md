# Providers

## When to use this

Use this guide to choose and configure an LLM judge provider for built-in metrics such as `Faithfulness` and `AnswerRelevancy`.

If you do not need an LLM judge, start with [Deterministic Checks](../examples/deterministic-checks.md).

## Provider matrix

| Provider | Package | Default model | Auth | Notes |
| --- | --- | --- | --- | --- |
| OpenAI | `provider/openai` | `gpt-4.1-mini` | required | Chat Completions structured output |
| Anthropic | `provider/anthropic` | `claude-sonnet-4-5` | required | Messages API structured output |
| Gemini | `provider/gemini` | `gemini-2.5-flash` | required | GenerateContent with JSON schema |
| xAI | `provider/xai` | `grok-4.3` | required | Responses API by default, Chat Completions optional |
| Local | `provider/local` | `llama3.1` | optional | Ollama native, OpenAI-compatible, or Anthropic-compatible local modes |

## Basic pattern

```go
judge, err := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4.1-mini",
})
if err != nil {
    t.Fatalf("openai judge config: %v", err)
}

suite := gaugo.New(t, gaugo.WithJudge(judge))
```

All hosted provider constructors return `(*Judge, error)` and validate configuration before any request is made.

## Shared config semantics

Most provider configs share:

```go
APIKey          string
Model           string
BaseURL         string
EndpointURL     string
AllowUnsafeURL  bool // hosted providers only
HTTPClient      *http.Client
Retry           gaugo.RetryConfig
MaxResponseBody int64
```

`BaseURL` is an API root. `EndpointURL` is a full endpoint override. If both are set, `EndpointURL` wins.

Hosted providers (OpenAI, Anthropic, Gemini, xAI) are strict by default: `https` only and official provider hosts only. Set `AllowUnsafeURL: true` only for trusted local stubs, test servers, or custom gateways.

See [Configuration](../reference/configuration.md) for shared behavior.

## Choose a provider

- Choose OpenAI when you want a widely available structured-output judge with simple setup.
- Choose Anthropic when your stack already uses Anthropic or you want Claude-based judgments.
- Choose Gemini when you use Google AI infrastructure or want Gemini structured output.
- Choose xAI when your evaluation policy standardizes on Grok models.
- Choose Local for local development, offline tests, LM Studio, Ollama, or private model experiments.

## Provider pages

- [OpenAI](openai.md)
- [Anthropic](anthropic.md)
- [Gemini](gemini.md)
- [xAI](xai.md)
- [Local](local.md)
