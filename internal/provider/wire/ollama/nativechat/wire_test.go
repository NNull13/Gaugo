package nativechat

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
		if got := r.Header.Get("Authorization"); got != "Bearer local-key" {
			t.Fatalf("authorization got=%q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if _, ok := body["format"]; !ok {
			t.Fatalf("missing format schema")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"llama-test","message":{"content":"{\"ok\":true}"}}`))
	}))
	defer srv.Close()

	res, err := EvaluateJSON(context.Background(), Config{
		APIKey:  "local-key",
		Model:   "llama-test",
		BaseURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "Faithfulness",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err != nil {
		t.Fatalf("EvaluateJSON error: %v", err)
	}
	if got, want := strings.TrimSpace(string(res.RawJSON)), `{"ok":true}`; got != want {
		t.Fatalf("raw json got=%q want=%q", got, want)
	}
}

func TestEvaluateJSONStatusError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal"}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
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
	if !strings.Contains(err.Error(), "status=500") {
		t.Fatalf("error missing status: %v", err)
	}
}

func TestEvaluateJSONInvalidSchema(t *testing.T) {
	t.Parallel()

	_, err := EvaluateJSON(context.Background(), Config{
		BaseURL: "http://127.0.0.1:11434/api/chat",
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

func TestEvaluateJSONEmptyMessageContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"m","message":{"content":" "}}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
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

func TestEvaluateJSONInvalidJSONPayload(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"m","message":{"content":"not-json"}}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
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
