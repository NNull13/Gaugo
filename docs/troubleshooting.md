# Troubleshooting

## When to use this

Use this page when a Gaugo integration fails, produces no metrics, or behaves differently between local runs and CI.

## Operational failure or quality failure?

Start by identifying which kind of failure you have:

| Signal | Meaning |
| --- | --- |
| `Runner.Run` returned an error | The suite could not start or the reporter failed. |
| `CaseResult.RunError` is set | Your application did not produce an answer for that case. |
| `MetricResult` fails and `gaugo.MetricErrorInfo` returns metadata | The metric could not evaluate because of a judge, provider, timeout, parse, or config issue. |
| `MetricResult` fails without `MetricErrorInfo` | The metric evaluated successfully and judged quality below threshold. |

Treat the first three as operational failures. Treat the last one as a quality
failure that should usually be fixed in retrieval, prompting, generation, or
source content.

## `no effective metrics provided`

Cause: the run has no `ExpectedContains` checks and no non-nil metrics.

Fix:

```go
suite.Case("pricing",
    gaugo.Question("How does enterprise pricing work?"),
    gaugo.ExpectedContains("sales"),
)
```

Or pass a metric:

```go
suite.Assert(ctx, yourGaugoEvaluation, gaugo.AnswerRelevancy())
```

LLM-backed metrics require `WithJudge`.

## `no cases registered`

Cause: `Run` or `Assert` was called before registering cases.

Fix:

```go
suite.Case("pricing", gaugo.Question("Pricing?"), gaugo.ExpectedContains("sales"))
suite.Assert(ctx, yourGaugoEvaluation)
```

## `input question is required`

Cause: a case did not call `gaugo.Question`, or the question was empty after trimming.

Fix:

```go
suite.Case("pricing",
    gaugo.Question("How does enterprise pricing work?"),
    gaugo.ExpectedContains("sales"),
)
```

## `ExpectedContains[...] must be non-empty`

Cause: one `gaugo.ExpectedContains(...)` value was empty or whitespace-only after trimming.

Fix:

```go
suite.Case("pricing",
    gaugo.Question("How does enterprise pricing work?"),
    gaugo.ExpectedContains("sales"),
)
```

## Duplicate case names

Cause: two cases have the same final name.

Fix: use stable unique names.

```go
suite.Case("pricing enterprise", ...)
suite.Case("pricing self serve", ...)
```

## LLM metric says a judge is required

Cause: `ContextRelevancy`, `Faithfulness`, or `AnswerRelevancy` was used without `WithJudge`.

Fix:

```go
judge, err := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})
if err != nil {
    t.Fatalf("openai judge config: %v", err)
}

suite := gaugo.New(t, gaugo.WithJudge(judge))
```

See [Provider](provider/index.md).

## Provider config error

Cause: required provider fields are missing or invalid.

Common fixes:

- Pass `APIKey` for hosted providers.
- Use a valid absolute `BaseURL` or `EndpointURL`.
- In strict mode (default), hosted providers require `https` and official provider hosts.
- Set `AllowUnsafeURL: true` only for trusted local test servers, stubs, or custom gateways.
- Keep retry durations non-negative.
- Keep `MaxResponseBody` non-negative.

See [Configuration](reference/configuration.md).

## Provider request failed

Cause: network failure, timeout, rate limit, server error, or invalid credentials. This is an operational failure, not a quality score.

Fix:

- Verify the API key in the environment.
- Lower `WithParallelism`.
- Increase `WithCaseTimeout`.
- Configure retries.
- Use a custom `HTTPClient` with a timeout.

```go
judge, err := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    HTTPClient: &http.Client{
        Timeout: 30 * time.Second,
    },
    Retry: gaugo.RetryConfig{
        MaxAttempts: 4,
        BaseDelay:   250 * time.Millisecond,
        MaxDelay:    5 * time.Second,
    },
})
```

## Anthropic large suite is slow or rate limited

Cause: broad suites can create many judge requests. With the RAG triad, request
volume is roughly `cases * 3`, and retries can increase that during provider
incidents or rate limits.

Fix:

- Start with `WithParallelism(1)` or `WithParallelism(2)`.
- Keep `RetryConfig` bounded and honor `Retry-After`.
- Set `HTTPClient.Timeout` so one request cannot hang indefinitely.
- Set `WithCaseTimeout` high enough for your `RunFunc` plus all metrics in the case.
- Increase Anthropic `MaxTokens` if failures mention output truncation.
- Move broad Anthropic suites to scheduled CI and keep PR suites smaller.

```go
judge, err := anthropic.New(anthropic.Config{
    APIKey: os.Getenv("ANTHROPIC_API_KEY"),
    HTTPClient: &http.Client{
        Timeout: 45 * time.Second,
    },
    Retry: gaugo.RetryConfig{
        MaxAttempts: 4,
        BaseDelay:   250 * time.Millisecond,
        MaxDelay:    8 * time.Second,
    },
    MaxTokens: 1024,
})
if err != nil {
    t.Fatalf("anthropic judge config: %v", err)
}

suite := gaugo.New(t,
    gaugo.WithJudge(judge),
    gaugo.WithParallelism(1),
    gaugo.WithCaseTimeout(90*time.Second),
)
```

## Output truncated by token limit

Cause: the provider stopped before returning complete metric JSON.

Fix:

- Increase provider output token settings when available.
- Use a model with better structured-output reliability.
- Keep case context shorter.
- Keep the answer under evaluation concise.

Anthropic exposes `MaxTokens` in provider config.

## Model refusal or blocked output

Cause: the judge refused or the provider blocked the metric response.

Fix:

- Inspect the case question, answer, and context.
- Keep metric prompts focused on evaluation.
- Use deterministic checks for sensitive policy boundaries.
- Consider a provider/model better suited for your evaluation data.

## JSON parse failures from judge output

Cause: the judge returned JSON that does not match the metric schema.

Fix:

- Use a bundled provider with structured output.
- Keep temperature deterministic.
- For custom judges, return raw JSON in `JudgeResponse.RawJSON`.
- Do not wrap `RawJSON` in Markdown or prose.

See [Custom Judges](extending/custom-judges.md).

## Tests pass locally but fail in CI

Common causes:

- Missing API keys in CI secrets.
- Different provider model behavior.
- Higher CI latency causing timeouts.
- Rate limits from parallel jobs.
- Tests relying on live LLM behavior with tight thresholds.

Fix:

- Skip LLM-backed tests when credentials are missing.
- Lower `WithParallelism`.
- Increase `WithCaseTimeout`.
- Pin provider model names.
- Start with deterministic checks in required CI.
- Run expensive LLM metrics in a scheduled or protected workflow.
- Classify `MetricResult` failures with `gaugo.MetricErrorInfo` before treating them as quality regressions.

See [CI Integration](guides/ci-integration.md) and [Production](guides/production.md).
