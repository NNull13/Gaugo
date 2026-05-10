# Cases and Inputs

## When to use this

Use this reference when you are defining evaluation cases, building the input passed to your system, or choosing which deterministic checks belong in a case.

For the bigger mental model, see [Concepts](../concepts.md). For the `go test` workflow, see [Testing with Suite](../guides/testing-with-suite.md).

## Core types

```go
type Document struct {
    ID   string
    Text string
}

type Input struct {
    Question string
    Context  []Document
}

type Output struct {
    Answer string
}

type Case struct {
    Name     string
    Input    Input
    Expected Expected
}

type Expected struct {
    Contains []string
}
```

`Input` is what Gaugo passes to your `RunFunc`. Your application reads `Input.Question` and, when relevant, `Input.Context`, then returns an `Output` with the final answer to evaluate.

## Register cases

With `Suite`:

```go
suite := gaugo.New(t)

suite.Case("enterprise pricing",
    gaugo.Question("How does enterprise pricing work?"),
    gaugo.ContextDocs(
        gaugo.Document{ID: "pricing.md", Text: "Enterprise plans are custom and require sales."},
    ),
    gaugo.ExpectedContains("sales"),
)
```

With `Runner`:

```go
runner, err := gaugo.NewRunner()
if err != nil {
    return err
}

if err := runner.Case("enterprise pricing",
    gaugo.Question("How does enterprise pricing work?"),
    gaugo.ContextDocs(
        gaugo.Document{ID: "pricing.md", Text: "Enterprise plans are custom and require sales."},
    ),
    gaugo.ExpectedContains("sales"),
); err != nil {
    return err
}
```

## Case options

`Question(question string)` sets the user question. The value is trimmed, and an empty final question is invalid.

`ContextDocs(docs ...Document)` sets retrieved context documents. Gaugo copies the slice so later changes to the caller's slice do not change the registered case.

`ExpectedContains(substr string)` adds a deterministic substring assertion. You can call it more than once on the same case. Empty or whitespace-only values are invalid.

```go
suite.Case("refund policy",
    gaugo.Question("Can I get a refund after 30 days?"),
    gaugo.ContextDocs(
        gaugo.Document{ID: "refunds.md", Text: "Refunds are available for 30 days after purchase."},
    ),
    gaugo.ExpectedContains("30 days"),
    gaugo.ExpectedContains("refund"),
)
```

## Validation rules

Gaugo validates cases at registration time.

- Case names must be non-empty after trimming.
- Case names must be unique within a runner or suite.
- Questions must be non-empty after trimming.
- `ExpectedContains` values must be non-empty after trimming.
- A case may omit `ContextDocs` when the system being evaluated does not use retrieval.
- A case may omit `ExpectedContains` if you pass one or more LLM-judged metrics to `Assert` or `Run`.

`Suite.Case` fails the test immediately on invalid input. `Runner.Case` returns an error.

## RunFunc contract

```go
type RunFunc func(ctx context.Context, in gaugo.Input) (gaugo.Output, error)
```

Your run function should be a thin adapter around the system under evaluation. It should honor `ctx`, return errors for system failures, and avoid doing metric or assertion work itself.

```go
func yourGaugoEvaluation(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
    answer, err := answerer.Answer(ctx, answerer.Request{
        Question: in.Question,
        Context:  convertDocs(in.Context),
    })
    if err != nil {
        return gaugo.Output{}, err
    }
    return gaugo.Output{Answer: answer}, nil
}
```

## Tips

- Keep case names stable. They appear in `RunResult`, test failures, dashboards, and custom reporters.
- Put source identifiers in `Document.ID`. It makes metric details and debugging easier.
- Use deterministic `ExpectedContains` checks for contract-level requirements, then add LLM metrics for quality dimensions.
- Keep context documents short and relevant. Long, noisy context increases provider cost and makes failures harder to interpret.
