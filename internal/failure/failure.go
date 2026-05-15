package failure

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Kind classifies operational failures separately from low quality scores.
type Kind string

const (
	KindUnknown             Kind = "unknown"
	KindConfig              Kind = "config"
	KindValidation          Kind = "validation"
	KindContextCanceled     Kind = "context_canceled"
	KindContextDeadline     Kind = "context_deadline"
	KindPanic               Kind = "panic"
	KindMetric              Kind = "metric"
	KindMetricParse         Kind = "metric_parse"
	KindProviderRequest     Kind = "provider_request"
	KindProviderAuth        Kind = "provider_auth"
	KindProviderRateLimit   Kind = "provider_rate_limit"
	KindProviderUnavailable Kind = "provider_unavailable"
	KindProviderResponse    Kind = "provider_response"
	KindProviderRefusal     Kind = "provider_refusal"
	KindProviderTruncated   Kind = "provider_truncated"
)

// Code is a stable machine-readable reason for a Gaugo failure.
type Code string

const (
	CodeUnknown                  Code = "unknown"
	CodeConfigInvalid            Code = "config_invalid"
	CodeRunnerConfigInvalid      Code = "runner_config_invalid"
	CodeValidationInvalid        Code = "validation_invalid"
	CodeContextCanceled          Code = "context_canceled"
	CodeContextDeadline          Code = "context_deadline"
	CodeCaseInvalid              Code = "case_invalid"
	CodeRetryInvalid             Code = "retry_invalid"
	CodeMetricInvalid            Code = "metric_invalid"
	CodeMetricParse              Code = "metric_parse"
	CodeProviderConfigInvalid    Code = "provider_config_invalid"
	CodeProviderAPIKeyRequired   Code = "provider_api_key_required"
	CodeProviderHTTPRequest      Code = "provider_http_request"
	CodeProviderHTTPStatus       Code = "provider_http_status"
	CodeProviderResponseDecode   Code = "provider_response_decode"
	CodeProviderResponseInvalid  Code = "provider_response_invalid"
	CodeProviderResponseTooLarge Code = "provider_response_too_large"
	CodeProviderRefusal          Code = "provider_refusal"
	CodeProviderTruncated        Code = "provider_truncated"
	CodePanic                    Code = "panic"
)

// Info is a redacted, structured description of an operational failure.
type Info struct {
	Kind       Kind   `json:"kind,omitempty"`
	Code       Code   `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
	Operation  string `json:"operation,omitempty"`
	Field      string `json:"field,omitempty"`
	Provider   string `json:"provider,omitempty"`
	Wire       string `json:"wire,omitempty"`
	Model      string `json:"model,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	BodyBytes  int    `json:"body_bytes,omitempty"`
	LimitBytes int64  `json:"limit_bytes,omitempty"`
	Retryable  bool   `json:"retryable,omitempty"`
}

// Error is Gaugo's typed operational error. It keeps a human message,
// structured integration metadata, and an optional wrapped cause.
type Error struct {
	Kind       Kind
	Code       Code
	Message    string
	Operation  string
	Field      string
	Provider   string
	Wire       string
	Model      string
	StatusCode int
	RequestID  string
	BodyBytes  int
	LimitBytes int64
	Retryable  bool
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		switch {
		case strings.TrimSpace(string(e.Code)) != "":
			msg = string(e.Code)
		case strings.TrimSpace(string(e.Kind)) != "":
			msg = string(e.Kind)
		default:
			msg = "gaugo error"
		}
	}
	if e.Err == nil {
		return msg
	}
	cause := e.Err.Error()
	if cause == "" || strings.Contains(msg, cause) {
		return msg
	}
	return msg + ": " + cause
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is lets callers use errors.Is(err, gaugo.ErrProviderRateLimit) style checks.
func (e *Error) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}
	info := e.Info()
	if targetInfo, ok := target.(*Error); ok {
		return matchInfo(info, targetInfo.Info())
	}
	var kindErr interface{ GaugoErrorKind() string }
	if ok := errors.As(target, &kindErr); ok {
		kind := Kind(strings.TrimSpace(kindErr.GaugoErrorKind()))
		if kind != "" && kind != KindUnknown && info.Kind == kind {
			return true
		}
	}
	var codeErr interface{ GaugoErrorCode() string }
	if ok := errors.As(target, &codeErr); ok {
		code := Code(strings.TrimSpace(codeErr.GaugoErrorCode()))
		return code != "" && code != CodeUnknown && info.Code == code
	}
	return false
}

