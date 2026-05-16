# Configuration

## When to use this

Use this reference when configuring runners, suites, judges, retries, timeouts, parallelism, or provider URLs.

## Runner and Suite options

Options are passed to `gaugo.NewRunner` or `gaugo.New`.

```go
runner, err := gaugo.NewRunner(
    gaugo.WithParallelism(8),
    gaugo.WithCaseTimeout(10*time.Second),
)
```

```go
suite := gaugo.New(t,
    gaugo.WithJudge(judge),
    gaugo.WithParallelism(8),
    gaugo.WithCaseTimeout(10*time.Second),
)
```

Invalid options fail early. `NewRunner` returns an error. `New(t, ...)` fails the test.

## WithJudge

```go
gaugo.WithJudge(judge)
```

Configures the LLM judge used by LLM-based metrics such as `ContextRelevancy`, `Faithfulness`, and `AnswerRelevancy`.

`WithJudge` may receive any value that implements:

```go
type Judge interface {
    EvaluateJSON(ctx context.Context, req gaugo.JudgeRequest) (gaugo.JudgeResponse, error)
}
```

## WithParallelism

```go
gaugo.WithParallelism(16)
```

Sets the maximum number of cases evaluated concurrently. The default is `runtime.GOMAXPROCS(0)`.

Use lower values when:

- Provider rate limits are tight.
- Your system under test shares limited resources.
- You need easier local debugging.

Use higher values when:

- Cases are mostly I/O-bound.
- Provider quotas allow concurrent requests.
- You are running larger suites in CI or batch jobs.

The value must be positive.

## WithCaseTimeout

```go
gaugo.WithCaseTimeout(15 * time.Second)
```

Applies a per-case timeout to the run function and metric evaluation. The timeout is enforced with the case context, so your `RunFunc`, provider client, and custom metrics should honor `ctx`.

The value must be non-negative. Use `0` for no per-case timeout.

## WithReporter

```go
gaugo.WithReporter(reporter)
```

Overrides the default reporter. A reporter receives the complete `RunResult`.

```go
type Reporter interface {
    Report(ctx context.Context, result gaugo.RunResult)
}
```

Important: when a custom reporter is configured on a `Suite`, Gaugo does not also run the default `testing` assertion reporter. If you still want test failures, make your reporter call `gaugo.Assert(t, result)` or use `Runner` directly and assert afterward.

## WithMetricDetailsLimit

```go
gaugo.WithMetricDetailsLimit(16 * 1024)
```

Caps stored `MetricResult.Details` bytes per metric. The default is 8 KiB.

Use `0` to disable metric details entirely.

The value must be non-negative.

## Metric options

Built-in metrics accept metric-specific options in their constructors.

```go
gaugo.Faithfulness(gaugo.WithThreshold(0.9))
gaugo.SchemaCompliance(gaugo.WithSchema(schema))
gaugo.ExpectedJSON(gaugo.WithExpectedFields(map[string]any{"status": "ok"}))
gaugo.Latency(gaugo.WithMaxLatency(250 * time.Millisecond))
gaugo.AnswerLength(gaugo.WithMinLength(20), gaugo.WithMaxLength(500))
```

Options:

- `WithThreshold(v float64)` sets pass/fail threshold in `[0,1]`.
- `WithSchema(schema json.RawMessage)` configures `SchemaCompliance`.
- `WithExpectedFields(fields map[string]any)` configures `ExpectedJSON`; dotted paths such as `meta.count` and array indexes such as `items.0.id` are supported.
- `WithMaxLatency(d time.Duration)` configures `Latency`.
- `WithMinLength(n int)` and `WithMaxLength(n int)` configure `AnswerLength`.

## RetryConfig

Bundled providers accept `gaugo.RetryConfig`.

```go
type RetryConfig struct {
    MaxAttempts int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
}
```

Defaults:

```go
gaugo.DefaultRetryConfig()
// MaxAttempts: 3
// BaseDelay:   100 * time.Millisecond
// MaxDelay:    2 * time.Second
```

Zero values use defaults. Negative values are invalid. Providers retry transient HTTP statuses (`429`, `5xx`) and transient transport errors (for example timeout, EOF, and connection reset). `Retry-After` is honored and capped by `MaxDelay`.

## RateLimitConfig

Bundled providers accept `gaugo.RateLimitConfig` to throttle outgoing judge requests before they reach the provider.

