# Errors and Retries

## When to use this

Use this reference when debugging failures, configuring retry behavior, or deciding where errors appear in a Gaugo run.

For symptom-based debugging, see [Troubleshooting](../troubleshooting.md). For provider setup, see [Provider](../provider/index.md).

## Structured errors

Gaugo errors are meant to be inspected structurally. Do not parse error
strings. Use `errors.Is` or `ErrorInfo.Kind` for branching, and use `Code` as
fine-grained diagnostic metadata when you log or route failures.

```go
if errors.Is(err, gaugo.ErrProviderRateLimit) {
    // back off, mark infra as flaky, or retry later
}

info := gaugo.ClassifyError(err)
log.Printf("kind=%s code=%s provider=%s status=%d request_id=%s",
    info.Kind, info.Code, info.Provider, info.StatusCode, info.RequestID)
```

`ErrorInfo` includes `Kind`, `Code`, `Operation`, `Field`, `Provider`, `Wire`,
`Model`, `StatusCode`, `RequestID`, `BodyBytes`, `LimitBytes`, and `Retryable`
when available. Bundled Gaugo errors also support `errors.As`:

```go
var e *gaugo.Error
if errors.As(err, &e) {
    fmt.Println(e.Kind, e.Code, e.Retryable)
}
```

Custom judges and metrics can integrate without depending on Gaugo internals by
returning errors that implement any of these methods:

```go
GaugoErrorKind() string
GaugoErrorCode() string
GaugoOperation() string
GaugoField() string
GaugoProvider() string
GaugoWire() string
GaugoModel() string
GaugoStatusCode() int
GaugoRequestID() string
GaugoBodyBytes() int
GaugoLimitBytes() int64
GaugoRetryable() bool
```

## Configuration errors

Configuration errors are returned early.

```go
runner, err := gaugo.NewRunner(gaugo.WithParallelism(0))
if err != nil {
    return fmt.Errorf("invalid eval runner: %w", err)
}
```

Provider constructors also validate configuration.

```go
judge, err := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})
if err != nil {
    return fmt.Errorf("openai judge config: %w", err)
}
```

Common invalid config:

- missing API key for hosted providers
- invalid `BaseURL` or `EndpointURL`
- hosted provider URL not `https` in strict mode
- hosted provider URL host not allowed in strict mode
- negative retry delays
- negative body limits
- invalid parallelism
- invalid case timeout

## Run setup errors

`Runner.Run` returns an error when Gaugo cannot start a meaningful run.

```go
result, err := runner.Run(ctx, yourGaugoEvaluation, gaugo.AnswerRelevancy())
if err != nil {
    return fmt.Errorf("gaugo run setup: %w", err)
}
```

Examples:

- no cases registered
- nil run function
- no effective checks
- nil runner
- custom reporter panic (when `WithReporter` is configured)

## Case run errors

Errors returned by your `RunFunc` are stored per case.

```go
for _, c := range result.Cases {
    if c.RunError != nil {
        log.Printf("case %s failed: %v", c.Name, c.RunError)
    }
}
```

This preserves full-suite reporting even when several cases fail.

Panics in `RunFunc` are also contained and recorded as case run failures instead of crashing the process.

## Metric errors

Metric evaluation errors become failing `MetricResult` values.

Examples:

- missing judge for LLM-backed metric
- provider request failure
- invalid judge JSON
- output truncated by provider token limit
- model refusal or blocked content
- invalid threshold

```go
for _, c := range result.Cases {
    for _, m := range c.Metrics {
        if !m.Pass {
            log.Printf("case=%s metric=%s reason=%s", c.Name, m.Name, m.Reason)
        }
    }
}
```

## Provider HTTP errors

Bundled providers redact HTTP response bodies from status errors. Error strings include provider name, HTTP status, request id when available, and body length.

Treat provider errors as operational failures. In CI, prefer clear logs and bounded retries over unbounded reruns.

## Retry behavior

Bundled providers retry:

- `429 Too Many Requests`
- `5xx` server errors
- transient transport failures (timeouts, EOF, connection reset)

They do not retry:

- `400` invalid request
- `401` or `403` authentication failures
- malformed responses
- context cancellation
- body too large

Default retry config:

```go
retry := gaugo.DefaultRetryConfig()
// MaxAttempts: 3
// BaseDelay:   100 * time.Millisecond
// MaxDelay:    2 * time.Second
```

Custom retry config:

```go
judge, err := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Retry: gaugo.RetryConfig{
        MaxAttempts: 5,
        BaseDelay:   250 * time.Millisecond,
        MaxDelay:    5 * time.Second,
    },
})
```

`Retry-After` is honored when providers send it, capped by `MaxDelay`.

## Response body limits

Bundled providers cap response bodies. Use `MaxResponseBody` to adjust the limit.

```go
judge, err := openai.New(openai.Config{
    APIKey:          os.Getenv("OPENAI_API_KEY"),
    MaxResponseBody: 2 << 20,
})
```

Use `0` for the default limit. Negative values are invalid.

## Context cancellation

Gaugo passes the case context into your `RunFunc` and each metric. Use `WithCaseTimeout` to bound case execution.

```go
suite := gaugo.New(t,
    gaugo.WithCaseTimeout(10*time.Second),
    gaugo.WithJudge(judge),
)
```

Your system and custom metrics should check or pass through `ctx`.