func (e *Error) Info() Info {
	if e == nil {
		return Info{}
	}
	kind := normalizeKind(e.Kind)
	if (kind == "" || kind == KindUnknown) && e.StatusCode != 0 {
		kind = KindFromStatusCode(e.StatusCode)
	}
	return Info{
		Kind:       kind,
		Code:       normalizeCode(e.Code),
		Message:    strings.TrimSpace(e.Message),
		Operation:  strings.TrimSpace(e.Operation),
		Field:      strings.TrimSpace(e.Field),
		Provider:   strings.TrimSpace(e.Provider),
		Wire:       strings.TrimSpace(e.Wire),
		Model:      strings.TrimSpace(e.Model),
		StatusCode: e.StatusCode,
		RequestID:  strings.TrimSpace(e.RequestID),
		BodyBytes:  e.BodyBytes,
		LimitBytes: e.LimitBytes,
		Retryable:  e.Retryable,
	}
}

func (e *Error) GaugoErrorKind() string {
	if e == nil {
		return ""
	}
	return string(e.Info().Kind)
}

func (e *Error) GaugoErrorCode() string {
	if e == nil {
		return ""
	}
	return string(normalizeCode(e.Code))
}

func (e *Error) GaugoOperation() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.Operation)
}

func (e *Error) GaugoField() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.Field)
}

func (e *Error) GaugoProvider() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.Provider)
}

func (e *Error) GaugoWire() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.Wire)
}

func (e *Error) GaugoModel() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.Model)
}

func (e *Error) GaugoStatusCode() int {
	if e == nil {
		return 0
	}
	return e.StatusCode
}

func (e *Error) GaugoRequestID() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.RequestID)
}

func (e *Error) GaugoBodyBytes() int {
	if e == nil {
		return 0
	}
	return e.BodyBytes
}

func (e *Error) GaugoLimitBytes() int64 {
	if e == nil {
		return 0
	}
	return e.LimitBytes
}

func (e *Error) GaugoRetryable() bool {
	return e != nil && e.Retryable
}

type sentinel struct {
	kind Kind
	code Code
	msg  string
}

func (s sentinel) Error() string {
	if s.msg != "" {
		return s.msg
	}
	if s.code != "" {
		return string(s.code)
	}
	return string(s.kind)
}

func (s sentinel) GaugoErrorKind() string { return string(s.kind) }
func (s sentinel) GaugoErrorCode() string { return string(s.code) }

var (
	ErrConfig              error = sentinel{kind: KindConfig, msg: "gaugo config error"}
	ErrValidation          error = sentinel{kind: KindValidation, msg: "gaugo validation error"}
	ErrMetric              error = sentinel{kind: KindMetric, msg: "gaugo metric error"}
	ErrMetricParse         error = sentinel{kind: KindMetricParse, msg: "gaugo metric parse error"}
	ErrProviderRequest     error = sentinel{kind: KindProviderRequest, msg: "gaugo provider request error"}
	ErrProviderAuth        error = sentinel{kind: KindProviderAuth, msg: "gaugo provider auth error"}
	ErrProviderRateLimit   error = sentinel{kind: KindProviderRateLimit, msg: "gaugo provider rate limit error"}
	ErrProviderUnavailable error = sentinel{kind: KindProviderUnavailable, msg: "gaugo provider unavailable error"}
	ErrProviderResponse    error = sentinel{kind: KindProviderResponse, msg: "gaugo provider response error"}
	ErrProviderRefusal     error = sentinel{kind: KindProviderRefusal, msg: "gaugo provider refusal error"}
	ErrProviderTruncated   error = sentinel{kind: KindProviderTruncated, msg: "gaugo provider truncated error"}
	ErrPanic               error = sentinel{kind: KindPanic, msg: "gaugo panic error"}
)

