package responses

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nnull13/gaugo/internal/provider"
	"github.com/nnull13/gaugo/internal/provider/wire"
)

const wireName = provider.ResponsesWireName

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
	Model       string          `json:"model"`
	Temperature int             `json:"temperature"`
	Input       []inputMessage  `json:"input"`
	Text        textInstruction `json:"text"`
}

type inputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type textInstruction struct {
	Format schemaFormat `json:"format"`
}

type schemaFormat struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Schema any    `json:"schema"`
	Strict bool   `json:"strict"`
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
		return wire.EvalResult{}, provider.APIKeyRequiredError()
	}

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = provider.OpenAIDefaultModel
	}
	endpoint := endpointURL(cfg.EndpointURL, cfg.BaseURL, provider.OpenAIBaseURL, provider.ResponsesPath)

	schema, err = wire.DecodeSchema(req.Schema)
	if err != nil {
		return wire.EvalResult{}, err
	}

	body, err = json.Marshal(requestBody{
		Model:       model,
		Temperature: 0,
		Input: []inputMessage{
			{Role: "system", Content: req.Instructions},
			{Role: "user", Content: req.UserPrompt},
		},
		Text: textInstruction{
			Format: schemaFormat{
				Type:   wire.SchemaTypeJSONSchema,
				Name:   wire.NormalizeSchemaName(req.Metric),
				Schema: schema,
				Strict: true,
			},
		},
	})
	if err != nil {
		return wire.EvalResult{}, fmt.Errorf("marshal responses request: %w", err)
	}

	resp, err = wire.PostJSONWithOptions(ctx, wire.NewHTTPClient(cfg.HTTPClient), endpoint, map[string]string{
		wire.HeaderAuthorization: wire.AuthBearerPrefix + apiKey,
		wire.HeaderContentType:   wire.ContentTypeJSON,
	}, body, wire.HTTPOptions{Retry: cfg.Retry, MaxBodyBytes: cfg.MaxResponseBody})
	if err != nil {
		return wire.EvalResult{}, fmt.Errorf("%s judge request failed: %w", wireName, err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return wire.EvalResult{}, wire.StatusError(wireName, resp)
	}

	var parsed struct {
		Model             string `json:"model"`
		Status            string `json:"status"`
		IncompleteDetails struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
		OutputText string `json:"output_text"`
		Output     []struct {
			Type    string `json:"type"`
			Text    string `json:"text"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	err = json.Unmarshal(resp.Body, &parsed)
	if err != nil {
		return wire.EvalResult{}, fmt.Errorf("decode responses response: %w", err)
	}
	if strings.EqualFold(parsed.Status, "incomplete") {
		reason := strings.TrimSpace(parsed.IncompleteDetails.Reason)
		if reason == "" {
			reason = "unknown"
		}
		return wire.EvalResult{}, fmt.Errorf("decode responses response: incomplete output reason=%q", reason)
	}

	content := strings.TrimSpace(parsed.OutputText)
	if content == "" {
		content = extractOutputText(parsed.Output)
	}
	content = wire.StripCodeFence(content)
	if strings.TrimSpace(content) == "" {
		return wire.EvalResult{}, fmt.Errorf("decode responses response: empty output text")
	}
	rawJSON := []byte(strings.TrimSpace(content))
	if !json.Valid(rawJSON) {
		return wire.EvalResult{}, fmt.Errorf("decode responses response: invalid json payload")
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

func extractOutputText(out []struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}) string {
	for _, item := range out {
		if strings.TrimSpace(item.Text) != "" && strings.EqualFold(item.Type, "output_text") {
			return item.Text
		}
		for _, c := range item.Content {
			if strings.EqualFold(c.Type, "refusal") {
				return ""
			}
			if strings.EqualFold(c.Type, "output_text") || strings.EqualFold(c.Type, "text") {
				if strings.TrimSpace(c.Text) != "" {
					return c.Text
				}
			}
		}
	}
	return ""
}
