package wire

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nnull13/gaugo/internal/failure"
)

const DefaultMaxResponseBody int64 = 1 << 20

const (
	defaultProviderLabel = "provider"
	defaultSchemaName    = "gaugo_metric"
	codeFenceJSON        = "```json"
	codeFence            = "```"
)

const (
	HeaderAuthorization    = "Authorization"
	HeaderContentType      = "Content-Type"
	HeaderRetryAfter       = "Retry-After"
	HeaderRequestID        = "request-id"
	HeaderXRequestID       = "x-request-id"
	HeaderAmazonRequestID  = "x-amzn-requestid"
	HeaderAnthropicAPIKey  = "x-api-key"
	HeaderAnthropicVersion = "anthropic-version"
	HeaderGeminiAPIKey     = "x-goog-api-key"
)

const (
	AuthBearerPrefix     = "Bearer "
	ContentTypeJSON      = "application/json"
	SchemaTypeJSONSchema = "json_schema"

	RoleSystem = "system"
	RoleUser   = "user"
)

const (
	ErrorKindProviderResponse  = failure.KindProviderResponse
	ErrorKindProviderRefusal   = failure.KindProviderRefusal
	ErrorKindProviderTruncated = failure.KindProviderTruncated
)

// Common error detail strings used by wire sub-packages.
const (
	ErrJudgeRequestSchemaRequired = "judge request schema is required"
	ErrInvalidJSONPayload         = "invalid json payload"
	ErrRefusal                    = "model refusal"
	ErrNoChoicesReturned          = "no choices returned"
	ErrOutputTruncatedTokenLimit  = "output truncated by token limit"
	ErrOutputTruncatedMaxTokens   = "output truncated by max_tokens"
	ErrEmptyMessageContent        = "empty message content"
	ErrEmptyTextContent           = "empty text content"
)

const (
	errDecodeJSONSchema = "decode json schema"
	errBuildRequest     = "build request"
	errExecuteRequest   = "execute request"
	errReadResponseBody = "read response body"
)

const (
	httpStatusErrorFormat     = "%s request failed status=%d request_id=%q body=redacted(len=%d)"
	decodeResponseErrorFormat = "decode %s response: %s"
)

var (
	defaultHTTPClient = &http.Client{
		Timeout: 60 * time.Second,
	}

	// ErrResponseBodyTooLarge is returned when a provider response exceeds the configured limit.
	ErrResponseBodyTooLarge = errors.New("provider response body too large")
)

// RetryConfig controls retries for transient provider HTTP failures.
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// HTTPOptions configures one provider HTTP request.
type HTTPOptions struct {
	MaxBodyBytes int64
	Retry        RetryConfig
}

// EvalRequest is the internal provider-agnostic request model for wire packages.
type EvalRequest struct {
	Metric       string
	Instructions string
	UserPrompt   string
	Schema       json.RawMessage
}

// EvalResult is the normalized output from a wire protocol call.
type EvalResult struct {
	RawJSON   []byte
	Model     string
	Latency   time.Duration
	RequestID string
}

// NewHTTPClient returns cfg or a deterministic default client.
func NewHTTPClient(cfg *http.Client) *http.Client {
	if cfg != nil {
		return cfg
	}
	return defaultHTTPClient
}

// DefaultRetryConfig returns production-oriented retry defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    2 * time.Second,
	}
}

// NormalizeHTTPOptions fills default HTTP options.
func NormalizeHTTPOptions(opts HTTPOptions) HTTPOptions {
	if opts.MaxBodyBytes <= 0 {
		opts.MaxBodyBytes = DefaultMaxResponseBody
	}
	opts.Retry = NormalizeRetryConfig(opts.Retry)
	return opts
}

