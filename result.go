package gaugo

import "time"

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
