package gaugo

import (
	"context"
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

func leftPad(v, width int) string {
	s := strconv.Itoa(v)
	if len(s) >= width {
		return s
	}
	return strings.Repeat("0", width-len(s)) + s
}
