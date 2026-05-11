# Results and Reporting

## When to use this

Use this reference when consuming programmatic results, writing custom reporters, or understanding what Gaugo records for each run.

For custom reporter implementations, see [Custom Reporters](../extending/custom-reporters.md). For pipelines and dashboards, see [Dashboards and Pipelines](../examples/dashboards-and-pipelines.md).

## Result types

```go
type RunResult struct {
    Cases []CaseResult
}

type CaseResult struct {
    Name     string
    Metrics  []MetricResult
    RunError error
    Elapsed  time.Duration
}

type MetricResult struct {
    Name    string
    Score   float64
    Pass    bool
    Reason  string
    Details []byte
}
```

`RunResult.Cases` preserves registration order even when cases run concurrently.

## Programmatic consumption

```go
runner, err := gaugo.NewRunner(gaugo.WithParallelism(8))
if err != nil {
    return err
}

if err := runner.Case("pricing",
    gaugo.Question("How does enterprise pricing work?"),
    gaugo.ExpectedContains("sales"),
); err != nil {
    return err
}

result, err := runner.Run(ctx, yourGaugoEvaluation)
if err != nil {
    return err
}

for _, c := range result.Cases {
    if c.RunError != nil {
        log.Printf("case %s failed to run: %v", c.Name, c.RunError)
        continue
    }
    for _, m := range c.Metrics {
        log.Printf("case=%s metric=%s score=%.3f pass=%t", c.Name, m.Name, m.Score, m.Pass)
    }
}
```

## Error placement

`Runner.Run` returns an error for invalid execution setup, such as:

- nil runner
- nil run function
- no registered cases
- no effective checks

Per-case application failures are stored in `CaseResult.RunError`. Metric evaluation errors are converted into failing `MetricResult` values.

This lets a large suite finish and report all failures instead of stopping at the first bad case.

## Operational failures vs quality failures

Gaugo reports both operational failures and quality failures, but they mean
different things.

| Signal | Meaning | Typical response |
| --- | --- | --- |
| `Runner.Run` returns `error` | The run could not start or reporting failed. | Fix suite setup or reporter code. |
| `CaseResult.RunError != nil` | Your application could not produce an answer for that case. | Debug the system under test, dependency, timeout, or panic. |
| `MetricResult.Pass == false` with `MetricErrorInfo` | A metric could not evaluate cleanly. | Debug judge configuration, provider response, rate limit, parse failure, or timeout. |
| `MetricResult.Pass == false` without `MetricErrorInfo` | The case ran and the metric judged quality below threshold. | Treat it as a product quality failure. |

For programmatic consumers, use `MetricErrorInfo` to separate judge/provider
problems from low scores:

```go
for _, c := range result.Cases {
    if c.RunError != nil {
        log.Printf("operational case failure case=%s err=%v", c.Name, c.RunError)
        continue
    }
    for _, m := range c.Metrics {
        if m.Pass {
            continue
        }
        if info, ok := gaugo.MetricErrorInfo(m); ok {
            log.Printf("operational metric failure case=%s metric=%s kind=%s", c.Name, m.Name, info.Kind)
            continue
        }
        log.Printf("quality failure case=%s metric=%s score=%.3f reason=%s", c.Name, m.Name, m.Score, m.Reason)
    }
}
```

## Default testing reporter

`Suite.Assert` runs the default testing reporter unless a custom reporter is configured.

```go
suite := gaugo.New(t)
suite.Case("pricing", gaugo.Question("Pricing?"), gaugo.ExpectedContains("sales"))
suite.Assert(ctx, yourGaugoEvaluation)
```

The default reporter:

- reports run errors with `t.Errorf`
- reports failed metrics with case name, metric name, score, and reason
- logs metric details when present
- summarizes metric failures
- suppresses excessive failure logs after the first 100 failures
- includes safe operational metadata such as `error_kind`, provider, status code, and request id when available

## Standalone assertion

Use `gaugo.Assert` when you already have a `RunResult`.

```go
result, err := runner.Run(ctx, yourGaugoEvaluation, gaugo.AnswerRelevancy())
if err != nil {
    t.Fatalf("gaugo run: %v", err)
}
gaugo.Assert(t, result)
```

## Custom reporters

```go
type JSONReporter struct {
    W io.Writer
}

func (r JSONReporter) Report(_ context.Context, result gaugo.RunResult) {
    _ = json.NewEncoder(r.W).Encode(result)
}
```

When used with `Suite`, a custom reporter replaces the default assertion reporter:

```go
suite := gaugo.New(t, gaugo.WithReporter(JSONReporter{W: os.Stdout}))
```

If the reporter should also fail tests, capture `testing.TB` in the reporter:

```go
type TestingJSONReporter struct {
    T testing.TB
    W io.Writer
}

func (r TestingJSONReporter) Report(ctx context.Context, result gaugo.RunResult) {
    _ = json.NewEncoder(r.W).Encode(result)
    gaugo.Assert(r.T, result)
}
```

## Details

`MetricResult.Details` stores metric-specific structured details as JSON bytes. Built-in metrics use it for parsed judge output. Details are capped by `WithMetricDetailsLimit` and may be truncated.

Use details for debugging and dashboards, but do not build critical logic around provider-specific detail shapes. The default `gaugo.Assert` reporter logs safe metadata (`details_bytes` and classified error metadata) instead of raw detail payloads.
