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

func TestEvaluateJSONSanitizesNumericSchemaBounds(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		outputConfig, ok := body["output_config"].(map[string]any)
		if !ok {
			t.Fatalf("output_config type got=%T", body["output_config"])
		}
		format, ok := outputConfig["format"].(map[string]any)
		if !ok {
			t.Fatalf("format type got=%T", outputConfig["format"])
		}
		schema, ok := format["schema"].(map[string]any)
		if !ok {
			t.Fatalf("schema type got=%T", format["schema"])
		}
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("properties type got=%T", schema["properties"])
		}
		score, ok := properties["score"].(map[string]any)
		if !ok {
			t.Fatalf("score schema type got=%T", properties["score"])
		}
		if _, ok := score["minimum"]; ok {
			t.Fatalf("score schema should not include minimum: %#v", score)
		}
		if _, ok := score["maximum"]; ok {
			t.Fatalf("score schema should not include maximum: %#v", score)
		}
		if got := score["type"]; got != "number" {
			t.Fatalf("score type got=%v want=number", got)
		}
		issues, ok := properties["issues"].(map[string]any)
		if !ok {
			t.Fatalf("issues schema type got=%T", properties["issues"])
		}
		items, ok := issues["items"].(map[string]any)
		if !ok {
			t.Fatalf("issues items type got=%T", issues["items"])
		}
		itemProperties, ok := items["properties"].(map[string]any)
		if !ok {
			t.Fatalf("issues item properties type got=%T", items["properties"])
		}
		severity, ok := itemProperties["severity"].(map[string]any)
		if !ok {
			t.Fatalf("severity schema type got=%T", itemProperties["severity"])
		}
		if _, ok := severity["minimum"]; ok {
			t.Fatalf("nested severity schema should not include minimum: %#v", severity)
		}
		if _, ok := severity["maximum"]; ok {
			t.Fatalf("nested severity schema should not include maximum: %#v", severity)
		}
		confidence, ok := properties["confidence"].(map[string]any)
		if !ok {
			t.Fatalf("confidence schema type got=%T", properties["confidence"])
		}
		anyOf, ok := confidence["anyOf"].([]any)
		if !ok || len(anyOf) == 0 {
			t.Fatalf("confidence anyOf got=%#v", confidence["anyOf"])
		}
		numberBranch, ok := anyOf[0].(map[string]any)
		if !ok {
			t.Fatalf("confidence anyOf branch type got=%T", anyOf[0])
		}
		if _, ok := numberBranch["minimum"]; ok {
			t.Fatalf("nested anyOf branch should not include minimum: %#v", numberBranch)
		}
		if _, ok := numberBranch["maximum"]; ok {
			t.Fatalf("nested anyOf branch should not include maximum: %#v", numberBranch)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"claude-test",
			"content":[{"type":"text","text":"{\"score\":0.8,\"reason\":\"relevant\"}"}]
		}`))
	}))
	defer srv.Close()

	_, err := EvaluateJSON(context.Background(), Config{
		APIKey:  "anthropic-key",
		Model:   "claude-test",
		BaseURL: srv.URL,
	}, wire.EvalRequest{
		Metric:       "AnswerRelevancy",
		Instructions: "sys",
		UserPrompt:   "prompt",
		Schema: json.RawMessage(`{
			"type":"object",
			"additionalProperties":false,
			"required":["score","reason"],
			"properties":{
				"score":{"type":"number","minimum":0,"maximum":1},
				"reason":{"type":"string"},
				"issues":{
					"type":"array",
					"items":{
						"type":"object",
						"properties":{
							"severity":{"type":"number","minimum":1,"maximum":5},
							"label":{"type":"string"}
						}
					}
				},
				"confidence":{"anyOf":[{"type":"number","minimum":0,"maximum":1},{"type":"null"}]}
			}
		}`),
	})
	if err != nil {
		t.Fatalf("EvaluateJSON error: %v", err)
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