// NormalizeRetryConfig fills default retry options.
func NormalizeRetryConfig(cfg RetryConfig) RetryConfig {
	def := DefaultRetryConfig()
	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = def.MaxAttempts
	}
	if cfg.BaseDelay == 0 {
		cfg.BaseDelay = def.BaseDelay
	}
	if cfg.MaxDelay == 0 {
		cfg.MaxDelay = def.MaxDelay
	}
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}
	return cfg
}

// DecodeSchema ensures request schema is valid JSON and decodes to a generic object.
func DecodeSchema(schema json.RawMessage) (any, error) {
	if len(schema) == 0 {
		return nil, &failure.Error{
			Kind:    failure.KindProviderRequest,
			Code:    failure.CodeProviderHTTPRequest,
			Message: ErrJudgeRequestSchemaRequired,
		}
	}
	var decoded any
	err := json.Unmarshal(schema, &decoded)
	if err != nil {
		return nil, &failure.Error{
			Kind:    failure.KindProviderRequest,
			Code:    failure.CodeProviderHTTPRequest,
			Message: errDecodeJSONSchema,
			Err:     err,
		}
	}
	return decoded, nil
}

type HTTPResponse struct {
	Body       []byte
	Header     http.Header
	StatusCode int
	Latency    time.Duration
}

// HTTPStatusError reports a non-success provider HTTP response without exposing the body.
type HTTPStatusError = failure.Error

// OperationalError reports non-HTTP provider wire failures with structured metadata.
type OperationalError = failure.Error

func PostJSON(ctx context.Context, client *http.Client, url string, headers map[string]string, body []byte) (HTTPResponse, error) {
	return PostJSONWithOptions(ctx, client, url, headers, body, HTTPOptions{})
}

func PostJSONWithOptions(ctx context.Context, client *http.Client, url string, headers map[string]string, body []byte, opts HTTPOptions) (HTTPResponse, error) {
	opts = NormalizeHTTPOptions(opts)
	var last HTTPResponse
	start := time.Now()

	for attempt := 1; attempt <= opts.Retry.MaxAttempts; attempt++ {
		resp, err := postJSONOnce(ctx, client, url, headers, body, opts.MaxBodyBytes)
		if err != nil {
			if retryableTransportError(err) && attempt < opts.Retry.MaxAttempts {
				err = sleepRetry(ctx, retryDelay(nil, opts.Retry, attempt))
				if err != nil {
					return last, err
				}
				continue
			}
			return HTTPResponse{}, err
		}
		resp.Latency = time.Since(start)
		last = resp
		if !retryableStatus(resp.StatusCode) || attempt == opts.Retry.MaxAttempts {
			return resp, nil
		}
		err = sleepRetry(ctx, retryDelay(resp.Header, opts.Retry, attempt))
		if err != nil {
			return last, err
		}
	}

	return last, nil
}

func postJSONOnce(ctx context.Context, client *http.Client, url string, headers map[string]string, body []byte, maxBodyBytes int64) (HTTPResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return HTTPResponse{}, &failure.Error{
			Kind:    failure.KindProviderRequest,
			Code:    failure.CodeProviderHTTPRequest,
			Message: errBuildRequest,
			Err:     err,
		}
	}
	for k, v := range headers {
		if strings.TrimSpace(v) == "" {
			continue
		}
		req.Header.Set(k, v)
	}
	if req.Header.Get(HeaderContentType) == "" {
		req.Header.Set(HeaderContentType, ContentTypeJSON)
	}

	start := time.Now()

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return HTTPResponse{}, &failure.Error{
			Kind:      failure.KindProviderRequest,
			Code:      failure.CodeProviderHTTPRequest,
			Message:   errExecuteRequest,
			Retryable: retryableTransportError(err),
			Err:       err,
		}
	}
	defer resp.Body.Close()

	var respBody []byte
	respBody, err = readLimited(resp.Body, maxBodyBytes)
	if err != nil {
		return HTTPResponse{}, err
	}

	return HTTPResponse{
		Body:       respBody,
		Header:     resp.Header.Clone(),
		StatusCode: resp.StatusCode,
		Latency:    time.Since(start),
	}, nil
}

