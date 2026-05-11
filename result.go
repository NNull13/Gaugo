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

// MetricResult represents a score and pass/fail outcome for one metric.
type MetricResult struct {
	Name    string
	Score   float64
	Pass    bool
	Reason  string
	Details []byte
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

// PassRate returns the fraction of metrics that passed across all cases, in [0,1].
// Cases with run errors count as zero passing metrics.
// Returns 0 when no metrics are present.
func (r RunResult) PassRate() float64 {
	var total, passed int
	for _, c := range r.Cases {
		if c.RunError != nil {
			total++
			continue
		}
		for _, m := range c.Metrics {
			total++
			if m.Pass {
				passed++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(passed) / float64(total)
}

// Summary returns a human-readable one-line summary suitable for logs and CI output.
func (r RunResult) Summary() string {
	var totalMetrics, failedMetrics, failedCases int
	for _, c := range r.Cases {
		if c.RunError != nil {
			failedCases++
			continue
		}
		for _, m := range c.Metrics {
			totalMetrics++
			if !m.Pass {
				failedMetrics++
			}
		}
	}
	return fmt.Sprintf("cases=%d failed_cases=%d metrics=%d failed_metrics=%d pass_rate=%.1f%%",
		len(r.Cases), failedCases, totalMetrics, failedMetrics, r.PassRate()*100)
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
