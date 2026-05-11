package gaugo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nnull13/gaugo/internal/runner"
)

const (
	metricNameExpectedContains = "ExpectedContains"
	panicValueRedacted         = "redacted"
	componentRunFunction       = "run function"
	componentReporter          = "reporter"
	detailsTruncatedSuffix     = "...(truncated)"
)

// Runner executes registered cases and returns structured results without
// depending on the testing package.
type Runner struct {
	cfg config

	mu    sync.Mutex
	cases []Case
	names map[string]struct{}
}

// Suite is a testing wrapper around Runner.
type Suite struct {
	t      testing.TB
	runner *Runner
}

// RunFunc executes the system under test for one case.
type RunFunc func(ctx context.Context, in Input) (Output, error)

// NewRunner creates a programmatic evaluation runner.
func NewRunner(opts ...Option) (*Runner, error) {
	cfg, err := applyOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("gaugo runner config invalid: %w", err)
	}
	return &Runner{
		cfg:   cfg,
		names: make(map[string]struct{}),
	}, nil
}

// New creates a testing Suite. Configuration errors fail the test immediately.
func New(t testing.TB, opts ...Option) *Suite {
	if t == nil {
		panic("gaugo.New: testing.TB cannot be nil")
	}
	t.Helper()

	r, err := NewRunner(opts...)
	if err != nil {
		t.Fatalf("%v", err)
	}
	return &Suite{
		t:      t,
		runner: r,
	}
}

// Case registers one evaluation case.
func (r *Runner) Case(name string, opts ...CaseOption) error {
	if r == nil {
		return errors.New("runner is nil")
	}

	c := Case{Name: strings.TrimSpace(name)}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&c)
	}
	if err := validateCase(c); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.names[c.Name]; exists {
		return fmt.Errorf("case %q already registered", c.Name)
	}
	r.names[c.Name] = struct{}{}
	r.cases = append(r.cases, c)
	return nil
}

// Case registers one evaluation case and fails the test if it is invalid.
func (s *Suite) Case(name string, opts ...CaseOption) {
	s.t.Helper()
	if err := s.runner.Case(name, opts...); err != nil {
		s.t.Fatalf("gaugo case invalid: %v", err)
	}
}

