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

const (
	wireName                 = provider.ResponsesWireName
	responseStatusIncomplete = "incomplete"
	incompleteReasonUnknown  = "unknown"
	incompleteOutputFormat   = "incomplete output reason=%q"
	outputTypeOutputText     = "output_text"
	outputTypeRefusal        = "refusal"
	outputTypeText           = "text"
	errEmptyOutputText       = "empty output text"
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
	providerName := wire.ProviderLabel(cfg.Provider, provider.OpenAI)
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return wire.EvalResult{}, provider.ProviderAPIKeyRequiredError(providerName)
	}

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = provider.OpenAIDefaultModel
	}
	endpoint := wire.EndpointURL(cfg.EndpointURL, cfg.BaseURL, provider.OpenAIBaseURL, provider.ResponsesPath)

	schema, err := wire.DecodeSchema(req.Schema)
	if err != nil {
		return wire.EvalResult{}, err
	}

	var body []byte
	body, err = json.Marshal(requestBody{
		Model:       model,
		Temperature: 0,
		Input: []inputMessage{
			{Role: wire.RoleSystem, Content: req.Instructions},
			{Role: wire.RoleUser, Content: req.UserPrompt},
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
		return wire.EvalResult{}, wire.MarshalRequestErrorForWire(providerName, wireName, err)
	}

	var resp wire.HTTPResponse
	resp, err = wire.PostJSONWithOptions(ctx, wire.NewHTTPClient(cfg.HTTPClient), endpoint, map[string]string{
		wire.HeaderAuthorization: wire.AuthBearerPrefix + apiKey,
		wire.HeaderContentType:   wire.ContentTypeJSON,
	}, body, wire.HTTPOptions{Retry: cfg.Retry, MaxBodyBytes: cfg.MaxResponseBody})
	if err != nil {
		return wire.EvalResult{}, wire.JudgeRequestErrorForWire(providerName, wireName, err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return wire.EvalResult{}, wire.StatusErrorForWire(providerName, wireName, resp)
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
	requestID := wire.RequestID(resp.Header)
	if err != nil {
		return wire.EvalResult{}, wire.DecodeResponseWrapErrorForWire(providerName, wireName, err, requestID)
	}
	if strings.EqualFold(parsed.Status, responseStatusIncomplete) {
		reason := strings.TrimSpace(parsed.IncompleteDetails.Reason)
		if reason == "" {
			reason = incompleteReasonUnknown
		}
		return wire.EvalResult{}, wire.DecodeResponseErrorForWire(providerName, wireName, fmt.Sprintf(incompleteOutputFormat, reason), requestID)
	}

	content := strings.TrimSpace(parsed.OutputText)
	if content == "" {
		extracted, refused := extractOutputText(parsed.Output)
		if refused {
			return wire.EvalResult{}, wire.DecodeResponseErrorForWire(providerName, wireName, wire.ErrRefusal, requestID)
		}
		content = extracted
	}
	content = wire.StripCodeFence(content)
	if strings.TrimSpace(content) == "" {
		return wire.EvalResult{}, wire.DecodeResponseErrorForWire(providerName, wireName, errEmptyOutputText, requestID)
	}
	rawJSON := []byte(strings.TrimSpace(content))
	if !json.Valid(rawJSON) {
		return wire.EvalResult{}, wire.DecodeResponseErrorForWire(providerName, wireName, wire.ErrInvalidJSONPayload, requestID)
	}

	outModel := strings.TrimSpace(parsed.Model)
	if outModel == "" {
		outModel = model
	}

	return wire.EvalResult{
		RawJSON:   rawJSON,
		Model:     outModel,
		Latency:   resp.Latency,
		RequestID: requestID,
	}, nil
}

func extractOutputText(out []struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}) (string, bool) {
	for _, item := range out {
		if strings.TrimSpace(item.Text) != "" && strings.EqualFold(item.Type, outputTypeOutputText) {
			return item.Text, false
		}
		for _, c := range item.Content {
			if strings.EqualFold(c.Type, outputTypeRefusal) {
				return "", true
			}
			if strings.EqualFold(c.Type, outputTypeOutputText) || strings.EqualFold(c.Type, outputTypeText) {
				if strings.TrimSpace(c.Text) != "" {
					return c.Text, false
				}
			}
		}
	}
	return "", false
}
