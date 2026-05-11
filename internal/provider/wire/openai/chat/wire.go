package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/nnull13/gaugo/internal/provider"
	"github.com/nnull13/gaugo/internal/provider/wire"
)

type Config struct {
	Provider        string
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
	providerName := wire.ProviderLabel(cfg.Provider, provider.OpenAI)
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return wire.EvalResult{}, provider.ProviderAPIKeyRequiredError(providerName)
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = provider.OpenAIDefaultModel
	}
	endpoint := wire.EndpointURL(cfg.EndpointURL, cfg.BaseURL, provider.OpenAIBaseURL, provider.OpenAIChatCompletionsPath)

	schema, err := wire.DecodeSchema(req.Schema)
	if err != nil {
		return wire.EvalResult{}, err
	}

	var body []byte
	body, err = json.Marshal(requestBody{
		Model:       model,
		Temperature: 0,
		Messages: []message{
			{Role: wire.RoleSystem, Content: req.Instructions},
			{Role: wire.RoleUser, Content: req.UserPrompt},
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
		return wire.EvalResult{}, wire.MarshalRequestError(providerName, err)
	}

	var resp wire.HTTPResponse
	resp, err = wire.PostJSONWithOptions(ctx, wire.NewHTTPClient(cfg.HTTPClient), endpoint, map[string]string{
		wire.HeaderAuthorization: wire.AuthBearerPrefix + apiKey,
		wire.HeaderContentType:   wire.ContentTypeJSON,
	}, body, wire.HTTPOptions{Retry: cfg.Retry, MaxBodyBytes: cfg.MaxResponseBody})
	if err != nil {
		return wire.EvalResult{}, wire.JudgeRequestError(providerName, err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return wire.EvalResult{}, wire.StatusError(providerName, resp)
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
		return wire.EvalResult{}, wire.DecodeResponseWrapError(providerName, err)
	}
	if len(parsed.Choices) == 0 {
		return wire.EvalResult{}, wire.DecodeResponseError(providerName, "no choices returned")
	}
	if strings.EqualFold(parsed.Choices[0].FinishReason, "length") {
		return wire.EvalResult{}, wire.DecodeResponseError(providerName, "output truncated by token limit")
	}
	if strings.TrimSpace(parsed.Choices[0].Message.Refusal) != "" {
		return wire.EvalResult{}, wire.DecodeResponseError(providerName, wire.ErrRefusal)
	}

	content := wire.StripCodeFence(parsed.Choices[0].Message.Content)
	if strings.TrimSpace(content) == "" {
		return wire.EvalResult{}, wire.DecodeResponseError(providerName, "empty message content")
	}
	rawJSON := []byte(strings.TrimSpace(content))
	if !json.Valid(rawJSON) {
		return wire.EvalResult{}, wire.DecodeResponseError(providerName, wire.ErrInvalidJSONPayload)
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
