# Custom Reporters

## When to use this

Implement a custom reporter when you want JSON output, CI annotations, logs, dashboards, or custom failure behavior.

For result shapes, see [Results and Reporting](../reference/results-and-reporting.md).

## Interface

```go
type Reporter interface {
    Report(ctx context.Context, result gaugo.RunResult)
}
```

Reporters receive the complete result after all cases finish.

## Complete JSON reporter example

```go
package yourpkg_test

import (
    "context"
    "encoding/json"
    "io"
    "os"
    "testing"

    "github.com/nnull13/gaugo"
)

type JSONReporter struct {
    W io.Writer
}

func (r JSONReporter) Report(ctx context.Context, result gaugo.RunResult) {
    w := r.W
    if w == nil {
        w = os.Stdout
    }
    _ = json.NewEncoder(w).Encode(result)
}

func TestWithJSONReporter(t *testing.T) {
    suite := gaugo.New(t,
        gaugo.WithReporter(JSONReporter{W: os.Stdout}),
    )

    suite.Case("pricing",
        gaugo.Question("How does enterprise pricing work?"),
        gaugo.ExpectedContains("sales"),
    )

    suite.Assert(context.Background(), func(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
        return gaugo.Output{Answer: "Contact sales for enterprise pricing."}, nil
    })
}
```

Important: a custom reporter on `Suite` replaces Gaugo's default testing reporter. The example above emits JSON but does not fail the test on failed metrics.

## Reporter that also fails tests

```go
package yourpkg_test

import (
    "context"
    "encoding/json"
    "io"
    "os"
    "testing"

    "github.com/nnull13/gaugo"
)

type TestingJSONReporter struct {
    T testing.TB
    W io.Writer
}

func (r TestingJSONReporter) Report(ctx context.Context, result gaugo.RunResult) {
    w := r.W
    if w == nil {
        w = os.Stdout
    }
    _ = json.NewEncoder(w).Encode(result)
    gaugo.Assert(r.T, result)
}

func TestWithFailingJSONReporter(t *testing.T) {
    suite := gaugo.New(t,
        gaugo.WithReporter(TestingJSONReporter{T: t, W: os.Stdout}),
    )

    suite.Case("pricing",
        gaugo.Question("How does enterprise pricing work?"),
        gaugo.ExpectedContains("sales"),
    )

    suite.Assert(context.Background(), func(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
        return gaugo.Output{Answer: "Contact sales for enterprise pricing."}, nil
    })
}
```

## Programmatic alternative

For many integrations, `Runner` is simpler than `WithReporter`.

```go
result, err := runner.Run(ctx, yourGaugoEvaluation, gaugo.AnswerRelevancy())
if err != nil {
    return err
}

if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
    return err
}
```

Use `Runner` for CLIs, dashboards, batch jobs, and services. Use reporters when you want suite-level hooks inside `go test`.

## Tips

- Keep reporters non-blocking or bounded; they run after evaluation.
- Do not write secrets into reports.
- Preserve case names and metric names as stable dimensions.
- Call `gaugo.Assert` from reporters that should fail tests.
