package metric

import (
	"fmt"
	"strings"
)

// kindError tags a metric-related error so the root ClassifyError can map it
// back to a typed ErrorKind. The kind strings here MUST match the values of
// gaugo.ErrorKindMetric and gaugo.ErrorKindMetricParse — they are part of the
// public contract observable by users through gaugo.ClassifyError.
type kindError struct {
	kind  string
	msg   string
	cause error
}

const (
	kindMetric      = "metric"
	kindMetricParse = "metric_parse"
)

func (e kindError) Error() string {
	if e.cause == nil {
		return e.msg
	}
	return e.msg + ": " + e.cause.Error()
}

func (e kindError) Unwrap() error { return e.cause }

// GaugoErrorKind is matched structurally by gaugo.ClassifyError.
func (e kindError) GaugoErrorKind() string { return e.kind }

func newError(kind, msg string) error {
	return kindError{kind: kind, msg: msg}
}

func wrapError(kind, msg string, cause error) error {
	return kindError{kind: kind, msg: msg, cause: cause}
}

func metricErrorf(format string, args ...any) error {
	return newError(kindMetric, fmt.Sprintf(format, args...))
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
	return wrapError(kindMetricParse, fmt.Sprintf("%s parse failed", label), cause)
}

func marshalError(label string, cause error) error {
	return wrapError(kindMetric, fmt.Sprintf("%s marshal details failed", label), cause)
}

func newDetailsMarshalError(name string, cause error) error {
	return wrapError(kindMetric, fmt.Sprintf("%s marshal details failed", strings.ToLower(name)), cause)
}

func judgeEvaluationError(cause error) error {
	return fmt.Errorf("judge evaluation failed: %w", cause)
}
