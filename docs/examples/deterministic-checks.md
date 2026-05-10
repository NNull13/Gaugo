# Deterministic Checks

## When to use this

Use deterministic checks when you want fast, cheap, no-network evaluation with stable pass/fail behavior.

This is the best first integration path for most teams.

## Complete test

```go
package yourpkg_test

import (
    "context"
    "testing"

    "github.com/nnull13/gaugo"
)

func TestPricingAnswer(t *testing.T) {
    suite := gaugo.New(t)

    suite.Case("enterprise pricing",
        gaugo.Question("How does enterprise pricing work?"),
        gaugo.ExpectedContains("sales"),
        gaugo.ExpectedContains("custom"),
    )

    suite.Assert(context.Background(),
        func(ctx context.Context, in gaugo.Input) (gaugo.Output, error) {
            return gaugo.Output{Answer: "Enterprise pricing is custom. Contact sales for a quote."}, nil
        },
    )
}
```

Run it:

```sh
go test ./...
```

## Multiple cases

```go
suite.Case("enterprise pricing",
    gaugo.Question("How does enterprise pricing work?"),
    gaugo.ExpectedContains("sales"),
)

suite.Case("refund policy",
    gaugo.Question("Can I get a refund?"),
    gaugo.ExpectedContains("30 days"),
)
```

Cases run concurrently by default while results remain in registration order.

## Programmatic deterministic run

```go
runner, err := gaugo.NewRunner(gaugo.WithParallelism(4))
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
```

## When deterministic checks are enough

Use only deterministic checks when you need to verify:

- required strings
- legal disclaimers
- support channels
- known product names
- no-regression smoke tests
- CI gates that must be perfectly stable

## When to add LLM metrics

Add LLM metrics when correctness is semantic:

- answer is grounded in retrieved context
- answer addresses the question
- answer is complete enough
- answer avoids unsupported claims

For a RAG example, see [RAG Evaluation](rag-evaluation.md).
