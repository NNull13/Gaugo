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
	StatusCode int
	RequestID  string
	BodyLen    int
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf(
		"%s request failed status=%d request_id=%q body=redacted(len=%d)",
		e.Provider,
		e.StatusCode,
		e.RequestID,
		e.BodyLen,
	)
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
				if err := sleepRetry(ctx, retryDelay(nil, opts.Retry, attempt)); err != nil {
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
		if err := sleepRetry(ctx, retryDelay(resp.Header, opts.Retry, attempt)); err != nil {
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
	resp, err := client.Do(req)
	if err != nil {
		return HTTPResponse{}, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := readLimited(resp.Body, maxBodyBytes)
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

func StatusErrorFor(provider string, resp HTTPResponse) error {
	return &HTTPStatusError{
		Provider:   provider,
		StatusCode: resp.StatusCode,
		RequestID:  RequestID(resp.Header),
		BodyLen:    len(resp.Body),
	}
}

func StatusError(provider string, resp HTTPResponse) error {
	return StatusErrorFor(provider, resp)
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
	name := strings.ToLower(strings.TrimSpace(metric))
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
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
