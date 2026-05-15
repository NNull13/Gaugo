package gaugo

import (
	"encoding/json"

	"github.com/nnull13/gaugo/internal/failure"
)

// ErrorKind classifies operational failures separately from low quality scores.
type ErrorKind = failure.Kind

const (
	ErrorKindUnknown             = failure.KindUnknown
	ErrorKindConfig              = failure.KindConfig
	ErrorKindValidation          = failure.KindValidation
	ErrorKindContextCanceled     = failure.KindContextCanceled
	ErrorKindContextDeadline     = failure.KindContextDeadline
	ErrorKindPanic               = failure.KindPanic
	ErrorKindMetric              = failure.KindMetric
	ErrorKindMetricParse         = failure.KindMetricParse
	ErrorKindProviderRequest     = failure.KindProviderRequest
	ErrorKindProviderAuth        = failure.KindProviderAuth
	ErrorKindProviderRateLimit   = failure.KindProviderRateLimit
	ErrorKindProviderUnavailable = failure.KindProviderUnavailable
	ErrorKindProviderResponse    = failure.KindProviderResponse
	ErrorKindProviderRefusal     = failure.KindProviderRefusal
	ErrorKindProviderTruncated   = failure.KindProviderTruncated
)

// ErrorCode is a stable machine-readable reason for a Gaugo failure.
type ErrorCode = failure.Code

// Error is Gaugo's typed operational error. It keeps a human message,
// structured integration metadata, and an optional wrapped cause.
type Error = failure.Error

// ErrorInfo is a redacted, structured description of an operational failure.
type ErrorInfo = failure.Info

var (
	ErrConfig              = failure.ErrConfig
	ErrValidation          = failure.ErrValidation
	ErrMetric              = failure.ErrMetric
	ErrMetricParse         = failure.ErrMetricParse
	ErrProviderRequest     = failure.ErrProviderRequest
	ErrProviderAuth        = failure.ErrProviderAuth
	ErrProviderRateLimit   = failure.ErrProviderRateLimit
	ErrProviderUnavailable = failure.ErrProviderUnavailable
	ErrProviderResponse    = failure.ErrProviderResponse
	ErrProviderRefusal     = failure.ErrProviderRefusal
	ErrProviderTruncated   = failure.ErrProviderTruncated
	ErrPanic               = failure.ErrPanic
)

// ClassifyError returns redacted operational metadata for err.
func ClassifyError(err error) ErrorInfo {
	return failure.Classify(err)
}

// MetricErrorInfo extracts ErrorInfo stored in MetricResult details.
func MetricErrorInfo(m MetricResult) (ErrorInfo, bool) {
	if len(m.Details) == 0 {
		return ErrorInfo{}, false
	}
	var info ErrorInfo
	if err := json.Unmarshal(m.Details, &info); err != nil {
		return ErrorInfo{}, false
	}
	if info.Kind == "" {
		return ErrorInfo{}, false
	}
	return info, true
}

func errorInfoDetails(err error) []byte {
	info := ClassifyError(err)
	if info.Kind == "" {
		return nil
	}
	details, marshalErr := json.Marshal(info)
	if marshalErr != nil {
		return nil
	}
	return details
}
