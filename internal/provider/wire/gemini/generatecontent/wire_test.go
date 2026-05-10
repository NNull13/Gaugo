package generatecontent

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
		if got := r.Header.Get("x-goog-api-key"); got != "gemini-key" {
			t.Fatalf("x-goog-api-key got=%q", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if _, ok := body["generationConfig"]; !ok {
			t.Fatalf("missing generationConfig")
		}
		cfg, ok := body["generationConfig"].(map[string]any)
		if !ok {
			t.Fatalf("generationConfig type got=%T", body["generationConfig"])
		}
		if got := cfg["responseMimeType"]; got != "application/json" {
			t.Fatalf("responseMimeType got=%v", got)
		}
		if _, ok := cfg["responseJsonSchema"]; !ok {
			t.Fatalf("missing responseJsonSchema")
		}
		if _, ok := cfg["responseFormat"]; ok {
			t.Fatalf("unexpected legacy responseFormat in request")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"modelVersion":"gemini-test",
			"candidates":[{"content":{"parts":[{"text":"{\"ok\":true}"}]}}]
		}`))
	}))
	defer srv.Close()

	res, err := EvaluateJSON(context.Background(), Config{
		APIKey:      "gemini-key",
		Model:       "gemini-test",
		EndpointURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "AnswerRelevancy",
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
	if got, want := res.Model, "gemini-test"; got != want {
		t.Fatalf("model got=%q want=%q", got, want)
	}
}

func TestEvaluateJSONStatusError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:      "gemini-key",
		EndpointURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "AnswerRelevancy",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "status=403") {
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

func TestEvaluateJSONNoCandidates(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"modelVersion":"gemini-test","candidates":[]}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:      "gemini-key",
		EndpointURL: srv.URL,
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
		APIKey:  "gemini-key",
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
		APIKey:      "gemini-key",
		EndpointURL: srv.URL,
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

func TestEvaluateJSONFinishReasonFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"modelVersion":"gemini-test",
			"candidates":[{"finishReason":"MAX_TOKENS","content":{"parts":[{"text":"{\"x\""}]}}]
		}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:      "gemini-key",
		EndpointURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "AnswerRelevancy",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema:       json.RawMessage(`{"type":"object"}`),
	})
	if err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("expected truncated error, got %v", err)
	}
}

func TestBuildEndpoint(t *testing.T) {
	t.Parallel()

	if got := buildEndpoint("", "", "gemini-2.5-flash"); !strings.Contains(got, ":generateContent") {
		t.Fatalf("default endpoint got=%q", got)
	}
	if got := buildEndpoint("https://host/v1beta/models/x:generateContent", "https://ignored", "gemini"); got != "https://host/v1beta/models/x:generateContent" {
		t.Fatalf("passthrough endpoint got=%q", got)
	}
	if got := buildEndpoint("", "https://host", "gemini"); !strings.Contains(got, "/v1beta/models/gemini:generateContent") {
		t.Fatalf("base endpoint got=%q", got)
	}
}

func TestEvaluateJSONInvalidJSONPayload(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"modelVersion":"gemini-test",
			"candidates":[{"content":{"parts":[{"text":"not-json"}]}}]
		}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:      "gemini-key",
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
