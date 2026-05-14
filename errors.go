package gaugo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// ErrorKind classifies operational failures separately from low quality scores.
type ErrorKind string

const (
	ErrorKindUnknown             ErrorKind = "unknown"
	ErrorKindContextCanceled     ErrorKind = "context_canceled"
	ErrorKindContextDeadline     ErrorKind = "context_deadline"
	ErrorKindPanic               ErrorKind = "panic"
	ErrorKindMetric              ErrorKind = "metric"
	ErrorKindMetricParse         ErrorKind = "metric_parse"
	ErrorKindProviderRequest     ErrorKind = "provider_request"
	ErrorKindProviderAuth        ErrorKind = "provider_auth"
	ErrorKindProviderRateLimit   ErrorKind = "provider_rate_limit"
	ErrorKindProviderUnavailable ErrorKind = "provider_unavailable"
	ErrorKindProviderResponse    ErrorKind = "provider_response"
	ErrorKindProviderRefusal     ErrorKind = "provider_refusal"
	ErrorKindProviderTruncated   ErrorKind = "provider_truncated"
)

// ErrorInfo is a redacted, structured description of an operational failure.
type ErrorInfo struct {
	Kind       ErrorKind `json:"kind,omitempty"`
	Message    string    `json:"message,omitempty"`
	Provider   string    `json:"provider,omitempty"`
	Wire       string    `json:"wire,omitempty"`
	Model      string    `json:"model,omitempty"`
	StatusCode int       `json:"status_code,omitempty"`
	RequestID  string    `json:"request_id,omitempty"`
	BodyBytes  int       `json:"body_bytes,omitempty"`
}

// The following interfaces are matched structurally by ClassifyError: any
// error in the chain that implements one or more of them contributes its
// metadata to the returned ErrorInfo. Provider and metric packages implement
// these on their own error types — no shared base struct is required.

type gaugoKindError interface{ GaugoErrorKind() string }
type gaugoProviderError interface{ GaugoProvider() string }
type gaugoWireError interface{ GaugoWire() string }
type gaugoModelError interface{ GaugoModel() string }
type gaugoStatusCodeError interface{ GaugoStatusCode() int }
type gaugoRequestIDError interface{ GaugoRequestID() string }
type gaugoBodyBytesError interface{ GaugoBodyBytes() int }

// ClassifyError returns redacted operational metadata for err.
func ClassifyError(err error) ErrorInfo {
	if err == nil {
		return ErrorInfo{}
	}

	info := ErrorInfo{
		Kind:    ErrorKindUnknown,
		Message: err.Error(),
	}

	if errors.Is(err, context.Canceled) {
		info.Kind = ErrorKindContextCanceled
		return info
	}
	if errors.Is(err, context.DeadlineExceeded) {
		info.Kind = ErrorKindContextDeadline
		return info
	}

	var kindErr gaugoKindError
	if errors.As(err, &kindErr) {
		info.Kind = ErrorKind(strings.TrimSpace(kindErr.GaugoErrorKind()))
		if info.Kind == "" {
			info.Kind = ErrorKindUnknown
		}
	} else {
		// Prefer a structured status code from the error chain over string matching.
		var statusErr gaugoStatusCodeError
		if errors.As(err, &statusErr) {
			info.Kind = kindFromStatusCode(statusErr.GaugoStatusCode())
		} else {
			info.Kind = inferErrorKind(err)
		}
	}

	var providerErr gaugoProviderError
	if errors.As(err, &providerErr) {
		info.Provider = strings.TrimSpace(providerErr.GaugoProvider())
	}
	var wireErr gaugoWireError
	if errors.As(err, &wireErr) {
		info.Wire = strings.TrimSpace(wireErr.GaugoWire())
	}
	var modelErr gaugoModelError
	if errors.As(err, &modelErr) {
		info.Model = strings.TrimSpace(modelErr.GaugoModel())
	}
	var statusErr gaugoStatusCodeError
	if errors.As(err, &statusErr) {
		info.StatusCode = statusErr.GaugoStatusCode()
	}
	var requestErr gaugoRequestIDError
	if errors.As(err, &requestErr) {
		info.RequestID = strings.TrimSpace(requestErr.GaugoRequestID())
	}
	var bodyErr gaugoBodyBytesError
	if errors.As(err, &bodyErr) {
		info.BodyBytes = bodyErr.GaugoBodyBytes()
	}

	return info
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

// kindFromStatusCode classifies an error by HTTP status code.
func kindFromStatusCode(code int) ErrorKind {
	switch {
	case code == 429:
		return ErrorKindProviderRateLimit
	case code == 401 || code == 403:
		return ErrorKindProviderAuth
	case code >= 400 && code < 500:
		return ErrorKindProviderRequest
	case code >= 500:
		return ErrorKindProviderUnavailable
	default:
		return ErrorKindUnknown
	}
}

// inferErrorKind is a best-effort fallback for errors that do not implement
// gaugoKindError or gaugoStatusCodeError. It matches against known error
// message substrings produced by Gaugo's own packages. This is inherently
// fragile — prefer implementing GaugoErrorKind() on error types when possible.
func inferErrorKind(err error) ErrorKind {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, " parse failed"):
		return ErrorKindMetricParse
	case strings.Contains(msg, "provider response body too large"):
		return ErrorKindProviderResponse
	case strings.Contains(msg, "refusal"):
		return ErrorKindProviderRefusal
	case strings.Contains(msg, "truncated"):
		return ErrorKindProviderTruncated
	case strings.Contains(msg, "requires a configured judge"), strings.Contains(msg, "threshold must be"):
		return ErrorKindMetric
	default:
		return ErrorKindUnknown
	}
}
