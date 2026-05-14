package gaugo

import (
	"fmt"
	"strings"
	"time"
)

// RunResult is the deterministic output of a suite execution.
type RunResult struct {
	Cases []CaseResult
}

// CaseResult contains execution and metric results for a single case.
type CaseResult struct {
	Name     string
	Metrics  []MetricResult
	RunError error
	Elapsed  time.Duration
}

// Failed reports whether any case has a run error or a failing metric.
func (r RunResult) Failed() bool {
	for _, c := range r.Cases {
		if c.Failed() {
			return true
		}
	}
	return false
}

// PassRate returns passed checks divided by explicit checks plus run errors.
// Run errors count as failed executions in the denominator.
func (r RunResult) PassRate() float64 {
	checks, failedChecks, runErrors := r.counts()
	total := checks + runErrors
	passed := checks - failedChecks
	if total == 0 {
		return 0
	}
	return float64(passed) / float64(total)
}

// Summary returns a human-readable one-line summary suitable for logs and CI output.
func (r RunResult) Summary() string {
	checks, failedChecks, runErrors := r.counts()
	failedCases := r.failedCases()
	return fmt.Sprintf("cases=%d failed_cases=%d checks=%d failed_checks=%d run_errors=%d pass_rate=%.1f%%",
		len(r.Cases), failedCases, checks, failedChecks, runErrors, r.PassRate()*100)
}

func (r RunResult) counts() (checks int, failedChecks int, runErrors int) {
	for _, c := range r.Cases {
		if c.RunError != nil {
			runErrors++
			continue
		}
		for _, m := range c.Metrics {
			checks++
			if !m.Pass {
				failedChecks++
			}
		}
	}
	return checks, failedChecks, runErrors
}

func (r RunResult) failedCases() int {
	var failed int
	for _, c := range r.Cases {
		if c.Failed() {
			failed++
		}
	}
	return failed
}

// Failed reports whether this case has a run error or any failing metric.
func (c CaseResult) Failed() bool {
	if c.RunError != nil {
		return true
	}
	for _, m := range c.Metrics {
		if !m.Pass {
			return true
		}
	}
	return false
}

// FailedMetrics returns only the metrics that did not pass.
func (c CaseResult) FailedMetrics() []MetricResult {
	var out []MetricResult
	for _, m := range c.Metrics {
		if !m.Pass {
			out = append(out, m)
		}
	}
	return out
}

// MetricsByName returns metrics matching the given name.
func (c CaseResult) MetricsByName(name string) []MetricResult {
	name = strings.TrimSpace(name)
	var out []MetricResult
	for _, m := range c.Metrics {
		if m.Name == name {
			out = append(out, m)
		}
	}
	return out
}
