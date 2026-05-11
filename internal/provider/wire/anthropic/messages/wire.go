package messages

import (
	"context"
	"encoding/json"
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
	APIVersion      string
	MaxTokens       int
	HTTPClient      *http.Client
	Retry           wire.RetryConfig
	MaxResponseBody int64
}

type requestBody struct {
	Model        string       `json:"model"`
	MaxTokens    int          `json:"max_tokens"`
	System       string       `json:"system,omitempty"`
	Messages     []message    `json:"messages"`
	OutputConfig outputConfig `json:"output_config"`
	Temperature  int          `json:"temperature"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type outputConfig struct {
	Format formatSpec `json:"format"`
}

type formatSpec struct {
	Type   string `json:"type"`
	Schema any    `json:"schema"`
}

func EvaluateJSON(ctx context.Context, cfg Config, req wire.EvalRequest) (wire.EvalResult, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return wire.EvalResult{}, provider.ProviderAPIKeyRequiredError(provider.Anthropic)
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = provider.AnthropicDefaultModel
	}
	endpoint := wire.EndpointURL(cfg.EndpointURL, cfg.BaseURL, provider.AnthropicBaseURL, provider.AnthropicMessagesPath)
	apiVersion := strings.TrimSpace(cfg.APIVersion)
	if apiVersion == "" {
		apiVersion = provider.AnthropicDefaultAPIVersion
	}
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = provider.AnthropicDefaultMaxTokens
	}

	schema, err := wire.DecodeSchema(req.Schema)
	if err != nil {
		return wire.EvalResult{}, err
	}
	schema = sanitizeSchema(schema)

	var body []byte
	body, err = json.Marshal(requestBody{
		Model:       model,
		MaxTokens:   maxTokens,
		System:      req.Instructions,
		Temperature: 0,
		Messages: []message{
			{Role: wire.RoleUser, Content: req.UserPrompt},
		},
		OutputConfig: outputConfig{
			Format: formatSpec{
				Type:   wire.SchemaTypeJSONSchema,
				Schema: schema,
			},
		},
	})
	if err != nil {
		return wire.EvalResult{}, wire.MarshalRequestError(provider.Anthropic, err)
	}

	var resp wire.HTTPResponse
	resp, err = wire.PostJSONWithOptions(ctx, wire.NewHTTPClient(cfg.HTTPClient), endpoint, map[string]string{
		wire.HeaderAnthropicAPIKey:  apiKey,
		wire.HeaderAnthropicVersion: apiVersion,
		wire.HeaderContentType:      wire.ContentTypeJSON,
	}, body, wire.HTTPOptions{Retry: cfg.Retry, MaxBodyBytes: cfg.MaxResponseBody})
	if err != nil {
		return wire.EvalResult{}, wire.JudgeRequestError(provider.Anthropic, err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return wire.EvalResult{}, wire.StatusError(provider.Anthropic, resp)
	}

	var parsed struct {
		Model      string `json:"model"`
		StopReason string `json:"stop_reason"`
		Content    []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	err = json.Unmarshal(resp.Body, &parsed)
	if err != nil {
		return wire.EvalResult{}, wire.DecodeResponseWrapError(provider.Anthropic, err)
	}
	if strings.EqualFold(parsed.StopReason, "refusal") {
		return wire.EvalResult{}, wire.DecodeResponseError(provider.Anthropic, wire.ErrRefusal)
	}
	if strings.EqualFold(parsed.StopReason, "max_tokens") {
		return wire.EvalResult{}, wire.DecodeResponseError(provider.Anthropic, "output truncated by max_tokens")
	}

	content := ""
	for _, block := range parsed.Content {
		if strings.EqualFold(block.Type, "text") && strings.TrimSpace(block.Text) != "" {
			content = block.Text
			break
		}
	}
	content = wire.StripCodeFence(content)
	if strings.TrimSpace(content) == "" {
		return wire.EvalResult{}, wire.DecodeResponseError(provider.Anthropic, "empty text content")
	}
	rawJSON := []byte(strings.TrimSpace(content))
	if !json.Valid(rawJSON) {
		return wire.EvalResult{}, wire.DecodeResponseError(provider.Anthropic, wire.ErrInvalidJSONPayload)
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

func sanitizeSchema(schema any) any {
	switch v := schema.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, value := range v {
			if key == "minimum" || key == "maximum" {
				continue
			}
			out[key] = sanitizeSchema(value)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, value := range v {
			out[i] = sanitizeSchema(value)
		}
		return out
	default:
		return schema
	}
}
