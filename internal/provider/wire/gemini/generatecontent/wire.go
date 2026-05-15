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

const (
	wireName                      = "generate_content"
	finishReasonMaxTokens         = "MAX_TOKENS"
	finishReasonSafety            = "SAFETY"
	finishReasonBlocklist         = "BLOCKLIST"
	finishReasonProhibitedContent = "PROHIBITED_CONTENT"
	errNoTextCandidatesReturned   = "no text candidates returned"
	errOutputTruncatedMaxTokens   = "output truncated by max tokens"
	blockedOutputFormat           = "output blocked with finish reason %q"
	generateContentPathPrefix     = "/v1beta/models/"
	generateContentPathSuffix     = ":generateContent"
)

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
		return wire.EvalResult{}, wire.MarshalRequestErrorForWire(provider.Gemini, wireName, err)
	}

	var resp wire.HTTPResponse
	resp, err = wire.PostJSONWithOptions(ctx, wire.NewHTTPClient(cfg.HTTPClient), endpoint, map[string]string{
		wire.HeaderGeminiAPIKey: apiKey,
		wire.HeaderContentType:  wire.ContentTypeJSON,
	}, body, wire.HTTPOptions{Retry: cfg.Retry, MaxBodyBytes: cfg.MaxResponseBody})
	if err != nil {
		return wire.EvalResult{}, wire.JudgeRequestErrorForWire(provider.Gemini, wireName, err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return wire.EvalResult{}, wire.StatusErrorForWire(provider.Gemini, wireName, resp)
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
	requestID := wire.RequestID(resp.Header)
	if err != nil {
		return wire.EvalResult{}, wire.DecodeResponseWrapErrorForWire(provider.Gemini, wireName, err, requestID)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return wire.EvalResult{}, wire.DecodeResponseErrorForWire(wire.ErrorKindProviderResponse, provider.Gemini, wireName, errNoTextCandidatesReturned, requestID)
	}
	if strings.EqualFold(parsed.Candidates[0].FinishReason, finishReasonMaxTokens) {
		return wire.EvalResult{}, wire.DecodeResponseErrorForWire(wire.ErrorKindProviderTruncated, provider.Gemini, wireName, errOutputTruncatedMaxTokens, requestID)
	}
	if strings.EqualFold(parsed.Candidates[0].FinishReason, finishReasonSafety) ||
		strings.EqualFold(parsed.Candidates[0].FinishReason, finishReasonBlocklist) ||
		strings.EqualFold(parsed.Candidates[0].FinishReason, finishReasonProhibitedContent) {
		return wire.EvalResult{}, wire.DecodeResponseErrorForWire(wire.ErrorKindProviderRefusal, provider.Gemini, wireName, fmt.Sprintf(blockedOutputFormat, parsed.Candidates[0].FinishReason), requestID)
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
		return wire.EvalResult{}, wire.DecodeResponseErrorForWire(wire.ErrorKindProviderResponse, provider.Gemini, wireName, wire.ErrEmptyTextContent, requestID)
	}
	rawJSON := []byte(strings.TrimSpace(content))
	if !json.Valid(rawJSON) {
		return wire.EvalResult{}, wire.DecodeResponseErrorForWire(wire.ErrorKindProviderResponse, provider.Gemini, wireName, wire.ErrInvalidJSONPayload, requestID)
	}

	outModel := strings.TrimSpace(parsed.ModelVersion)
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

func buildEndpoint(endpointURL, baseURL, model string) string {
	if endpointURL != "" {
		return endpointURL
	}
	if baseURL == "" {
		baseURL = provider.GeminiBaseURL
	}
	return strings.TrimRight(baseURL, "/") + generateContentPathPrefix + url.PathEscape(model) + generateContentPathSuffix
}
