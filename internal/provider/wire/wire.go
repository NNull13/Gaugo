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
)

const DefaultMaxResponseBody int64 = 1 << 20

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
	errorKindUnknown             = "unknown"
	errorKindProviderRequest     = "provider_request"
	errorKindProviderAuth        = "provider_auth"
	errorKindProviderRateLimit   = "provider_rate_limit"
	errorKindProviderUnavailable = "provider_unavailable"
)

// Common error detail strings used by wire sub-packages.
const (
	ErrInvalidJSONPayload = "invalid json payload"
	ErrRefusal            = "model refusal"
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
		return nil, fmt.Errorf("judge request schema is required")
	}
	var decoded any
	err := json.Unmarshal(schema, &decoded)
	if err != nil {
		return nil, fmt.Errorf("decode json schema: %w", err)
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
type HTTPStatusError struct {
	Provider   string
	Wire       string
	StatusCode int
	RequestID  string
	BodyLen    int
}

func (e *HTTPStatusError) Error() string {
	label := strings.TrimSpace(e.Provider)
	if wire := strings.TrimSpace(e.Wire); wire != "" {
		if label == "" {
			label = wire
		} else {
			label += "/" + wire
		}
	}
	if label == "" {
		label = "provider"
	}
	return fmt.Sprintf(
		"%s request failed status=%d request_id=%q body=redacted(len=%d)",
		label,
		e.StatusCode,
		e.RequestID,
		e.BodyLen,
	)
}

func (e *HTTPStatusError) GaugoErrorKind() string {
	switch {
	case e.StatusCode == http.StatusTooManyRequests:
		return errorKindProviderRateLimit
	case e.StatusCode == http.StatusUnauthorized || e.StatusCode == http.StatusForbidden:
		return errorKindProviderAuth
	case e.StatusCode >= http.StatusBadRequest && e.StatusCode < http.StatusInternalServerError:
		return errorKindProviderRequest
	case e.StatusCode >= http.StatusInternalServerError:
		return errorKindProviderUnavailable
	default:
		return errorKindUnknown
	}
}

func (e *HTTPStatusError) GaugoProvider() string {
	return strings.TrimSpace(e.Provider)
}

func (e *HTTPStatusError) GaugoWire() string {
	return strings.TrimSpace(e.Wire)
}

func (e *HTTPStatusError) GaugoStatusCode() int {
	return e.StatusCode
}

func (e *HTTPStatusError) GaugoRequestID() string {
	return strings.TrimSpace(e.RequestID)
}

func (e *HTTPStatusError) GaugoBodyBytes() int {
	return e.BodyLen
}

func PostJSON(ctx context.Context, client *http.Client, url string, headers map[string]string, body []byte) (HTTPResponse, error) {
	return PostJSONWithOptions(ctx, client, url, headers, body, HTTPOptions{})
}

func PostJSONWithOptions(ctx context.Context, client *http.Client, url string, headers map[string]string, body []byte, opts HTTPOptions) (HTTPResponse, error) {
	opts = NormalizeHTTPOptions(opts)
	var last HTTPResponse

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
		return HTTPResponse{}, fmt.Errorf("build request: %w", err)
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
		return HTTPResponse{}, fmt.Errorf("execute request: %w", err)
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

// MarshalRequestError returns a formatted marshal error for a wire request.
func MarshalRequestError(label string, err error) error {
	return fmt.Errorf("marshal %s request: %w", label, err)
}

// DecodeResponseError returns a formatted decode error with a detail message.
func DecodeResponseError(label, detail string) error {
	return fmt.Errorf("decode %s response: %s", label, detail)
}

// DecodeResponseWrapError returns a formatted decode error wrapping an underlying error.
func DecodeResponseWrapError(label string, err error) error {
	return fmt.Errorf("decode %s response: %w", label, err)
}

// JudgeRequestError returns a formatted judge request failure error.
func JudgeRequestError(label string, err error) error {
	return fmt.Errorf("%s judge request failed: %w", label, err)
}

func StatusErrorForWire(provider, wire string, resp HTTPResponse) error {
	return &HTTPStatusError{
		Provider:   provider,
		Wire:       wire,
		StatusCode: resp.StatusCode,
		RequestID:  RequestID(resp.Header),
		BodyLen:    len(resp.Body),
	}
}

func StatusError(provider string, resp HTTPResponse) error {
	return StatusErrorForWire(provider, "", resp)
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
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "```")
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
		return "gaugo_metric"
	}
	return name
}

func readLimited(r io.Reader, maxBodyBytes int64) ([]byte, error) {
	if maxBodyBytes <= 0 {
		maxBodyBytes = DefaultMaxResponseBody
	}
	body, err := io.ReadAll(io.LimitReader(r, maxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if int64(len(body)) > maxBodyBytes {
		return nil, fmt.Errorf("%w: limit=%d", ErrResponseBodyTooLarge, maxBodyBytes)
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
	if seconds, err := strconv.Atoi(raw); err == nil {
		if seconds <= 0 {
			return 0, true
		}
		return time.Duration(seconds) * time.Second, true
	}
	when, err := http.ParseTime(raw)
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
