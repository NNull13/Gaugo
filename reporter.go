package gaugo

import (
	"context"
	"fmt"
	"sort"
	"testing"
)

// Reporter receives completed suite results without depending on testing.T.
type Reporter interface {
	Report(ctx context.Context, result RunResult)
}

// Assert reports result failures through the Go testing package.
func Assert(t testing.TB, result RunResult) {
	t.Helper()

	const maxFailureLogs = 100

	failed := false
	loggedFailures := 0
	suppressedFailures := 0
	totalMetrics := 0
	failedCases := 0
	failedMetrics := 0
	metricTotals := map[string]int{}
	metricFailed := map[string]int{}

	for _, c := range result.Cases {
		if c.RunError != nil {
			failed = true
			failedCases++
			if loggedFailures < maxFailureLogs {
				t.Errorf("gaugo case %q run failed: %v", c.Name, c.RunError)
				loggedFailures++
			} else {
				suppressedFailures++
			}
			continue
		}

		for _, m := range c.Metrics {
			totalMetrics++
			metricTotals[m.Name]++

			if m.Pass {
				continue
			}
			failed = true
			failedMetrics++
			metricFailed[m.Name]++

			if loggedFailures < maxFailureLogs {
				t.Errorf("gaugo case %q metric %q failed (score=%.3f): %s (%s)",
					c.Name, m.Name, m.Score, m.Reason, metricFailureMetadata(m))
				loggedFailures++
			} else {
				suppressedFailures++
			}
		}
	}

	if !failed {
		t.Logf("gaugo summary: cases=%d failed_cases=0 metrics=%d failed_metrics=0", len(result.Cases), totalMetrics)
		return
	}

	metricNames := make([]string, 0, len(metricTotals))
	for name := range metricTotals {
		metricNames = append(metricNames, name)
	}
	sort.Strings(metricNames)
	for _, name := range metricNames {
		t.Logf("gaugo metric summary: name=%q failed=%d total=%d", name, metricFailed[name], metricTotals[name])
	}

	if suppressedFailures > 0 {
		t.Errorf("gaugo: suppressed %d additional failure log(s) after first %d", suppressedFailures, maxFailureLogs)
	}

	t.Logf("gaugo summary: cases=%d failed_cases=%d metrics=%d failed_metrics=%d logged_failures=%d",
		len(result.Cases), failedCases, totalMetrics, failedMetrics, loggedFailures)
}

func metricFailureMetadata(m MetricResult) string {
	metadata := fmt.Sprintf("details_bytes=%d", len(m.Details))
	if info, ok := MetricErrorInfo(m); ok {
		metadata += fmt.Sprintf(" error_kind=%q", info.Kind)
		if info.Provider != "" {
			metadata += fmt.Sprintf(" provider=%q", info.Provider)
		}
		if info.StatusCode != 0 {
			metadata += fmt.Sprintf(" status_code=%d", info.StatusCode)
		}
		if info.RequestID != "" {
			metadata += fmt.Sprintf(" request_id=%q", info.RequestID)
		}
	}
	return metadata
}