func Config(code Code, operation, message string, cause error) error {
	return &Error{Kind: KindConfig, Code: code, Operation: operation, Message: message, Err: cause}
}

func Validation(code Code, operation, field, message string, cause error) error {
	return &Error{Kind: KindValidation, Code: code, Operation: operation, Field: field, Message: message, Err: cause}
}

func Metric(code Code, message string, cause error) error {
	return &Error{Kind: KindMetric, Code: code, Message: message, Err: cause}
}

func MetricParse(message string, cause error) error {
	return &Error{Kind: KindMetricParse, Code: CodeMetricParse, Message: message, Err: cause}
}

func KindFromStatusCode(code int) Kind {
	switch {
	case code == http.StatusTooManyRequests:
		return KindProviderRateLimit
	case code == http.StatusUnauthorized || code == http.StatusForbidden:
		return KindProviderAuth
	case code >= http.StatusBadRequest && code < http.StatusInternalServerError:
		return KindProviderRequest
	case code >= http.StatusInternalServerError:
		return KindProviderUnavailable
	default:
		return KindUnknown
	}
}

func RetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= http.StatusInternalServerError
}

func Classify(err error) Info {
	if err == nil {
		return Info{}
	}

	info := Info{
		Kind:    KindUnknown,
		Message: err.Error(),
	}

	if errors.Is(err, context.Canceled) {
		info.Kind = KindContextCanceled
		info.Code = CodeContextCanceled
		return info
	}
	if errors.Is(err, context.DeadlineExceeded) {
		info.Kind = KindContextDeadline
		info.Code = CodeContextDeadline
		return info
	}

	walk(err, func(e error) {
		mergeFromError(&info, e)
	})

	if info.Kind == "" {
		info.Kind = KindUnknown
	}
	if (info.Kind == KindUnknown || info.Kind == "") && info.StatusCode != 0 {
		info.Kind = KindFromStatusCode(info.StatusCode)
	}
	return info
}

func mergeFromError(info *Info, err error) {
	if err == nil {
		return
	}
	if typed, ok := err.(*Error); ok {
		mergeInfo(info, typed.Info())
	}
	if typed, ok := err.(interface{ Info() Info }); ok {
		mergeInfo(info, typed.Info())
	}
	if typed, ok := err.(interface{ GaugoErrorKind() string }); ok {
		mergeKind(info, Kind(strings.TrimSpace(typed.GaugoErrorKind())))
	}
	if typed, ok := err.(interface{ GaugoErrorCode() string }); ok {
		mergeCode(info, Code(strings.TrimSpace(typed.GaugoErrorCode())))
	}
	if typed, ok := err.(interface{ GaugoOperation() string }); ok && info.Operation == "" {
		info.Operation = strings.TrimSpace(typed.GaugoOperation())
	}
	if typed, ok := err.(interface{ GaugoField() string }); ok && info.Field == "" {
		info.Field = strings.TrimSpace(typed.GaugoField())
	}
	if typed, ok := err.(interface{ GaugoProvider() string }); ok && info.Provider == "" {
		info.Provider = strings.TrimSpace(typed.GaugoProvider())
	}
	if typed, ok := err.(interface{ GaugoWire() string }); ok && info.Wire == "" {
		info.Wire = strings.TrimSpace(typed.GaugoWire())
	}
	if typed, ok := err.(interface{ GaugoModel() string }); ok && info.Model == "" {
		info.Model = strings.TrimSpace(typed.GaugoModel())
	}
	if typed, ok := err.(interface{ GaugoStatusCode() int }); ok && info.StatusCode == 0 {
		info.StatusCode = typed.GaugoStatusCode()
	}
	if typed, ok := err.(interface{ GaugoRequestID() string }); ok && info.RequestID == "" {
		info.RequestID = strings.TrimSpace(typed.GaugoRequestID())
	}
	if typed, ok := err.(interface{ GaugoBodyBytes() int }); ok && info.BodyBytes == 0 {
		info.BodyBytes = typed.GaugoBodyBytes()
	}
	if typed, ok := err.(interface{ GaugoLimitBytes() int64 }); ok && info.LimitBytes == 0 {
		info.LimitBytes = typed.GaugoLimitBytes()
	}
	if typed, ok := err.(interface{ GaugoRetryable() bool }); ok && typed.GaugoRetryable() {
		info.Retryable = true
	}
}

