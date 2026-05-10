package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nnull13/gaugo"
)

func TestModeNative(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"native-model","message":{"content":"{\"mode\":\"native\"}"}}`))
	}))
	defer srv.Close()

	j, err := New(Config{
		Mode:        ModeNative,
		EndpointURL: srv.URL,
		Model:       "native-model",
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	res, err := j.EvaluateJSON(context.Background(), sampleJudgeRequest())
	if err != nil {
		t.Fatalf("EvaluateJSON error: %v", err)
	}
	if got, want := strings.TrimSpace(string(res.RawJSON)), `{"mode":"native"}`; got != want {
		t.Fatalf("raw json got=%q want=%q", got, want)
	}
}

func TestModeOpenAIChat(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got == "" {
			t.Fatalf("expected authorization header")
		}
		_, _ = w.Write([]byte(`{"model":"chat-model","choices":[{"message":{"content":"{\"mode\":\"openai_chat\"}"}}]}`))
	}))
	defer srv.Close()

	j, err := New(Config{
		Mode:                 ModeOpenAI,
		OpenAICompatEndpoint: OpenAIEndpointChat,
		EndpointURL:          srv.URL,
		Model:                "chat-model",
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	res, err := j.EvaluateJSON(context.Background(), sampleJudgeRequest())
	if err != nil {
		t.Fatalf("EvaluateJSON error: %v", err)
	}
	if got, want := strings.TrimSpace(string(res.RawJSON)), `{"mode":"openai_chat"}`; got != want {
		t.Fatalf("raw json got=%q want=%q", got, want)
	}
}

func TestModeOpenAIResponses(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"resp-model","output_text":"{\"mode\":\"openai_responses\"}"}`))
	}))
	defer srv.Close()

	j, err := New(Config{
		Mode:                 ModeOpenAI,
		OpenAICompatEndpoint: OpenAIEndpointResponses,
		EndpointURL:          srv.URL,
		Model:                "resp-model",
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	res, err := j.EvaluateJSON(context.Background(), sampleJudgeRequest())
	if err != nil {
		t.Fatalf("EvaluateJSON error: %v", err)
	}
	if got, want := strings.TrimSpace(string(res.RawJSON)), `{"mode":"openai_responses"}`; got != want {
		t.Fatalf("raw json got=%q want=%q", got, want)
	}
}

func TestModeAnthropic(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got == "" {
			t.Fatalf("expected x-api-key header")
		}
		_, _ = w.Write([]byte(`{"model":"anthropic-model","content":[{"type":"text","text":"{\"mode\":\"anthropic\"}"}]}`))
	}))
	defer srv.Close()

	j, err := New(Config{
		Mode:        ModeAnthropic,
		EndpointURL: srv.URL,
		Model:       "anthropic-model",
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	res, err := j.EvaluateJSON(context.Background(), sampleJudgeRequest())
	if err != nil {
		t.Fatalf("EvaluateJSON error: %v", err)
	}
	if got, want := strings.TrimSpace(string(res.RawJSON)), `{"mode":"anthropic"}`; got != want {
		t.Fatalf("raw json got=%q want=%q", got, want)
	}
}

func TestInvalidMode(t *testing.T) {
	t.Parallel()

	_, err := New(Config{Mode: Mode("invalid")})
	if err == nil || !strings.Contains(err.Error(), "invalid ollama config") {
		t.Fatalf("expected invalid config error, got: %v", err)
	}
}

func TestInvalidOpenAIEndpoint(t *testing.T) {
	t.Parallel()

	_, err := New(Config{
		Mode:                 ModeOpenAI,
		OpenAICompatEndpoint: OpenAIEndpoint("bad"),
	})
	if err == nil || !strings.Contains(err.Error(), "invalid ollama config") {
		t.Fatalf("expected invalid config error, got: %v", err)
	}
}

func TestOpenAIEndpointRequiresModeOpenAI(t *testing.T) {
	t.Parallel()

	_, err := New(Config{
		Mode:                 ModeNative,
		OpenAICompatEndpoint: OpenAIEndpointResponses,
	})
	if err == nil || !strings.Contains(err.Error(), "invalid ollama config") {
		t.Fatalf("expected invalid config error, got: %v", err)
	}
}

func TestInvalidBaseURL(t *testing.T) {
	t.Parallel()

	_, err := New(Config{
		Mode:    ModeNative,
		BaseURL: "://bad",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid ollama config") {
		t.Fatalf("expected invalid config error, got: %v", err)
	}
}

func sampleJudgeRequest() gaugo.JudgeRequest {
	return gaugo.JudgeRequest{
		Metric:       "Faithfulness",
		Question:     "Q",
		Answer:       "A",
		ContextDocs:  []gaugo.Document{{ID: "d1", Text: "ctx"}},
		Instructions: "Return JSON only",
		Schema:       json.RawMessage(`{"type":"object","properties":{"mode":{"type":"string"}},"required":["mode"]}`),
	}
}
