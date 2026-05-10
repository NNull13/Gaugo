package wire

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestNewHTTPClientSingletonDefault(t *testing.T) {
	t.Parallel()

	c1 := NewHTTPClient(nil)
	c2 := NewHTTPClient(nil)
	if c1 != c2 {
		t.Fatalf("expected singleton default client pointer")
	}
}

func TestNewHTTPClientOverride(t *testing.T) {
	t.Parallel()

	custom := &http.Client{}
	if got := NewHTTPClient(custom); got != custom {
		t.Fatalf("expected custom client to be returned")
	}
}

func TestStatusErrorRedacted(t *testing.T) {
	t.Parallel()

	h := http.Header{}
	h.Set("request-id", "req_123")

	err := StatusError("openai", HTTPResponse{
		StatusCode: 400,
		Header:     h,
		Body:       []byte(`{"secret":"value"}`),
	})
	msg := err.Error()
	if !strings.Contains(msg, `status=400`) {
		t.Fatalf("missing status in error: %q", msg)
	}
	if !strings.Contains(msg, `request_id="req_123"`) {
		t.Fatalf("missing request id in error: %q", msg)
	}
	if strings.Contains(msg, "secret") {
		t.Fatalf("error should redact body, got: %q", msg)
	}
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("expected HTTPStatusError, got %T", err)
	}
	if statusErr.StatusCode != 400 || statusErr.RequestID != "req_123" || statusErr.BodyLen == 0 {
		t.Fatalf("unexpected status error: %+v", statusErr)
	}
}

func TestRequestID(t *testing.T) {
	t.Parallel()

	h := http.Header{}
	h.Set("x-request-id", "x1")
	if got := RequestID(h); got != "x1" {
		t.Fatalf("request id got=%q want=%q", got, "x1")
	}
}

func TestRequestIDFallbackKeys(t *testing.T) {
	t.Parallel()

	h := http.Header{}
	h.Set("request-id", "r1")
	if got := RequestID(h); got != "r1" {
		t.Fatalf("request id got=%q want=%q", got, "r1")
	}
}

func TestDecodeSchema(t *testing.T) {
	t.Parallel()

	if _, err := DecodeSchema(nil); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := DecodeSchema(json.RawMessage(`{`)); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := DecodeSchema(json.RawMessage(`{"type":"object"}`)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostJSONSuccess(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Custom"); got != "v1" {
			t.Fatalf("header got=%q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content-type got=%q", got)
		}
		_, _ = w.Write([]byte(`ok`))
	}))
	defer srv.Close()

	resp, err := PostJSON(context.Background(), NewHTTPClient(nil), srv.URL, map[string]string{
		"X-Custom": "v1",
	}, []byte(`{}`))
	if err != nil {
		t.Fatalf("PostJSON error: %v", err)
	}
	if string(resp.Body) != "ok" {
		t.Fatalf("body got=%q", string(resp.Body))
	}
}

func TestPostJSONBuildError(t *testing.T) {
	t.Parallel()

	_, err := PostJSON(context.Background(), NewHTTPClient(nil), "://bad", nil, nil)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestPostJSONExecuteError(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	})}
	_, err := PostJSON(context.Background(), client, "http://example.com", nil, nil)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestPostJSONReadBodyError(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       errReadCloser{},
		}, nil
	})}
	_, err := PostJSON(context.Background(), client, "http://example.com", nil, nil)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestPostJSONBodyTooLarge(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("too large"))
	}))
	defer srv.Close()

	_, err := PostJSONWithOptions(context.Background(), NewHTTPClient(nil), srv.URL, nil, nil, HTTPOptions{
		MaxBodyBytes: 2,
		Retry:        RetryConfig{MaxAttempts: 1},
	})
	if !errors.Is(err, ErrResponseBodyTooLarge) {
		t.Fatalf("expected body-too-large error, got %v", err)
	}
}

func TestPostJSONRetriesRetryableStatus(t *testing.T) {
	t.Parallel()

	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("retry"))
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	resp, err := PostJSONWithOptions(context.Background(), NewHTTPClient(nil), srv.URL, nil, nil, HTTPOptions{
		Retry: RetryConfig{MaxAttempts: 2},
	})
	if err != nil {
		t.Fatalf("PostJSONWithOptions error: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("attempts got=%d want=2", got)
	}
	if string(resp.Body) != "ok" {
		t.Fatalf("body got=%q", string(resp.Body))
	}
}

func TestPostJSONRetriesTransientTransportErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
	}{
		{name: "timeout", err: timeoutError{}},
		{name: "eof", err: io.EOF},
		{name: "conn-reset", err: syscall.ECONNRESET},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var attempts int32
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				if atomic.AddInt32(&attempts, 1) == 1 {
					return nil, tc.err
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader("ok")),
				}, nil
			})}

			resp, err := PostJSONWithOptions(context.Background(), client, "http://example.com", nil, nil, HTTPOptions{
				Retry: RetryConfig{
					MaxAttempts: 2,
					BaseDelay:   time.Nanosecond,
					MaxDelay:    time.Nanosecond,
				},
			})
			if err != nil {
				t.Fatalf("PostJSONWithOptions error: %v", err)
			}
			if got := atomic.LoadInt32(&attempts); got != 2 {
				t.Fatalf("attempts got=%d want=2", got)
			}
			if string(resp.Body) != "ok" {
				t.Fatalf("body got=%q", string(resp.Body))
			}
		})
	}
}

func TestHelpers(t *testing.T) {
	t.Parallel()

	if got := StripCodeFence("```json\n{\"x\":1}\n```"); got != "{\"x\":1}" {
		t.Fatalf("strip got=%q", got)
	}
	if got := NormalizeSchemaName(" Answer-Relevancy "); got != "answer_relevancy" {
		t.Fatalf("schema name got=%q", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (errReadCloser) Close() error             { return nil }

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }
