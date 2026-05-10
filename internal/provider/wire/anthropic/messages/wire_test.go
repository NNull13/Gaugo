package messages

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nnull13/gaugo/internal/provider/wire"
)

func TestEvaluateJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "anthropic-key" {
			t.Fatalf("x-api-key got=%q", got)
		}
		if got := r.Header.Get("anthropic-version"); got != "2023-06-01" {
			t.Fatalf("anthropic-version got=%q", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if _, ok := body["output_config"]; !ok {
			t.Fatalf("missing output_config")
		}
		outputConfig, ok := body["output_config"].(map[string]any)
		if !ok {
			t.Fatalf("output_config type got=%T", body["output_config"])
		}
		format, ok := outputConfig["format"].(map[string]any)
		if !ok {
			t.Fatalf("format type got=%T", outputConfig["format"])
		}
		if format["type"] != "json_schema" {
			t.Fatalf("format type got=%v", format["type"])
		}
		if _, ok := format["schema"]; !ok {
			t.Fatalf("missing schema")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"claude-test",
			"content":[{"type":"text","text":"{\"pass\":true}"}]
		}`))
	}))
	defer srv.Close()

	res, err := EvaluateJSON(context.Background(), Config{
		APIKey:     "anthropic-key",
		Model:      "claude-test",
		BaseURL:    srv.URL,
		APIVersion: "2023-06-01",
	}, wire.EvalRequest{
		Metric:       "Faithfulness",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err != nil {
		t.Fatalf("EvaluateJSON error: %v", err)
	}
	if got, want := strings.TrimSpace(string(res.RawJSON)), `{"pass":true}`; got != want {
		t.Fatalf("raw json got=%q want=%q", got, want)
	}
}

func TestEvaluateJSONStatusError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:  "anthropic-key",
		BaseURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "Faithfulness",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "status=429") {
		t.Fatalf("error missing status: %v", err)
	}
}

func TestEvaluateJSONMissingAPIKey(t *testing.T) {
	t.Parallel()

	_, err := EvaluateJSON(context.Background(), Config{}, wire.EvalRequest{
		Metric:       "Faithfulness",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestEvaluateJSONEmptyContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"claude","content":[]}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:  "anthropic-key",
		BaseURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "Faithfulness",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestEvaluateJSONInvalidSchema(t *testing.T) {
	t.Parallel()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:  "anthropic-key",
		BaseURL: "https://example.com",
	}, wire.EvalRequest{
		Metric:       "Faithfulness",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{`),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestEvaluateJSONDecodeError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:  "anthropic-key",
		BaseURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "Faithfulness",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestEvaluateJSONStopReasonFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"claude","stop_reason":"max_tokens","content":[{"type":"text","text":"{\"x\""}]}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:      "anthropic-key",
		EndpointURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "Faithfulness",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("expected truncated error, got %v", err)
	}
}

func TestEvaluateJSONInvalidJSONPayload(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"claude","content":[{"type":"text","text":"not-json"}]}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:      "anthropic-key",
		EndpointURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "Faithfulness",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err == nil || !strings.Contains(err.Error(), "invalid json payload") {
		t.Fatalf("expected invalid json payload error, got %v", err)
	}
}