```go
type RateLimitConfig struct {
    RequestsPerMinute int
    Burst             int
}
```

`RequestsPerMinute` is the maximum number of requests per minute. Zero disables rate limiting (default: no limit).

`Burst` is the maximum number of tokens that can accumulate. Zero defaults to `1`, meaning only one request is allowed before throttling begins. Set a higher value to allow an initial burst before the steady-state rate takes effect.

The limiter is shared across all goroutines that use the same judge instance, so it enforces a limit per API key rather than per goroutine.

```go
judge, err := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    RateLimit: gaugo.RateLimitConfig{
        RequestsPerMinute: 60,
    },
})
```

Rate limiting is proactive — requests are held before they are sent. This prevents `429` errors that would otherwise corrupt metric scores. Use it alongside `WithParallelism` when running large suites against providers with strict per-minute quotas.

## Anthropic large-suite tuning

For large RAG suites that use Anthropic as the judge, start conservative. A case
with the full RAG triad can make one provider request per LLM-backed metric, so
total request volume grows with `cases * metrics`.

```go
judge, err := anthropic.New(anthropic.Config{
    APIKey:    os.Getenv("ANTHROPIC_API_KEY"),
    Model:     "claude-sonnet-4-5",
    MaxTokens: 1024,
    HTTPClient: &http.Client{
        Timeout: 45 * time.Second,
    },
    Retry: gaugo.RetryConfig{
        MaxAttempts: 4,
        BaseDelay:   250 * time.Millisecond,
        MaxDelay:    8 * time.Second,
    },
})
if err != nil {
    return err
}

suite := gaugo.New(t,
    gaugo.WithJudge(judge),
    gaugo.WithParallelism(1),
    gaugo.WithCaseTimeout(90*time.Second),
)
```

Recommended starting points:

- Use `WithParallelism(1)` or `WithParallelism(2)` for broad Anthropic suites, then raise it only after observing rate limits and latency.
- Keep provider retries bounded; retrying `429` and `5xx` responses is useful, but high concurrency plus many retries can amplify load.
- Set `HTTPClient.Timeout` for each provider request and `WithCaseTimeout` for the whole case. The case timeout must allow your `RunFunc` plus every metric for that case.
- Increase `MaxTokens` if Anthropic reports truncated structured output, especially with long contexts or verbose metric reasons.
- Prefer small PR suites and scheduled large suites so quality signal does not depend on an overloaded provider lane.

## Provider HTTP configuration

Bundled provider configs share these fields:

```go
HTTPClient      *http.Client
Retry           gaugo.RetryConfig
RateLimit       gaugo.RateLimitConfig
MaxResponseBody int64
```

`HTTPClient` lets you provide custom timeouts, transports, proxies, or test servers.

`MaxResponseBody` caps provider response bodies. Use `0` for the provider default, currently 1 MiB.

## BaseURL and EndpointURL

Bundled provider configs use consistent URL semantics:

- `BaseURL` is an API root.
- `EndpointURL` is a full endpoint override.

Hosted providers (OpenAI, Anthropic, Gemini, xAI) validate URLs in strict mode by default:

- `https` is required.
- Host must be the official provider host.
- URL user info is rejected.

Hosted provider configs expose `AllowUnsafeURL bool` as an explicit escape hatch for local stubs, test servers, and custom gateways.

Prefer strict defaults in production.

```go
judge, err := openai.New(openai.Config{
    APIKey:  os.Getenv("OPENAI_API_KEY"),
    BaseURL: "https://api.openai.com/v1",
})
```

Use `EndpointURL` for tests, stubs, or gateways that expose a single exact endpoint.

```go
judge, err := openai.New(openai.Config{
    APIKey:         "test-key",
    EndpointURL:    server.URL,
    AllowUnsafeURL: true,
})
```

If both are set, `EndpointURL` wins.

## Tips

- Set `WithCaseTimeout` before running LLM-backed metrics in CI.
- Tune `WithParallelism` against provider rate limits, not CPU alone.
- Use `RateLimitConfig.RequestsPerMinute` when `WithParallelism` alone is not enough to stay under provider quotas.
- Keep `MaxResponseBody` small unless you intentionally need large provider responses.
- Use strict URL defaults for hosted providers.
- Use `AllowUnsafeURL` only for trusted test/proxy infrastructure.
