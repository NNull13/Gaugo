# Deterministic Contract Metrics

Four checks that enforce hard contracts on the answer without any LLM calls.
Fast, stable, and suitable for every build.

## ExpectedContains

Case-level substring assertion. Not a `Metric` — it is configured on the case,
not passed to `Assert`.

```go
suite.Case("pricing",
    gaugo.Question("What is enterprise pricing?"),
    gaugo.ExpectedContains("sales"),
    gaugo.ExpectedContains("custom"),
)
```

**Scoring:** Case-insensitive substring matching. Multiple calls add multiple
required substrings. Score = `found / total`. Passes only when all substrings
are found.

Empty or whitespace-only substrings are rejected at case registration.

Use it for required product terms, compliance phrases, legal disclaimers, and
no-regression smoke tests.

## ExpectedRegex

Pattern matching against the answer using Go `regexp` syntax.

```go
gaugo.ExpectedRegex(`(?i)contact\s+sales`)
gaugo.ExpectedRegex(`^\{.*\}$`) // starts and ends with braces
```

**Required input:** `Answer`.

**Scoring:** `1` if the pattern matches anywhere in the answer, `0` otherwise.

**Details:** `{"pattern": "(?i)contact\\s+sales", "match": true}`.

The pattern is compiled with `regexp.Compile`. Invalid patterns cause a
compilation error at evaluation time.

## AnswerLength

Enforces rune count bounds on the answer.

```go
gaugo.AnswerLength(gaugo.WithMinLength(20), gaugo.WithMaxLength(600))
gaugo.AnswerLength(gaugo.WithMaxLength(100)) // only upper bound
gaugo.AnswerLength(gaugo.WithMinLength(50))  // only lower bound
```

**Required input:** `Answer`.

**Required options:** At least one of `WithMinLength` or `WithMaxLength`.

**Scoring:** `1` if the rune count is within bounds, `0` otherwise. Counts
Unicode runes, not bytes.

**Details:** `{"length": 142}`.

Constraint: `min_length` must be `<= max_length` when both are set.

## Latency

Enforces a wall-time budget on the `RunFunc` execution.

```go
gaugo.Latency(gaugo.WithMaxLatency(500 * time.Millisecond))
```

**Required input:** `Elapsed` (measured automatically by the runner).

**Required option:** `WithMaxLatency(time.Duration)`.

**Scoring:** `1` if `elapsed <= max`, `0` otherwise.

**Details:** `{"elapsed_ms": 320.5, "max_ms": 500.0}`.

Use it to catch performance regressions. Set the budget based on your p99
target, not the average.

## Example

```go
suite.Case("support answer",
    gaugo.Question("How do I reset my password?"),
    gaugo.ExpectedContains("reset"),
    gaugo.ExpectedContains("email"),
)

suite.Assert(ctx, yourFunc,
    gaugo.ExpectedRegex(`(?i)(click|tap).*link`),
    gaugo.AnswerLength(gaugo.WithMinLength(20), gaugo.WithMaxLength(500)),
    gaugo.Latency(gaugo.WithMaxLatency(2*time.Second)),
)
```