// EndpointURL resolves the final URL from explicit endpoint, base URL, default base, and path.
func EndpointURL(endpointURL, baseURL, defaultBaseURL, path string) string {
	if endpoint := strings.TrimSpace(endpointURL); endpoint != "" {
		return endpoint
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = defaultBaseURL
	}
	return base + path
}

// ProviderLabel returns configured if non-empty, otherwise fallback.
func ProviderLabel(configured, fallback string) string {
	configured = strings.TrimSpace(configured)
	if configured != "" {
		return configured
	}
	return fallback
}

func MarshalRequestErrorForWire(provider, wire string, err error) error {
	label := providerWireLabel(provider, wire)
	return &OperationalError{
		Kind:     failure.KindProviderRequest,
		Code:     failure.CodeProviderHTTPRequest,
		Provider: provider,
		Wire:     wire,
		Message:  fmt.Sprintf("marshal %s request", label),
		Err:      err,
	}
}

func DecodeResponseErrorForWire(kind failure.Kind, provider, wireName, detail string, requestID ...string) error {
	if kind == "" || kind == failure.KindUnknown {
		kind = failure.KindProviderResponse
	}
	label := providerWireLabel(provider, wireName)
	return &OperationalError{
		Kind:      kind,
		Code:      decodeErrorCode(kind),
		Provider:  provider,
		Wire:      wireName,
		RequestID: optionalRequestID(requestID),
		Message:   fmt.Sprintf(decodeResponseErrorFormat, label, detail),
	}
}

func DecodeResponseWrapErrorForWire(provider, wireName string, err error, requestID ...string) error {
	label := providerWireLabel(provider, wireName)
	return &OperationalError{
		Kind:      failure.KindProviderResponse,
		Code:      failure.CodeProviderResponseDecode,
		Provider:  provider,
		Wire:      wireName,
		RequestID: optionalRequestID(requestID),
		Message:   fmt.Sprintf("decode %s response", label),
		Err:       err,
	}
}

func JudgeRequestErrorForWire(provider, wireName string, err error) error {
	label := providerWireLabel(provider, wireName)
	info := failure.Classify(err)
	kind := failure.KindProviderRequest
	code := failure.CodeProviderHTTPRequest
	if info.Kind != "" && info.Kind != failure.KindUnknown {
		kind = info.Kind
	}
	if info.Code != "" && info.Code != failure.CodeUnknown {
		code = info.Code
	}
	return &OperationalError{
		Kind:     kind,
		Code:     code,
		Provider: provider,
		Wire:     wireName,
		Message:  fmt.Sprintf("%s judge request failed", label),
		Err:      err,
	}
}

func StatusErrorForWire(provider, wire string, resp HTTPResponse) error {
	requestID := RequestID(resp.Header)
	bodyBytes := len(resp.Body)
	statusKind := failure.KindFromStatusCode(resp.StatusCode)
	label := providerWireLabel(provider, wire)
	return &HTTPStatusError{
		Kind:       statusKind,
		Code:       failure.CodeProviderHTTPStatus,
		Provider:   provider,
		Wire:       wire,
		StatusCode: resp.StatusCode,
		RequestID:  requestID,
		BodyBytes:  bodyBytes,
		Retryable:  failure.RetryableStatus(resp.StatusCode),
		Message:    fmt.Sprintf(httpStatusErrorFormat, label, resp.StatusCode, requestID, bodyBytes),
	}
}

func StatusError(provider string, resp HTTPResponse) error {
	return StatusErrorForWire(provider, "", resp)
}

func providerWireLabel(provider, wireName string) string {
	label := strings.TrimSpace(provider)
	wireName = strings.TrimSpace(wireName)
	if wireName != "" {
		if label == "" {
			label = wireName
		} else {
			label += "/" + wireName
		}
	}
	if label == "" {
		return defaultProviderLabel
	}
	return label
}