// Run executes all registered cases.
func (r *Runner) Run(ctx context.Context, run RunFunc, metrics ...Metric) (RunResult, error) {
	if r == nil || run == nil {
		return RunResult{}, errors.New("runner or run function is nil")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	r.mu.Lock()
	cases := append([]Case(nil), r.cases...)
	r.mu.Unlock()

	if len(cases) == 0 {
		return RunResult{}, errors.New("no cases registered")
	}

	metrics = compactMetrics(metrics)
	if len(metrics) == 0 && !containsChecksConfigured(cases) {
		return RunResult{}, errors.New("no effective metrics provided and no ExpectedContains assertions configured")
	}

	result := RunResult{
		Cases: make([]CaseResult, len(cases)),
	}
	done := make([]bool, len(cases))

	for i, c := range cases {
		result.Cases[i].Name = c.Name
	}

	runner.RunIndexed(ctx, r.cfg.parallelism, len(cases), func(parentCtx context.Context, index int) {
		c := cases[index]
		done[index] = true
		start := time.Now()

		caseCtx := parentCtx
		cancel := func() {}
		if r.cfg.caseTimeout > 0 {
			caseCtx, cancel = context.WithTimeout(parentCtx, r.cfg.caseTimeout)
		}
		defer cancel()

		out, err := runFuncSafely(caseCtx, run, c.Input)
		if err != nil {
			result.Cases[index].RunError = fmt.Errorf("run failed: %w", err)
			result.Cases[index].Elapsed = time.Since(start)
			return
		}

		if mr, ok := expectedContainsResult(out.Answer, c.Expected); ok {
			mr.Details = limitDetails(mr.Details, r.cfg.detailsMax)
			result.Cases[index].Metrics = append(result.Cases[index].Metrics, mr)
		}

		evalIn := EvalInput{
			CaseName: c.Name,
			Input:    c.Input,
			Output:   out,
			Expected: c.Expected,
		}

		for _, metric := range metrics {
			metricName := metricNameSafely(metric)
			mr, mErr := evaluateMetricSafely(caseCtx, metricName, metric, evalIn, r.cfg.judge)
			if mErr != nil {
				mr = MetricResult{
					Name:    metricName,
					Score:   0,
					Pass:    false,
					Reason:  mErr.Error(),
					Details: errorInfoDetails(mErr),
				}
			}
			mr.Details = limitDetails(mr.Details, r.cfg.detailsMax)
			result.Cases[index].Metrics = append(result.Cases[index].Metrics, mr)
		}

		result.Cases[index].Elapsed = time.Since(start)
	})

	if ctxErr := ctx.Err(); ctxErr != nil {
		for i := range done {
			if done[i] {
				continue
			}
			result.Cases[i].RunError = fmt.Errorf("run canceled before execution: %w", ctxErr)
		}
	}

	if r.cfg.reporter != nil {
		if err := reportSafely(ctx, r.cfg.reporter, result); err != nil {
			return result, err
		}
	}
	return result, nil
}

// Assert executes all registered cases and reports failures through testing.
func (s *Suite) Assert(ctx context.Context, run RunFunc, metrics ...Metric) {
	s.t.Helper()

	result, err := s.runner.Run(ctx, run, metrics...)
	if err != nil {
		s.t.Fatalf("gaugo assert invalid: %v", err)
		return
	}
	if s.runner.cfg.reporter == nil {
		Assert(s.t, result)
	}
}

func compactMetrics(metrics []Metric) []Metric {
	if len(metrics) == 0 {
		return nil
	}
	out := make([]Metric, 0, len(metrics))
	for _, metric := range metrics {
		if metric != nil {
			out = append(out, metric)
		}
	}
	return out
}

func limitDetails(details []byte, max int) []byte {
	if max == 0 {
		return nil
	}
	if len(details) <= max {
		return details
	}

	suffixBytes := []byte(detailsTruncatedSuffix)
	if max <= len(suffixBytes) {
		trimmed := make([]byte, max)
		copy(trimmed, details[:max])
		return trimmed
	}

	trimmed := make([]byte, max)
	copy(trimmed, details[:max-len(suffixBytes)])
	copy(trimmed[max-len(suffixBytes):], suffixBytes)
	return trimmed
}

func containsChecksConfigured(cases []Case) bool {
	for _, c := range cases {
		if len(c.Expected.Contains) > 0 {
			return true
		}
	}
	return false
}

func expectedContainsResult(answer string, expected Expected) (MetricResult, bool) {
	if len(expected.Contains) == 0 {
		return MetricResult{}, false
	}
	answerLower := strings.ToLower(answer)
	matched := 0
	missing := make([]string, 0, len(expected.Contains))
	for _, needle := range expected.Contains {
		if strings.Contains(answerLower, strings.ToLower(needle)) {
			matched++
			continue
		}
		missing = append(missing, needle)
	}

	score := float64(matched) / float64(len(expected.Contains))
	pass := matched == len(expected.Contains)
	reason := "all required substrings present"
	if !pass {
		reason = fmt.Sprintf("missing expected substrings: %q", missing)
	}

	return MetricResult{
		Name:   metricNameExpectedContains,
		Score:  score,
		Pass:   pass,
		Reason: reason,
	}, true
}

func runFuncSafely(ctx context.Context, run RunFunc, in Input) (out Output, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = newPanicError(componentRunFunction, recovered)
		}
	}()
	return run(ctx, in)
}

func evaluateMetricSafely(ctx context.Context, metricName string, metric Metric, in EvalInput, j Judge) (mr MetricResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = newPanicError(fmt.Sprintf("metric %q", metricName), recovered)
		}
	}()
	return metric.Evaluate(ctx, in, j)
}

func reportSafely(ctx context.Context, reporter Reporter, result RunResult) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = newPanicError(componentReporter, recovered)
		}
	}()
	reporter.Report(ctx, result)
	return nil
}

func metricNameSafely(metric Metric) (name string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			name = fmt.Sprintf("metric<name panic: type=%T value=%s>", recovered, panicValueRedacted)
		}
	}()
	return metric.Name()
}

type panicError struct {
	Component string
	PanicType string
}

func (e *panicError) Error() string {
	return fmt.Sprintf("%s panicked: type=%s value=%s", e.Component, e.PanicType, panicValueRedacted)
}

func (e *panicError) GaugoErrorKind() string {
	return string(ErrorKindPanic)
}

func newPanicError(component string, recovered any) error {
	return &panicError{
		Component: component,
		PanicType: fmt.Sprintf("%T", recovered),
	}
}
