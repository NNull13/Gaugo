package chat

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
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization header got=%q", got)
		}
		if got := r.URL.Path; got != "/chat/completions" {
			t.Fatalf("path got=%q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["model"] != "gpt-test" {
			t.Fatalf("model got=%v", body["model"])
		}
		format, ok := body["response_format"].(map[string]any)
		if !ok {
			t.Fatalf("response_format type got=%T", body["response_format"])
		}
		if format["type"] != "json_schema" {
			t.Fatalf("response_format type got=%v", format["type"])
		}
		jsonSchema, ok := format["json_schema"].(map[string]any)
		if !ok {
			t.Fatalf("json_schema type got=%T", format["json_schema"])
		}
		if jsonSchema["strict"] != true {
			t.Fatalf("json_schema strict got=%v", jsonSchema["strict"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"model":"gpt-test","choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
	}))
	defer srv.Close()

	res, err := EvaluateJSON(context.Background(), Config{
		APIKey:  "test-key",
		Model:   "gpt-test",
		BaseURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "Faithfulness",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}}}`),
	})
	if err != nil {
		t.Fatalf("EvaluateJSON error: %v", err)
	}
	if got, want := strings.TrimSpace(string(res.RawJSON)), `{"ok":true}`; got != want {
		t.Fatalf("raw json got=%q want=%q", got, want)
	}
	if got, want := res.Model, "gpt-test"; got != want {
		t.Fatalf("model got=%q want=%q", got, want)
	}
}

func TestEvaluateJSONStatusError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("request-id", "req_123")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:  "test-key",
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
	if !strings.Contains(err.Error(), "status=400") {
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

func TestEvaluateJSONInvalidSchema(t *testing.T) {
	t.Parallel()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:  "test-key",
		BaseURL: "https://example.com",
	}, wire.EvalRequest{
		Metric:       "Faithfulness",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{`),
	})
	if err == nil {
		t.Fatalf("expected schema error")
	}
}

func TestEvaluateJSONNoChoices(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"gpt-test","choices":[]}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:  "test-key",
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

func TestEvaluateJSONEmptyContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"gpt-test","choices":[{"message":{"content":" "}}]}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:  "test-key",
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

func TestEvaluateJSONTruncatedOutput(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"gpt-test","choices":[{"finish_reason":"length","message":{"content":"{\"ok\""}}]}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:      "test-key",
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

func TestEvaluateJSONRefusal(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"gpt-test","choices":[{"message":{"refusal":"no"}}]}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:      "test-key",
		EndpointURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "Faithfulness",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err == nil || !strings.Contains(err.Error(), "refusal") {
		t.Fatalf("expected refusal error, got %v", err)
	}
}

func TestEvaluateJSONInvalidJSONPayload(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"gpt-test","choices":[{"message":{"content":"not-json"}}]}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:      "test-key",
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