func mergeInfo(dst *Info, src Info) {
	mergeKind(dst, src.Kind)
	mergeCode(dst, src.Code)
	if dst.Operation == "" {
		dst.Operation = strings.TrimSpace(src.Operation)
	}
	if dst.Field == "" {
		dst.Field = strings.TrimSpace(src.Field)
	}
	if dst.Provider == "" {
		dst.Provider = strings.TrimSpace(src.Provider)
	}
	if dst.Wire == "" {
		dst.Wire = strings.TrimSpace(src.Wire)
	}
	if dst.Model == "" {
		dst.Model = strings.TrimSpace(src.Model)
	}
	if dst.StatusCode == 0 {
		dst.StatusCode = src.StatusCode
	}
	if dst.RequestID == "" {
		dst.RequestID = strings.TrimSpace(src.RequestID)
	}
	if dst.BodyBytes == 0 {
		dst.BodyBytes = src.BodyBytes
	}
	if dst.LimitBytes == 0 {
		dst.LimitBytes = src.LimitBytes
	}
	if src.Retryable {
		dst.Retryable = true
	}
}

func mergeKind(dst *Info, kind Kind) {
	kind = normalizeKind(kind)
	if kind == "" || kind == KindUnknown {
		return
	}
	if dst.Kind == "" || dst.Kind == KindUnknown {
		dst.Kind = kind
	}
}

func mergeCode(dst *Info, code Code) {
	code = normalizeCode(code)
	if code == "" || code == CodeUnknown {
		return
	}
	if dst.Code == "" || dst.Code == CodeUnknown {
		dst.Code = code
	}
}

func matchInfo(got, target Info) bool {
	if target.Kind != "" && target.Kind != KindUnknown && got.Kind != target.Kind {
		return false
	}
	if target.Code != "" && target.Code != CodeUnknown && got.Code != target.Code {
		return false
	}
	return (target.Kind != "" && target.Kind != KindUnknown) || (target.Code != "" && target.Code != CodeUnknown)
}

func walk(err error, visit func(error)) {
	if err == nil {
		return
	}
	visit(err)
	switch unwrapped := err.(type) {
	case interface{ Unwrap() []error }:
		for _, child := range unwrapped.Unwrap() {
			walk(child, visit)
		}
	case interface{ Unwrap() error }:
		walk(unwrapped.Unwrap(), visit)
	}
}

func normalizeKind(kind Kind) Kind {
	kind = Kind(strings.TrimSpace(string(kind)))
	if kind == "" {
		return ""
	}
	return kind
}

func normalizeCode(code Code) Code {
	code = Code(strings.TrimSpace(string(code)))
	if code == "" {
		return ""
	}
	return code
}

func RedactedURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err == nil && u != nil && u.User != nil {
		u.User = url.User("redacted")
		return u.String()
	}
	return raw
}

func FormatInvalidURL(raw, detail string) string {
	raw = RedactedURL(raw)
	if raw == "" {
		return strings.TrimSpace(detail)
	}
	return fmt.Sprintf("invalid base URL %q: %s", raw, detail)
}
