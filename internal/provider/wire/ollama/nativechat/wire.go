package nativechat

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
	HTTPClient      *http.Client
	Retry           wire.RetryConfig
	MaxResponseBody int64
}

type requestBody struct {
	Model    string      `json:"model"`
	Stream   bool        `json:"stream"`
	Messages []message   `json:"messages"`
	Format   any         `json:"format"`
	Options  optionsBody `json:"options"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type optionsBody struct {
	Temperature int `json:"temperature"`
}

const wireName = "ollama"

func EvaluateJSON(ctx context.Context, cfg Config, req wire.EvalRequest) (wire.EvalResult, error) {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = provider.LocalDefaultModel
	}
	endpoint := wire.EndpointURL(cfg.EndpointURL, cfg.BaseURL, provider.LocalBaseURL, provider.OllamaNativeChatPath)

	schema, err := wire.DecodeSchema(req.Schema)
	if err != nil {
		return wire.EvalResult{}, err
	}

	var body []byte
	body, err = json.Marshal(requestBody{
		Model:  model,
		Stream: false,
		Messages: []message{
			{Role: wire.RoleSystem, Content: req.Instructions},
			{Role: wire.RoleUser, Content: req.UserPrompt},
		},
		Format: schema,
		Options: optionsBody{
			Temperature: 0,
		},
	})
	if err != nil {
		return wire.EvalResult{}, wire.MarshalRequestError(wireName, err)
	}

	headers := map[string]string{
		wire.HeaderContentType: wire.ContentTypeJSON,
	}
	if apiKey := strings.TrimSpace(cfg.APIKey); apiKey != "" {
		headers[wire.HeaderAuthorization] = wire.AuthBearerPrefix + apiKey
	}

	var resp wire.HTTPResponse
	resp, err = wire.PostJSONWithOptions(ctx, wire.NewHTTPClient(cfg.HTTPClient), endpoint, headers, body, wire.HTTPOptions{
		Retry:        cfg.Retry,
		MaxBodyBytes: cfg.MaxResponseBody,
	})
	if err != nil {
		return wire.EvalResult{}, wire.JudgeRequestError(wireName, err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return wire.EvalResult{}, wire.StatusError(provider.Local, resp)
	}

	var parsed struct {
		Model   string `json:"model"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	err = json.Unmarshal(resp.Body, &parsed)
	if err != nil {
		return wire.EvalResult{}, wire.DecodeResponseWrapError(wireName, err)
	}

	content := wire.StripCodeFence(parsed.Message.Content)
	if strings.TrimSpace(content) == "" {
		return wire.EvalResult{}, wire.DecodeResponseError(wireName, "empty message content")
	}
	rawJSON := []byte(strings.TrimSpace(content))
	if !json.Valid(rawJSON) {
		return wire.EvalResult{}, wire.DecodeResponseError(wireName, wire.ErrInvalidJSONPayload)
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
