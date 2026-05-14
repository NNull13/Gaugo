package gaugo

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

type captureReporter struct {
	result RunResult
}

func (c *captureReporter) Report(_ context.Context, result RunResult) {
	c.result = result
}

type fakeJudge struct {
	eval func(ctx context.Context, req JudgeRequest) (JudgeResponse, error)
}

func (j fakeJudge) EvaluateJSON(ctx context.Context, req JudgeRequest) (JudgeResponse, error) {
	return j.eval(ctx, req)
}

type panicMetric struct {
	name  string
	value any
}

func (m panicMetric) Name() string { return m.name }

func (m panicMetric) Evaluate(context.Context, EvalInput, Judge) (MetricResult, error) {
	panic(m.value)
}

type namePanicMetric struct {
	nameValue  any
	evalValue  any
	resultName string
}

func (m namePanicMetric) Name() string {
	panic(m.nameValue)
}

func (m namePanicMetric) Evaluate(context.Context, EvalInput, Judge) (MetricResult, error) {
	if m.evalValue != nil {
		panic(m.evalValue)
	}
	return MetricResult{Name: m.resultName, Score: 0, Pass: false, Reason: "failed"}, nil
}

type passMetric struct {
	name string
}

func (m passMetric) Name() string { return m.name }

func (m passMetric) Evaluate(context.Context, EvalInput, Judge) (MetricResult, error) {
	return MetricResult{Name: m.name, Score: 1, Pass: true, Reason: "ok"}, nil
}

type inspectMetric struct {
	name string
	eval func(ctx context.Context, in EvalInput, j Judge) (MetricResult, error)
}

func (m inspectMetric) Name() string { return m.name }

func (m inspectMetric) Evaluate(ctx context.Context, in EvalInput, j Judge) (MetricResult, error) {
	return m.eval(ctx, in, j)
}

type errorMetric struct {
	name string
	err  error
}

func (m errorMetric) Name() string { return m.name }

func (m errorMetric) Evaluate(context.Context, EvalInput, Judge) (MetricResult, error) {
	return MetricResult{}, m.err
}

type panicReporter struct {
	value any
}

func (r panicReporter) Report(context.Context, RunResult) {
	panic(r.value)
}

func TestSuiteDeterministicOrderWithParallelism(t *testing.T) {
	reporter := &captureReporter{}
	suite := New(t,
		WithReporter(reporter),
		WithParallelism(32),
		WithJudge(fakeJudge{
			eval: func(_ context.Context, req JudgeRequest) (JudgeResponse, error) {
				switch req.Metric {
				case "Faithfulness":
					return JudgeResponse{RawJSON: []byte(`{"claims":[{"text":"x","supported":true}],"reason":"ok"}`)}, nil
				case "AnswerRelevancy":
					return JudgeResponse{RawJSON: []byte(`{"score":1,"reason":"ok","issues":[]}`)}, nil
				default:
					return JudgeResponse{RawJSON: []byte(`{"score":0,"reason":"unknown metric"}`)}, nil
				}
			},
		}),
	)

	wantOrder := make([]string, 0, 250)
	for i := range 250 {
		name := "case-" + leftPad(i, 3)
		wantOrder = append(wantOrder, name)
		suite.Case(name,
			Question("q"+name),
			ContextDocs(Document{ID: "doc-" + name, Text: "supporting context"}),
			ExpectedContains("support"),
		)
	}

	suite.Assert(context.Background(),
		func(_ context.Context, in Input) (Output, error) {
			time.Sleep(time.Duration(len(in.Question)%5) * time.Millisecond)
			return Output{Answer: "support answer"}, nil
		},
		Faithfulness(),
		AnswerRelevancy(),
	)

	if len(reporter.result.Cases) != len(wantOrder) {
		t.Fatalf("result cases len=%d want=%d", len(reporter.result.Cases), len(wantOrder))
	}
	for i := range reporter.result.Cases {
		if reporter.result.Cases[i].Name != wantOrder[i] {
			t.Fatalf("case order mismatch at index %d: got=%q want=%q", i, reporter.result.Cases[i].Name, wantOrder[i])
		}
	}
}

