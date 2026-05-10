package responses

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nnull13/gaugo/internal/provider/wire"
)

func TestEvaluateJSONWithOutputText(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization header got=%q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"grok-test","output_text":"{\"score\":1}"}`))
	}))
	defer srv.Close()

	res, err := EvaluateJSON(context.Background(), Config{
		APIKey:  "test-key",
		Model:   "grok-test",
		BaseURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "AnswerRelevancy",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err != nil {
		t.Fatalf("EvaluateJSON error: %v", err)
	}
	if got, want := strings.TrimSpace(string(res.RawJSON)), `{"score":1}`; got != want {
		t.Fatalf("raw json got=%q want=%q", got, want)
	}
}

func TestEvaluateJSONWithMessageOutput(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"grok-test",
			"output":[
				{"type":"message","content":[{"type":"output_text","text":"{\"score\":0.8}"}]}
			]
		}`))
	}))
	defer srv.Close()

	res, err := EvaluateJSON(context.Background(), Config{
		APIKey:  "test-key",
		BaseURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "AnswerRelevancy",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err != nil {
		t.Fatalf("EvaluateJSON error: %v", err)
	}
	if got, want := strings.TrimSpace(string(res.RawJSON)), `{"score":0.8}`; got != want {
		t.Fatalf("raw json got=%q want=%q", got, want)
	}
}

func TestEvaluateJSONStatusError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:  "test-key",
		BaseURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "AnswerRelevancy",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "status=401") {
		t.Fatalf("error missing status: %v", err)
	}
}

func TestEvaluateJSONMissingAPIKey(t *testing.T) {
	t.Parallel()

	_, err := EvaluateJSON(context.Background(), Config{}, wire.EvalRequest{
		Metric:       "AnswerRelevancy",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestEvaluateJSONEmptyOutput(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"m1","output":[]}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:  "test-key",
		BaseURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "AnswerRelevancy",
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
		Metric:       "AnswerRelevancy",
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
		APIKey:  "test-key",
		BaseURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "AnswerRelevancy",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestEvaluateJSONIncompleteOutput(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"m1","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:      "test-key",
		EndpointURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "AnswerRelevancy",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("expected incomplete error, got %v", err)
	}
}

func TestEvaluateJSONInvalidJSONPayload(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"m1","output_text":"not-json"}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:      "test-key",
		EndpointURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "AnswerRelevancy",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err == nil || !strings.Contains(err.Error(), "invalid json payload") {
		t.Fatalf("expected invalid json payload error, got %v", err)
	}
}
