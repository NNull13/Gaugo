package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nnull13/gaugo/internal/provider"
	"github.com/nnull13/gaugo/internal/provider/wire"
)

type Config struct {
	APIKey          string
	Model           string
	BaseURL         string
	EndpointURL     string
	HTTPClient      *http.Client
	Retry           wire.RetryConfig
	MaxResponseBody int64
}

type requestBody struct {
	Model          string         `json:"model"`
	Temperature    int            `json:"temperature"`
	Messages       []message      `json:"messages"`
	ResponseFormat responseFormat `json:"response_format"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type       string      `json:"type"`
	JSONSchema schemaField `json:"json_schema"`
}

type schemaField struct {
	Name   string `json:"name"`
	Strict bool   `json:"strict"`
	Schema any    `json:"schema"`
}

func EvaluateJSON(ctx context.Context, cfg Config, req wire.EvalRequest) (wire.EvalResult, error) {
	var (
		err    error
		schema any
		body   []byte
		resp   wire.HTTPResponse
	)

	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return wire.EvalResult{}, provider.ProviderAPIKeyRequiredError(provider.OpenAI)
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = provider.OpenAIDefaultModel
	}
	endpoint := endpointURL(cfg.EndpointURL, cfg.BaseURL, provider.OpenAIBaseURL, provider.OpenAIChatCompletionsPath)

	schema, err = wire.DecodeSchema(req.Schema)
	if err != nil {
		return wire.EvalResult{}, err
	}

	body, err = json.Marshal(requestBody{
		Model:       model,
		Temperature: 0,
		Messages: []message{
			{Role: "system", Content: req.Instructions},
			{Role: "user", Content: req.UserPrompt},
		},
		ResponseFormat: responseFormat{
			Type: wire.SchemaTypeJSONSchema,
			JSONSchema: schemaField{
				Name:   wire.NormalizeSchemaName(req.Metric),
				Strict: true,
				Schema: schema,
			},
		},
	})
	if err != nil {
		return wire.EvalResult{}, fmt.Errorf("marshal openai request: %w", err)
	}

	resp, err = wire.PostJSONWithOptions(ctx, wire.NewHTTPClient(cfg.HTTPClient), endpoint, map[string]string{
		wire.HeaderAuthorization: wire.AuthBearerPrefix + apiKey,
		wire.HeaderContentType:   wire.ContentTypeJSON,
	}, body, wire.HTTPOptions{Retry: cfg.Retry, MaxBodyBytes: cfg.MaxResponseBody})
	if err != nil {
		return wire.EvalResult{}, fmt.Errorf("openai judge request failed: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return wire.EvalResult{}, wire.StatusError(provider.OpenAI, resp)
	}

	var parsed struct {
		Model   string `json:"model"`
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
				Refusal string `json:"refusal"`
			} `json:"message"`
		} `json:"choices"`
	}
	err = json.Unmarshal(resp.Body, &parsed)
	if err != nil {
		return wire.EvalResult{}, fmt.Errorf("decode openai response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return wire.EvalResult{}, fmt.Errorf("decode openai response: no choices returned")
	}
	if strings.EqualFold(parsed.Choices[0].FinishReason, "length") {
		return wire.EvalResult{}, fmt.Errorf("decode openai response: output truncated by token limit")
	}
	if strings.TrimSpace(parsed.Choices[0].Message.Refusal) != "" {
		return wire.EvalResult{}, fmt.Errorf("decode openai response: model refusal")
	}

	content := wire.StripCodeFence(parsed.Choices[0].Message.Content)
	if strings.TrimSpace(content) == "" {
		return wire.EvalResult{}, fmt.Errorf("decode openai response: empty message content")
	}
	rawJSON := []byte(strings.TrimSpace(content))
	if !json.Valid(rawJSON) {
		return wire.EvalResult{}, fmt.Errorf("decode openai response: invalid json payload")
	}

	outModel := strings.TrimSpace(parsed.Model)
	if outModel == "" {
		outModel = model
	}

	return wire.EvalResult{
		RawJSON:   rawJSON,
		Model:     outModel,
		Latency:   resp.Latency,
		RequestID: wire.RequestID(resp.Header),
	}, nil
}

func endpointURL(endpointURL, baseURL, defaultBaseURL, path string) string {
	if endpoint := strings.TrimSpace(endpointURL); endpoint != "" {
		return endpoint
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = defaultBaseURL
	}
	return base + path
}
