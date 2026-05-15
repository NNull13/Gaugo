package metric

import (
	"fmt"
	"strings"

	"github.com/nnull13/gaugo/internal/failure"
)

func metricErrorf(format string, args ...any) error {
	return failure.Metric(failure.CodeMetricInvalid, fmt.Sprintf(format, args...), nil)
}

// optionErrorf builds an option-validation error.
func optionErrorf(format string, args ...any) error {
	return metricErrorf(format, args...)
}

func requiresJudgeError(label string) error {
	return metricErrorf("%s metric requires a configured judge", label)
}

func requiresExpectedAnswerError(label string) error {
	return metricErrorf("%s metric requires Expected.Answer", label)
}

func requiresExpectedInstructionsError(label string) error {
	return metricErrorf("%s metric requires Expected.Instructions", label)
}

func parseError(label string, cause error) error {
	return failure.MetricParse(fmt.Sprintf("%s parse failed", label), cause)
}

func marshalError(label string, cause error) error {
	return failure.Metric(failure.CodeMetricInvalid, fmt.Sprintf("%s marshal details failed", label), cause)
}

func newDetailsMarshalError(name string, cause error) error {
	return failure.Metric(failure.CodeMetricInvalid, fmt.Sprintf("%s marshal details failed", strings.ToLower(name)), cause)
}

func judgeEvaluationError(cause error) error {
	return fmt.Errorf("judge evaluation failed: %w", cause)
}