func TestSuiteCancellationPropagation(t *testing.T) {
	reporter := &captureReporter{}
	suite := New(t,
		WithReporter(reporter),
		WithParallelism(8),
		WithJudge(fakeJudge{
			eval: func(_ context.Context, _ JudgeRequest) (JudgeResponse, error) {
				return JudgeResponse{RawJSON: []byte(`{"score":1,"reason":"ok"}`)}, nil
			},
		}),
	)

	for i := range 25 {
		suite.Case("case-"+leftPad(i, 2), Question("what?"))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	suite.Assert(ctx,
		func(ctx context.Context, _ Input) (Output, error) {
			<-ctx.Done()
			return Output{}, ctx.Err()
		},
		AnswerRelevancy(),
	)

	for _, c := range reporter.result.Cases {
		if c.RunError == nil {
			t.Fatalf("expected run error for case %q after cancellation", c.Name)
		}
	}
}

func TestMetricMalformedJudgeJSONFailsDeterministically(t *testing.T) {
	reporter := &captureReporter{}
	suite := New(t,
		WithReporter(reporter),
		WithParallelism(1),
		WithJudge(fakeJudge{
			eval: func(_ context.Context, _ JudgeRequest) (JudgeResponse, error) {
				// "extra" is rejected by strict parsing.
				return JudgeResponse{RawJSON: []byte(`{"claims":[],"reason":"ok","extra":1}`)}, nil
			},
		}),
	)

	suite.Case("invalid-json-shape",
		Question("Is this faithful?"),
		ContextDocs(Document{ID: "d1", Text: "Only one fact."}),
	)

	suite.Assert(context.Background(),
		func(_ context.Context, _ Input) (Output, error) {
			return Output{Answer: "Only one fact."}, nil
		},
		Faithfulness(),
	)

	got := reporter.result.Cases[0].Metrics
	if len(got) != 1 {
		t.Fatalf("metric count got=%d want=1", len(got))
	}
	if got[0].Pass {
		t.Fatalf("expected malformed judge response to fail metric")
	}
	if !strings.Contains(got[0].Reason, "faithfulness parse failed") {
		t.Fatalf("unexpected reason: %q", got[0].Reason)
	}
}

func TestExpectedContainsProducesFailureMetric(t *testing.T) {
	reporter := &captureReporter{}
	suite := New(t, WithReporter(reporter), WithParallelism(1))

	suite.Case("contains-check",
		Question("Where can I buy enterprise?"),
		ExpectedContains("contact sales"),
	)

	suite.Assert(context.Background(),
		func(_ context.Context, _ Input) (Output, error) {
			return Output{Answer: "No details available."}, nil
		},
	)

	got := reporter.result.Cases[0].Metrics
	if len(got) != 1 {
		t.Fatalf("metric count got=%d want=1", len(got))
	}
	if got[0].Name != "ExpectedContains" {
		t.Fatalf("metric name got=%q want=%q", got[0].Name, "ExpectedContains")
	}
	if got[0].Pass {
		t.Fatalf("expected contains metric to fail when substring is missing")
	}
}

func TestExpectedAnswerAndInstructionsReachMetrics(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(WithParallelism(1))
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if err := r.Case("expected-fields",
		Question("Q?"),
		ExpectedAnswer("  reference answer  "),
		ExpectedInstructions("  answer tersely  "),
	); err != nil {
		t.Fatalf("Case error: %v", err)
	}

	result, err := r.Run(context.Background(), func(context.Context, Input) (Output, error) {
		time.Sleep(time.Millisecond)
		return Output{Answer: "actual answer"}, nil
	}, inspectMetric{
		name: "InspectExpected",
		eval: func(_ context.Context, in EvalInput, _ Judge) (MetricResult, error) {
			if in.Expected.Answer != "reference answer" {
				t.Fatalf("Expected.Answer got=%q want=reference answer", in.Expected.Answer)
			}
			if in.Expected.Instructions != "answer tersely" {
				t.Fatalf("Expected.Instructions got=%q want=answer tersely", in.Expected.Instructions)
			}
			if in.Elapsed <= 0 {
				t.Fatalf("Elapsed got=%v want positive duration", in.Elapsed)
			}
			return MetricResult{Name: "InspectExpected", Score: 1, Pass: true, Reason: "ok"}, nil
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(result.Cases) != 1 || len(result.Cases[0].Metrics) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRunnerDeterministicJSONMetrics(t *testing.T) {
	t.Parallel()

	reporter := &captureReporter{}
	suite := New(t, WithReporter(reporter), WithParallelism(1))
	suite.Case("json-output",
		Question("Return user JSON"),
	)

	schema := json.RawMessage(`{
		"type":"object",
		"required":["name"],
		"properties":{"name":{"type":"string"}}
	}`)
	suite.Assert(context.Background(),
		func(context.Context, Input) (Output, error) {
			return Output{Answer: `{"name":"Ada","ok":true}`}, nil
		},
		JSONValidity(),
		SchemaCompliance(WithSchema(schema)),
		ExpectedJSON(WithExpectedFields(map[string]any{"name": "Ada"})),
		Latency(WithMaxLatency(time.Second)),
		AnswerLength(WithMinLength(10), WithMaxLength(64)),
		ExpectedRegex(`"name"`),
	)

	got := reporter.result.Cases[0].Metrics
	if len(got) != 6 {
		t.Fatalf("metric count got=%d want=6", len(got))
	}
	for _, metric := range got {
		if !metric.Pass {
			t.Fatalf("metric %s expected pass, reason: %q", metric.Name, metric.Reason)
		}
	}
}

func TestRunnerRunProgrammatic(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(WithParallelism(1))
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if err := r.Case("programmatic", Question("Q?"), ExpectedContains("answer")); err != nil {
		t.Fatalf("Case error: %v", err)
	}

	result, err := r.Run(context.Background(), func(context.Context, Input) (Output, error) {
		return Output{Answer: "answer"}, nil
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(result.Cases) != 1 || len(result.Cases[0].Metrics) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRunnerRejectsVacuousNilMetrics(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(WithParallelism(1))
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if err := r.Case("vacuous", Question("Q?")); err != nil {
		t.Fatalf("Case error: %v", err)
	}

	_, err = r.Run(context.Background(), func(context.Context, Input) (Output, error) {
		return Output{Answer: "A"}, nil
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "no effective metrics") {
		t.Fatalf("expected vacuous metrics error, got %v", err)
	}
}

func TestRunnerRejectsMixedSuiteCaseWithoutEffectiveChecks(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(WithParallelism(1))
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if err := r.Case("checked", Question("Q?"), ExpectedContains("answer")); err != nil {
		t.Fatalf("Case error: %v", err)
	}
	if err := r.Case("unchecked", Question("Q?")); err != nil {
		t.Fatalf("Case error: %v", err)
	}

	_, err = r.Run(context.Background(), func(context.Context, Input) (Output, error) {
		return Output{Answer: "answer"}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "unchecked") {
		t.Fatalf("expected unchecked case error, got %v", err)
	}
}

func TestRunnerStopsMetricLoopAfterTimeout(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(WithParallelism(1), WithCaseTimeout(20*time.Millisecond))
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if err := r.Case("metric-timeout", Question("Q?")); err != nil {
		t.Fatalf("Case error: %v", err)
	}

	secondCalled := false
	result, err := r.Run(context.Background(), func(context.Context, Input) (Output, error) {
		return Output{Answer: "answer"}, nil
	},
		inspectMetric{name: "slow", eval: func(ctx context.Context, _ EvalInput, _ Judge) (MetricResult, error) {
			<-ctx.Done()
			return MetricResult{}, ctx.Err()
		}},
		inspectMetric{name: "second", eval: func(context.Context, EvalInput, Judge) (MetricResult, error) {
			secondCalled = true
			return MetricResult{Name: "second", Score: 1, Pass: true}, nil
		}},
	)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if secondCalled {
		t.Fatalf("second metric should not run after timeout")
	}
	if len(result.Cases) != 1 || len(result.Cases[0].Metrics) != 1 {
		t.Fatalf("expected exactly one metric failure, got %+v", result)
	}
	got := result.Cases[0].Metrics[0]
	if got.Name != "slow" || got.Pass || !strings.Contains(got.Reason, "context deadline exceeded") {
		t.Fatalf("unexpected timeout metric result: %+v", got)
	}
}

func TestRunnerRejectsDuplicateCaseNames(t *testing.T) {
	t.Parallel()

	r, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if err := r.Case("duplicate", Question("Q?")); err != nil {
		t.Fatalf("first Case error: %v", err)
	}
	if err := r.Case("duplicate", Question("Q?")); err == nil {
		t.Fatalf("expected duplicate case error")
	}
}

func TestInvalidOptionFailsEarly(t *testing.T) {
	t.Parallel()

	if _, err := NewRunner(WithParallelism(0)); err == nil {
		t.Fatalf("expected invalid parallelism error")
	}
	if _, err := NewRunner(WithMetricDetailsLimit(-1)); err == nil {
		t.Fatalf("expected invalid details limit error")
	}
	if _, err := NewRunner(WithCaseTimeout(-time.Second)); err == nil {
		t.Fatalf("expected invalid timeout error")
	}
}

func TestNewNilPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic")
		}
		if got := r.(string); got != "gaugo.New: testing.TB cannot be nil" {
			t.Fatalf("panic message got=%q", got)
		}
	}()

	New(nil)
}

func TestWithMetricDetailsLimit(t *testing.T) {
	t.Parallel()

	reporter := &captureReporter{}
	suite := New(t,
		WithReporter(reporter),
		WithParallelism(1),
		WithMetricDetailsLimit(48),
		WithJudge(fakeJudge{
			eval: func(_ context.Context, _ JudgeRequest) (JudgeResponse, error) {
				return JudgeResponse{RawJSON: []byte(`{"score":0.8,"reason":"ok","issues":["issue-1","issue-2","issue-3","issue-4"]}`)}, nil
			},
		}),
	)

	suite.Case("details-limit", Question("Q?"))
	suite.Assert(context.Background(),
		func(_ context.Context, _ Input) (Output, error) {
			return Output{Answer: "A"}, nil
		},
		AnswerRelevancy(),
	)

	got := reporter.result.Cases[0].Metrics
	if len(got) != 1 {
		t.Fatalf("metric count got=%d want=1", len(got))
	}
	if len(got[0].Details) > 48 {
		t.Fatalf("details len got=%d want<=48", len(got[0].Details))
	}
}

func TestWithMetricDetailsLimitZero(t *testing.T) {
	t.Parallel()

	reporter := &captureReporter{}
	suite := New(t,
		WithReporter(reporter),
		WithParallelism(1),
		WithMetricDetailsLimit(0),
		WithJudge(fakeJudge{
			eval: func(_ context.Context, _ JudgeRequest) (JudgeResponse, error) {
				return JudgeResponse{RawJSON: []byte(`{"score":0.8,"reason":"ok","issues":["issue-1"]}`)}, nil
			},
		}),
	)

	suite.Case("details-disabled", Question("Q?"))
	suite.Assert(context.Background(),
		func(_ context.Context, _ Input) (Output, error) {
			return Output{Answer: "A"}, nil
		},
		AnswerRelevancy(),
	)

	got := reporter.result.Cases[0].Metrics
	if len(got) != 1 {
		t.Fatalf("metric count got=%d want=1", len(got))
	}
	if len(got[0].Details) != 0 {
		t.Fatalf("details len got=%d want=0", len(got[0].Details))
	}
}

func TestMetricErrorsStoreClassifiedDetails(t *testing.T) {
	t.Parallel()

	reporter := &captureReporter{}
	suite := New(t,
		WithReporter(reporter),
		WithParallelism(1),
	)
	suite.Case("metric-error", Question("Q?"))

	suite.Assert(context.Background(),
		func(context.Context, Input) (Output, error) {
			return Output{Answer: "A"}, nil
		},
		errorMetric{name: "RateLimitMetric", err: classifiedTestError{}},
	)

	got := reporter.result.Cases[0].Metrics[0]
	if got.Pass {
		t.Fatalf("metric error should fail")
	}
	info, ok := MetricErrorInfo(got)
	if !ok {
		t.Fatalf("expected classified metric details, got %q", string(got.Details))
	}
	if info.Kind != ErrorKindProviderRateLimit || info.Provider != "anthropic" || info.StatusCode != 429 {
		t.Fatalf("unexpected error info: %+v", info)
	}
}

func TestMetricErrorDetailsRespectLimitZero(t *testing.T) {
	t.Parallel()

	reporter := &captureReporter{}
	suite := New(t,
		WithReporter(reporter),
		WithParallelism(1),
		WithMetricDetailsLimit(0),
	)
	suite.Case("metric-error-no-details", Question("Q?"))

	suite.Assert(context.Background(),
		func(context.Context, Input) (Output, error) {
			return Output{Answer: "A"}, nil
		},
		errorMetric{name: "RateLimitMetric", err: classifiedTestError{}},
	)

	got := reporter.result.Cases[0].Metrics[0]
	if len(got.Details) != 0 {
		t.Fatalf("details len got=%d want=0", len(got.Details))
	}
}

func TestRunnerContainsRunFuncPanics(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(WithParallelism(1))
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if err := r.Case("run-panic", Question("Q?"), ExpectedContains("answer")); err != nil {
		t.Fatalf("Case error: %v", err)
	}

	result, err := r.Run(context.Background(), func(context.Context, Input) (Output, error) {
		panic("secret-token-run")
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(result.Cases) != 1 {
		t.Fatalf("case count got=%d want=1", len(result.Cases))
	}
	if result.Cases[0].RunError == nil {
		t.Fatalf("expected run panic to be converted into run error")
	}
	if !strings.Contains(result.Cases[0].RunError.Error(), "run function panicked") {
		t.Fatalf("unexpected run error: %v", result.Cases[0].RunError)
	}
	if !strings.Contains(result.Cases[0].RunError.Error(), "type=string") {
		t.Fatalf("panic type should be retained: %v", result.Cases[0].RunError)
	}
	if strings.Contains(result.Cases[0].RunError.Error(), "secret-token-run") {
		t.Fatalf("panic value should be redacted: %v", result.Cases[0].RunError)
	}
}

func TestRunnerContainsMetricPanics(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(WithParallelism(1))
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if err := r.Case("metric-panic", Question("Q?")); err != nil {
		t.Fatalf("Case error: %v", err)
	}

	result, err := r.Run(context.Background(),
		func(context.Context, Input) (Output, error) {
			return Output{Answer: "A"}, nil
		},
		panicMetric{name: "PanicMetric", value: "secret-token-metric"},
		passMetric{name: "PassMetric"},
	)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(result.Cases) != 1 {
		t.Fatalf("case count got=%d want=1", len(result.Cases))
	}
	got := result.Cases[0].Metrics
	if len(got) != 2 {
		t.Fatalf("metric count got=%d want=2", len(got))
	}
	if got[0].Name != "PanicMetric" || got[0].Pass {
		t.Fatalf("panic metric result unexpected: %+v", got[0])
	}
	if !strings.Contains(got[0].Reason, `metric "PanicMetric" panicked`) {
		t.Fatalf("unexpected panic metric reason: %q", got[0].Reason)
	}
	if !strings.Contains(got[0].Reason, "type=string") {
		t.Fatalf("panic type should be retained: %q", got[0].Reason)
	}
	if strings.Contains(got[0].Reason, "secret-token-metric") {
		t.Fatalf("panic value should be redacted: %q", got[0].Reason)
	}
	if got[1].Name != "PassMetric" || !got[1].Pass {
		t.Fatalf("second metric should still execute after panic: %+v", got[1])
	}
}

func TestRunnerRedactsMetricNamePanics(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(WithParallelism(1))
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if err := r.Case("metric-name-panic", Question("Q?")); err != nil {
		t.Fatalf("Case error: %v", err)
	}

	result, err := r.Run(context.Background(),
		func(context.Context, Input) (Output, error) {
			return Output{Answer: "A"}, nil
		},
		namePanicMetric{
			nameValue: "secret-token-name",
			evalValue: "secret-token-eval",
		},
	)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(result.Cases) != 1 || len(result.Cases[0].Metrics) != 1 {
		t.Fatalf("unexpected result shape: %+v", result)
	}

	got := result.Cases[0].Metrics[0]
	if !strings.Contains(got.Name, "value=redacted") {
		t.Fatalf("fallback metric name should redact panic value: %q", got.Name)
	}
	if strings.Contains(got.Name, "secret-token-name") || strings.Contains(got.Reason, "secret-token-name") {
		t.Fatalf("metric name panic value should be redacted: name=%q reason=%q", got.Name, got.Reason)
	}
	if strings.Contains(got.Reason, "secret-token-eval") {
		t.Fatalf("metric evaluate panic value should be redacted: %q", got.Reason)
	}
}

func TestRunnerContainsReporterPanics(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(
		WithParallelism(1),
		WithReporter(panicReporter{value: "secret-token-reporter"}),
	)
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if err := r.Case("reporter-panic", Question("Q?"), ExpectedContains("answer")); err != nil {
		t.Fatalf("Case error: %v", err)
	}

	result, err := r.Run(context.Background(), func(context.Context, Input) (Output, error) {
		return Output{Answer: "answer"}, nil
	})
	if err == nil {
		t.Fatalf("expected reporter panic to be returned as error")
	}
	if !strings.Contains(err.Error(), "reporter panicked") {
		t.Fatalf("unexpected reporter error: %v", err)
	}
	if !strings.Contains(err.Error(), "type=string") {
		t.Fatalf("panic type should be retained: %v", err)
	}
	if strings.Contains(err.Error(), "secret-token-reporter") {
		t.Fatalf("panic value should be redacted: %v", err)
	}
	if len(result.Cases) != 1 {
		t.Fatalf("case count got=%d want=1", len(result.Cases))
	}
}

func TestRunnerRejectsEmptyExpectedContains(t *testing.T) {
	t.Parallel()

	inputs := []string{"", " ", "\n\t"}
	for i, needle := range inputs {
		r, err := NewRunner(WithParallelism(1))
		if err != nil {
			t.Fatalf("NewRunner error: %v", err)
		}
		err = r.Case("invalid-contains-"+strconv.Itoa(i), Question("Q?"), ExpectedContains(needle))
		if err == nil {
			t.Fatalf("expected ExpectedContains(%q) to fail validation", needle)
		}
		if !strings.Contains(err.Error(), "ExpectedContains") {
			t.Fatalf("unexpected validation error: %v", err)
		}
	}
}

func TestRunnerRunNilContext(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(WithParallelism(1))
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if err := r.Case("nil-ctx", Question("Q?"), ExpectedContains("answer")); err != nil {
		t.Fatalf("Case error: %v", err)
	}

	//nolint:staticcheck // intentionally testing nil context fallback
	result, err := r.Run(nil, func(_ context.Context, _ Input) (Output, error) {
		return Output{Answer: "answer"}, nil
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(result.Cases) != 1 {
		t.Fatalf("case count got=%d want=1", len(result.Cases))
	}
	if result.Cases[0].RunError != nil {
		t.Fatalf("unexpected run error: %v", result.Cases[0].RunError)
	}
}

func TestCaseTimeoutFires(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(WithParallelism(1), WithCaseTimeout(50*time.Millisecond))
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if err := r.Case("timeout-case", Question("Q?"), ExpectedContains("A")); err != nil {
		t.Fatalf("Case error: %v", err)
	}

	result, err := r.Run(context.Background(), func(ctx context.Context, _ Input) (Output, error) {
		select {
		case <-ctx.Done():
			return Output{}, ctx.Err()
		case <-time.After(500 * time.Millisecond):
			return Output{Answer: "A"}, nil
		}
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(result.Cases) != 1 {
		t.Fatalf("case count got=%d want=1", len(result.Cases))
	}
	if result.Cases[0].RunError == nil {
		t.Fatalf("expected run error due to timeout")
	}
	if !strings.Contains(result.Cases[0].RunError.Error(), "context deadline exceeded") {
		t.Fatalf("expected deadline error, got: %v", result.Cases[0].RunError)
	}
}

func TestLimitDetailsEdgeCases(t *testing.T) {
	t.Parallel()

	suffix := detailsTruncatedSuffix
	suffixLen := len(suffix)

	// max exactly equal to suffix length
	data := []byte("abcdefghijklmnopqrstuvwxyz")
	got := limitDetails(data, suffixLen)
	if len(got) != suffixLen {
		t.Fatalf("max=suffixLen: len got=%d want=%d", len(got), suffixLen)
	}
	// When max <= suffixLen, it just copies the first max bytes without suffix
	if string(got) != string(data[:suffixLen]) {
		t.Fatalf("max=suffixLen: got=%q want=%q", got, data[:suffixLen])
	}

	// max smaller than suffix
	got = limitDetails(data, 5)
	if len(got) != 5 {
		t.Fatalf("max=5: len got=%d want=5", len(got))
	}
	if string(got) != string(data[:5]) {
		t.Fatalf("max=5: got=%q want=%q", got, data[:5])
	}

	// max larger than data (no truncation)
	got = limitDetails(data, 100)
	if string(got) != string(data) {
		t.Fatalf("max=100: got=%q want=%q", got, data)
	}

	// empty details with positive max
	got = limitDetails(nil, 100)
	if got != nil {
		t.Fatalf("empty details: got=%v want=nil", got)
	}
	got = limitDetails([]byte{}, 100)
	if len(got) != 0 {
		t.Fatalf("empty details: len got=%d want=0", len(got))
	}
}

func TestCompactMetricsAllNil(t *testing.T) {
	t.Parallel()

	got := compactMetrics([]Metric{nil, nil, nil})
	if len(got) != 0 {
		t.Fatalf("compactMetrics all nil: len got=%d want=0", len(got))
	}
}

func TestExpectedContainsCaseInsensitive(t *testing.T) {
	t.Parallel()

	reporter := &captureReporter{}
	suite := New(t, WithReporter(reporter), WithParallelism(1))

	suite.Case("case-insensitive", Question("Q?"), ExpectedContains("HELLO"))
	suite.Assert(context.Background(),
		func(_ context.Context, _ Input) (Output, error) {
			return Output{Answer: "hello world"}, nil
		},
	)

	got := reporter.result.Cases[0].Metrics
	if len(got) != 1 {
		t.Fatalf("metric count got=%d want=1", len(got))
	}
	if !got[0].Pass {
		t.Fatalf("expected case-insensitive match to pass, reason: %q", got[0].Reason)
	}
}

func TestRunnerRunNilRunner(t *testing.T) {
	t.Parallel()

	var r *Runner
	_, err := r.Run(context.Background(), func(context.Context, Input) (Output, error) {
		return Output{}, nil
	})
	if err == nil {
		t.Fatalf("expected error for nil runner")
	}
}

func TestRunnerCaseNilRunner(t *testing.T) {
	t.Parallel()

	var r *Runner
	err := r.Case("test", Question("Q?"))
	if err == nil {
		t.Fatalf("expected error for nil runner")
	}
}

func TestRunResultFailed(t *testing.T) {
	t.Parallel()

	// All pass
	rr := RunResult{Cases: []CaseResult{
		{Metrics: []MetricResult{{Pass: true}, {Pass: true}}},
	}}
	if rr.Failed() {
		t.Fatalf("expected not failed when all pass")
	}

	// A metric fails
	rr = RunResult{Cases: []CaseResult{
		{Metrics: []MetricResult{{Pass: true}, {Pass: false}}},
	}}
	if !rr.Failed() {
		t.Fatalf("expected failed when a metric fails")
	}

	// RunError is set
	rr = RunResult{Cases: []CaseResult{
		{RunError: errors.New("boom")},
	}}
	if !rr.Failed() {
		t.Fatalf("expected failed when RunError is set")
	}

	// Empty result
	rr = RunResult{}
	if rr.Failed() {
		t.Fatalf("expected not failed for empty result")
	}
}

func TestRunResultPassRate(t *testing.T) {
	t.Parallel()

	// Mix of pass/fail
	rr := RunResult{Cases: []CaseResult{
		{Metrics: []MetricResult{{Pass: true}, {Pass: false}}},
		{Metrics: []MetricResult{{Pass: true}}},
	}}
	got := rr.PassRate()
	// 2 passed out of 3 total
	want := 2.0 / 3.0
	if got < want-0.001 || got > want+0.001 {
		t.Fatalf("PassRate got=%f want=%f", got, want)
	}

	// Empty result
	rr = RunResult{}
	if rr.PassRate() != 0 {
		t.Fatalf("empty PassRate got=%f want=0", rr.PassRate())
	}

	// Case with RunError counts as zero
	rr = RunResult{Cases: []CaseResult{
		{RunError: errors.New("boom")},
		{Metrics: []MetricResult{{Pass: true}}},
	}}
	got = rr.PassRate()
	// 1 passed out of 2 total (run error counts as 1 total, 0 passed)
	want = 0.5
	if got < want-0.001 || got > want+0.001 {
		t.Fatalf("PassRate with RunError got=%f want=%f", got, want)
	}
}

func TestRunResultSummary(t *testing.T) {
	t.Parallel()

	rr := RunResult{Cases: []CaseResult{
		{Metrics: []MetricResult{{Pass: true}, {Pass: false}}},
		{RunError: errors.New("boom")},
	}}
	summary := rr.Summary()
	if !strings.Contains(summary, "cases=2") {
		t.Fatalf("summary missing cases=2: %q", summary)
	}
	if !strings.Contains(summary, "failed_cases=2") {
		t.Fatalf("summary missing failed_cases=2: %q", summary)
	}
	if !strings.Contains(summary, "checks=2") {
		t.Fatalf("summary missing checks=2: %q", summary)
	}
	if !strings.Contains(summary, "failed_checks=1") {
		t.Fatalf("summary missing failed_checks=1: %q", summary)
	}
	if !strings.Contains(summary, "run_errors=1") {
		t.Fatalf("summary missing run_errors=1: %q", summary)
	}
	if !strings.Contains(summary, "pass_rate=") {
		t.Fatalf("summary missing pass_rate: %q", summary)
	}
}

func TestCaseResultFailedMetrics(t *testing.T) {
	t.Parallel()

	cr := CaseResult{Metrics: []MetricResult{
		{Name: "A", Pass: true},
		{Name: "B", Pass: false},
		{Name: "C", Pass: false},
		{Name: "D", Pass: true},
	}}
	failed := cr.FailedMetrics()
	if len(failed) != 2 {
		t.Fatalf("FailedMetrics count got=%d want=2", len(failed))
	}
	if failed[0].Name != "B" || failed[1].Name != "C" {
		t.Fatalf("FailedMetrics names got=%q,%q want=B,C", failed[0].Name, failed[1].Name)
	}
}

func TestCaseResultMetricsByName(t *testing.T) {
	t.Parallel()

	cr := CaseResult{Metrics: []MetricResult{
		{Name: "Faithfulness", Pass: true},
		{Name: "AnswerRelevancy", Pass: false},
		{Name: "Faithfulness", Pass: false},
	}}
	got := cr.MetricsByName("Faithfulness")
	if len(got) != 2 {
		t.Fatalf("MetricsByName count got=%d want=2", len(got))
	}
	got = cr.MetricsByName("AnswerRelevancy")
	if len(got) != 1 {
		t.Fatalf("MetricsByName count got=%d want=1", len(got))
	}
	got = cr.MetricsByName("NotExist")
	if len(got) != 0 {
		t.Fatalf("MetricsByName count got=%d want=0", len(got))
	}
}

func TestDocHelper(t *testing.T) {
	t.Parallel()

	d := Doc("my-id", "my-text")
	if d.ID != "my-id" {
		t.Fatalf("Doc ID got=%q want=%q", d.ID, "my-id")
	}
	if d.Text != "my-text" {
		t.Fatalf("Doc Text got=%q want=%q", d.Text, "my-text")
	}
}

func TestRunnerRunNilRunFunc(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(WithParallelism(1))
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if err := r.Case("nil-run", Question("Q?"), ExpectedContains("A")); err != nil {
		t.Fatalf("Case error: %v", err)
	}

	_, err = r.Run(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error for nil run func")
	}
}

func TestRunnerRunNoCases(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(WithParallelism(1))
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}

	_, err = r.Run(context.Background(), func(context.Context, Input) (Output, error) {
		return Output{}, nil
	}, passMetric{name: "M"})
	if err == nil {
		t.Fatalf("expected error for no cases")
	}
	if !strings.Contains(err.Error(), "no cases registered") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWithJudgeOption(t *testing.T) {
	t.Parallel()

	j := fakeJudge{eval: func(context.Context, JudgeRequest) (JudgeResponse, error) {
		return JudgeResponse{}, nil
	}}
	r, err := NewRunner(WithJudge(j))
	if err != nil {
		t.Fatalf("NewRunner with judge error: %v", err)
	}
	if r == nil {
		t.Fatalf("expected non-nil runner")
	}
}

func TestWithReporterNilFails(t *testing.T) {
	t.Parallel()

	_, err := NewRunner(WithReporter(nil))
	if err == nil {
		t.Fatalf("expected error for nil reporter")
	}
}

func TestNilOptionSkipped(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(nil, WithParallelism(2), nil)
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if r == nil {
		t.Fatalf("expected non-nil runner")
	}
}

func TestCaseValidation(t *testing.T) {
	t.Parallel()

	r, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}

	// Empty name
	if err := r.Case("", Question("Q?")); err == nil {
		t.Fatalf("expected error for empty name")
	}
	// Whitespace name
	if err := r.Case("  ", Question("Q?")); err == nil {
		t.Fatalf("expected error for whitespace name")
	}
	// Empty question
	if err := r.Case("valid-name"); err == nil {
		t.Fatalf("expected error for missing question")
	}
}

func TestContextDocsOption(t *testing.T) {
	t.Parallel()

	reporter := &captureReporter{}
	suite := New(t, WithReporter(reporter), WithParallelism(1),
		WithJudge(fakeJudge{
			eval: func(_ context.Context, req JudgeRequest) (JudgeResponse, error) {
				if len(req.ContextDocs) != 2 {
					return JudgeResponse{}, errors.New("expected 2 context docs")
				}
				return JudgeResponse{RawJSON: []byte(`{"claims":[{"text":"c","supported":true}],"reason":"ok"}`)}, nil
			},
		}),
	)

	suite.Case("with-docs",
		Question("Q?"),
		ContextDocs(Doc("d1", "text1"), Doc("d2", "text2")),
	)
	suite.Assert(context.Background(),
		func(_ context.Context, in Input) (Output, error) {
			if len(in.Context) != 2 {
				return Output{}, errors.New("expected 2 context docs in input")
			}
			return Output{Answer: "A"}, nil
		},
		Faithfulness(),
	)

	got := reporter.result.Cases[0].Metrics
	if len(got) != 1 {
		t.Fatalf("metric count got=%d want=1", len(got))
	}
	if !got[0].Pass {
		t.Fatalf("expected pass, reason: %q", got[0].Reason)
	}
}

func TestExpectedContainsMultiple(t *testing.T) {
	t.Parallel()

	reporter := &captureReporter{}
	suite := New(t, WithReporter(reporter), WithParallelism(1))

	suite.Case("multi-contains",
		Question("Q?"),
		ExpectedContains("hello"),
		ExpectedContains("world"),
	)
	suite.Assert(context.Background(),
		func(_ context.Context, _ Input) (Output, error) {
			return Output{Answer: "hello world"}, nil
		},
	)

	got := reporter.result.Cases[0].Metrics
	if len(got) != 1 {
		t.Fatalf("metric count got=%d want=1", len(got))
	}
	if !got[0].Pass {
		t.Fatalf("expected pass with both substrings present")
	}
	if got[0].Score != 1.0 {
		t.Fatalf("score got=%f want=1.0", got[0].Score)
	}
}

func TestExpectedContainsPartialMissing(t *testing.T) {
	t.Parallel()

	reporter := &captureReporter{}
	suite := New(t, WithReporter(reporter), WithParallelism(1))

	suite.Case("partial-contains",
		Question("Q?"),
		ExpectedContains("hello"),
		ExpectedContains("missing"),
	)
	suite.Assert(context.Background(),
		func(_ context.Context, _ Input) (Output, error) {
			return Output{Answer: "hello world"}, nil
		},
	)

	got := reporter.result.Cases[0].Metrics
	if len(got) != 1 {
		t.Fatalf("metric count got=%d want=1", len(got))
	}
	if got[0].Pass {
		t.Fatalf("expected fail with one missing substring")
	}
	if got[0].Score != 0.5 {
		t.Fatalf("score got=%f want=0.5", got[0].Score)
	}
}

func TestRunFuncReturnsError(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(WithParallelism(1))
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if err := r.Case("run-error", Question("Q?"), ExpectedContains("A")); err != nil {
		t.Fatalf("Case error: %v", err)
	}

	result, err := r.Run(context.Background(), func(context.Context, Input) (Output, error) {
		return Output{}, errors.New("system failure")
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Cases[0].RunError == nil {
		t.Fatalf("expected run error")
	}
	if !strings.Contains(result.Cases[0].RunError.Error(), "system failure") {
		t.Fatalf("unexpected error: %v", result.Cases[0].RunError)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultRetryConfig()
	if cfg.MaxAttempts != 3 {
		t.Fatalf("MaxAttempts got=%d want=3", cfg.MaxAttempts)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
}

func TestRetryConfigValidation(t *testing.T) {
	t.Parallel()

	if err := (RetryConfig{MaxAttempts: -1}).Validate(); err == nil {
		t.Fatalf("expected error for negative MaxAttempts")
	}
	if err := (RetryConfig{BaseDelay: -1}).Validate(); err == nil {
		t.Fatalf("expected error for negative BaseDelay")
	}
	if err := (RetryConfig{MaxDelay: -1}).Validate(); err == nil {
		t.Fatalf("expected error for negative MaxDelay")
	}
	if err := (RetryConfig{BaseDelay: 5 * time.Second, MaxDelay: 1 * time.Second}).Validate(); err == nil {
		t.Fatalf("expected error when BaseDelay > MaxDelay")
	}
	if err := (RetryConfig{}).Validate(); err != nil {
		t.Fatalf("zero config should be valid: %v", err)
	}
}

func TestInferErrorKindBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		msg  string
		want ErrorKind
	}{
		{"provider response body too large", ErrorKindProviderResponse},
		{"refusal detected", ErrorKindProviderRefusal},
		{"output was truncated", ErrorKindProviderTruncated},
		{"requires a configured judge", ErrorKindMetric},
		{"threshold must be in range", ErrorKindMetric},
		{"something random", ErrorKindUnknown},
		{"request failed status=429 too many requests", ErrorKindUnknown},
		{"request failed status=401 unauthorized", ErrorKindUnknown},
		{"request failed status=500 server error", ErrorKindUnknown},
	}
	for _, tt := range tests {
		info := ClassifyError(errors.New(tt.msg))
		if info.Kind != tt.want {
			t.Fatalf("ClassifyError(%q).Kind got=%q want=%q", tt.msg, info.Kind, tt.want)
		}
	}
}

func TestClassifyErrorNil(t *testing.T) {
	t.Parallel()

	info := ClassifyError(nil)
	if info.Kind != "" {
		t.Fatalf("ClassifyError(nil).Kind got=%q want empty", info.Kind)
	}
}

func TestContextRelevancyNoJudge(t *testing.T) {
	t.Parallel()

	_, err := ContextRelevancy().Evaluate(context.Background(), EvalInput{}, nil)
	if err == nil {
		t.Fatalf("expected error for nil judge")
	}
}

func TestAnswerRelevancyNoJudge(t *testing.T) {
	t.Parallel()

	_, err := AnswerRelevancy().Evaluate(context.Background(), EvalInput{}, nil)
	if err == nil {
		t.Fatalf("expected error for nil judge")
	}
}

func TestInvalidMetricThresholdContextRelevancy(t *testing.T) {
	t.Parallel()

	_, err := ContextRelevancy(WithThreshold(-0.1)).Evaluate(context.Background(), EvalInput{}, fakeJudge{
		eval: func(context.Context, JudgeRequest) (JudgeResponse, error) {
			return JudgeResponse{RawJSON: []byte(`{"documents":[],"reason":"ok"}`)}, nil
		},
	})
	if err == nil {
		t.Fatalf("expected invalid threshold error")
	}
}

func TestInvalidMetricThresholdFaithfulness(t *testing.T) {
	t.Parallel()

	_, err := Faithfulness(WithThreshold(1.5)).Evaluate(context.Background(), EvalInput{}, fakeJudge{
		eval: func(context.Context, JudgeRequest) (JudgeResponse, error) {
			return JudgeResponse{RawJSON: []byte(`{"claims":[],"reason":"ok"}`)}, nil
		},
	})
	if err == nil {
		t.Fatalf("expected invalid threshold error")
	}
}

func TestAssertWithPassingResult(t *testing.T) {
	t.Parallel()

	result := RunResult{
		Cases: []CaseResult{
			{
				Name:    "pass-case",
				Metrics: []MetricResult{{Name: "M1", Pass: true, Score: 1, Reason: "ok"}},
			},
		},
	}
	// Assert on a passing result should not fail
	Assert(t, result)
}

func TestCompactMetricsMixed(t *testing.T) {
	t.Parallel()

	got := compactMetrics([]Metric{nil, passMetric{name: "A"}, nil, passMetric{name: "B"}, nil})
	if len(got) != 2 {
		t.Fatalf("compactMetrics mixed: len got=%d want=2", len(got))
	}
	if got[0].Name() != "A" || got[1].Name() != "B" {
		t.Fatalf("compactMetrics order wrong: got=%q,%q", got[0].Name(), got[1].Name())
	}
}

func TestCompactMetricsEmpty(t *testing.T) {
	t.Parallel()

	got := compactMetrics(nil)
	if got != nil {
		t.Fatalf("compactMetrics nil: got=%v want=nil", got)
	}
	got = compactMetrics([]Metric{})
	if got != nil {
		t.Fatalf("compactMetrics empty: got=%v want=nil", got)
	}
}

func TestCaseNilOptionSkipped(t *testing.T) {
	t.Parallel()

	r, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	err = r.Case("nil-opt", Question("Q?"), nil, ExpectedContains("A"))
	if err != nil {
		t.Fatalf("Case with nil option error: %v", err)
	}
}

func TestWithCaseTimeoutZero(t *testing.T) {
	t.Parallel()

	r, err := NewRunner(WithCaseTimeout(0))
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}
	if err := r.Case("no-timeout", Question("Q?"), ExpectedContains("A")); err != nil {
		t.Fatalf("Case error: %v", err)
	}

	result, err := r.Run(context.Background(), func(context.Context, Input) (Output, error) {
		return Output{Answer: "A"}, nil
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Cases[0].RunError != nil {
		t.Fatalf("unexpected run error: %v", result.Cases[0].RunError)
	}
}

func TestRunResultEmptySummary(t *testing.T) {
	t.Parallel()

	rr := RunResult{}
	summary := rr.Summary()
	if !strings.Contains(summary, "cases=0") {
		t.Fatalf("empty summary missing cases=0: %q", summary)
	}
}

func leftPad(v, width int) string {
	s := strconv.Itoa(v)
	if len(s) >= width {
		return s
	}
	return strings.Repeat("0", width-len(s)) + s
}
