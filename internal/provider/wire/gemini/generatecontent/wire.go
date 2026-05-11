package generatecontent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	Contents         []content        `json:"contents"`
	GenerationConfig generationConfig `json:"generationConfig"`
}

type content struct {
	Role  string `json:"role"`
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type generationConfig struct {
	Temperature        int    `json:"temperature"`
	ResponseMIMEType   string `json:"responseMimeType"`
	ResponseJSONSchema any    `json:"responseJsonSchema"`
}

func EvaluateJSON(ctx context.Context, cfg Config, req wire.EvalRequest) (wire.EvalResult, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return wire.EvalResult{}, provider.ProviderAPIKeyRequiredError(provider.Gemini)
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = provider.GeminiDefaultModel
	}

	schema, err := wire.DecodeSchema(req.Schema)
	if err != nil {
		return wire.EvalResult{}, err
	}

	endpoint := buildEndpoint(strings.TrimSpace(cfg.EndpointURL), strings.TrimSpace(cfg.BaseURL), model)
	prompt := strings.TrimSpace(req.Instructions) + "\n\n" + strings.TrimSpace(req.UserPrompt)

	var body []byte
	body, err = json.Marshal(requestBody{
		Contents: []content{
			{
				Role: wire.RoleUser,
				Parts: []part{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: generationConfig{
			Temperature:        0,
			ResponseMIMEType:   wire.ContentTypeJSON,
			ResponseJSONSchema: schema,
		},
	})
	if err != nil {
		return wire.EvalResult{}, wire.MarshalRequestError(provider.Gemini, err)
	}

	var resp wire.HTTPResponse
	resp, err = wire.PostJSONWithOptions(ctx, wire.NewHTTPClient(cfg.HTTPClient), endpoint, map[string]string{
		wire.HeaderGeminiAPIKey: apiKey,
		wire.HeaderContentType:  wire.ContentTypeJSON,
	}, body, wire.HTTPOptions{Retry: cfg.Retry, MaxBodyBytes: cfg.MaxResponseBody})
	if err != nil {
		return wire.EvalResult{}, wire.JudgeRequestError(provider.Gemini, err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return wire.EvalResult{}, wire.StatusError(provider.Gemini, resp)
	}

	var parsed struct {
		Candidates []struct {
			FinishReason string `json:"finishReason"`
			Content      struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		ModelVersion string `json:"modelVersion"`
	}
	err = json.Unmarshal(resp.Body, &parsed)
	if err != nil {
		return wire.EvalResult{}, wire.DecodeResponseWrapError(provider.Gemini, err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return wire.EvalResult{}, wire.DecodeResponseError(provider.Gemini, "no text candidates returned")
	}
	if strings.EqualFold(parsed.Candidates[0].FinishReason, "MAX_TOKENS") {
		return wire.EvalResult{}, wire.DecodeResponseError(provider.Gemini, "output truncated by max tokens")
	}
	if strings.EqualFold(parsed.Candidates[0].FinishReason, "SAFETY") ||
		strings.EqualFold(parsed.Candidates[0].FinishReason, "BLOCKLIST") ||
		strings.EqualFold(parsed.Candidates[0].FinishReason, "PROHIBITED_CONTENT") {
		return wire.EvalResult{}, wire.DecodeResponseError(provider.Gemini, fmt.Sprintf("output blocked with finish reason %q", parsed.Candidates[0].FinishReason))
	}

	var sb strings.Builder
	for _, p := range parsed.Candidates[0].Content.Parts {
		if strings.TrimSpace(p.Text) == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(p.Text)
	}
	content := wire.StripCodeFence(sb.String())
	if strings.TrimSpace(content) == "" {
		return wire.EvalResult{}, wire.DecodeResponseError(provider.Gemini, "empty text content")
	}
	rawJSON := []byte(strings.TrimSpace(content))
	if !json.Valid(rawJSON) {
		return wire.EvalResult{}, wire.DecodeResponseError(provider.Gemini, wire.ErrInvalidJSONPayload)
	}

	outModel := strings.TrimSpace(parsed.ModelVersion)
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

func buildEndpoint(endpointURL, baseURL, model string) string {
	if endpointURL != "" {
		return endpointURL
	}
	if baseURL == "" {
		baseURL = provider.GeminiBaseURL
	}
	return strings.TrimRight(baseURL, "/") + "/v1beta/models/" + url.PathEscape(model) + ":generateContent"
}