func decodeErrorCode(kind failure.Kind) failure.Code {
	switch kind {
	case failure.KindProviderRefusal:
		return failure.CodeProviderRefusal
	case failure.KindProviderTruncated:
		return failure.CodeProviderTruncated
	default:
		return failure.CodeProviderResponseInvalid
	}
}

func optionalRequestID(values []string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func RequestID(h http.Header) string {
	for _, key := range []string{HeaderXRequestID, HeaderRequestID, HeaderAmazonRequestID} {
		v := strings.TrimSpace(h.Get(key))
		if v != "" {
			return v
		}
	}
	return ""
}

func StripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, codeFence) {
		return s
	}
	s = strings.TrimPrefix(s, codeFenceJSON)
	s = strings.TrimPrefix(s, codeFence)
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, codeFence)
	return strings.TrimSpace(s)
}

func NormalizeSchemaName(metric string) string {
	const maxSchemaNameLen = 64

	var b strings.Builder
	lastSeparator := false
	for _, r := range strings.ToLower(strings.TrimSpace(metric)) {
		if b.Len() >= maxSchemaNameLen {
			break
		}

		switch {
		case r >= 'a' && r <= 'z':
			b.WriteByte(byte(r))
			lastSeparator = false
		case r >= '0' && r <= '9':
			b.WriteByte(byte(r))
			lastSeparator = false
		default:
			if b.Len() == 0 || lastSeparator || b.Len() == maxSchemaNameLen-1 {
				continue
			}
			b.WriteByte('_')
			lastSeparator = true
		}
	}

	name := strings.Trim(b.String(), "_-")
	if name == "" {
		return defaultSchemaName
	}
	return name
}

func readLimited(r io.Reader, maxBodyBytes int64) ([]byte, error) {
	if maxBodyBytes <= 0 {
		maxBodyBytes = DefaultMaxResponseBody
	}
	body, err := io.ReadAll(io.LimitReader(r, maxBodyBytes+1))
	if err != nil {
		return nil, &failure.Error{
			Kind:    failure.KindProviderResponse,
			Code:    failure.CodeProviderResponseDecode,
			Message: errReadResponseBody,
			Err:     err,
		}
	}
	if int64(len(body)) > maxBodyBytes {
		return nil, &failure.Error{
			Kind:       failure.KindProviderResponse,
			Code:       failure.CodeProviderResponseTooLarge,
			Message:    fmt.Sprintf("%s: limit=%d", ErrResponseBodyTooLarge, maxBodyBytes),
			LimitBytes: maxBodyBytes,
			Err:        ErrResponseBodyTooLarge,
		}
	}
	return body, nil
}

func retryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

func retryableTransportError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	if errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
		type temporary interface {
			Temporary() bool
		}
		if tempErr, ok := any(netErr).(temporary); ok && tempErr.Temporary() {
			return true
		}
	}
	return false
}

func retryDelay(h http.Header, cfg RetryConfig, attempt int) time.Duration {
	if d, ok := retryAfter(h.Get(HeaderRetryAfter)); ok {
		return capDuration(d, cfg.MaxDelay)
	}

	exp := math.Pow(2, float64(attempt-1))
	delay := time.Duration(float64(cfg.BaseDelay) * exp)
	return capDuration(delay, cfg.MaxDelay)
}

func retryAfter(raw string) (time.Duration, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	seconds, err := strconv.Atoi(raw)
	if err == nil {
		if seconds <= 0 {
			return 0, true
		}
		return time.Duration(seconds) * time.Second, true
	}
	var when time.Time
	when, err = http.ParseTime(raw)
	if err != nil {
		return 0, false
	}
	d := time.Until(when)
	if d < 0 {
		return 0, true
	}
	return d, true
}

func capDuration(d, max time.Duration) time.Duration {
	if max > 0 && d > max {
		return max
	}
	return d
}

func sleepRetry(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
