# Dashboards and Pipelines

## When to use this

Use this example when Gaugo results need to feed a CLI, dashboard, build artifact, database, or evaluation pipeline instead of only `go test`.

## Programmatic runner

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "time"

    "github.com/nnull13/gaugo"
)

func main() {
    if err := run(context.Background()); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(ctx context.Context) error {
    runner, err := gaugo.NewRunner(
        gaugo.WithParallelism(8),
        gaugo.WithCaseTimeout(10*time.Second),
    )
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

    return json.NewEncoder(os.Stdout).Encode(result)
}

func yourGaugoEvaluation(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
    return gaugo.Output{Answer: "Enterprise pricing is custom. Contact sales."}, nil
}
```

## Compute an exit code

```go
func failed(result gaugo.RunResult) bool {
    for _, c := range result.Cases {
        if c.RunError != nil {
            return true
        }
        for _, m := range c.Metrics {
            if !m.Pass {
                return true
            }
        }
    }
    return false
}
```

## Flatten results

```go
type Row struct {
    Case    string  `json:"case"`
    Metric  string  `json:"metric"`
    Score   float64 `json:"score"`
    Pass    bool    `json:"pass"`
    Reason  string  `json:"reason"`
    Elapsed string  `json:"elapsed"`
}

func flatten(result gaugo.RunResult) []Row {
    rows := make([]Row, 0)
    for _, c := range result.Cases {
        if c.RunError != nil {
            rows = append(rows, Row{
                Case:    c.Name,
                Metric:  "Run",
                Score:   0,
                Pass:    false,
                Reason:  c.RunError.Error(),
                Elapsed: c.Elapsed.String(),
            })
            continue
        }
        for _, m := range c.Metrics {
            rows = append(rows, Row{
                Case:    c.Name,
                Metric:  m.Name,
                Score:   m.Score,
                Pass:    m.Pass,
                Reason:  m.Reason,
                Elapsed: c.Elapsed.String(),
            })
        }
    }
    return rows
}
```

## Tips

- Use `Runner` rather than `Suite` for non-test workflows.
- Keep case names stable so dashboards can track trends.
- Store `Elapsed` for latency regression checks.
- Store `Reason` and capped `Details` for debugging.
- Run deterministic checks in every build; run expensive LLM metrics on scheduled jobs or protected branches.
