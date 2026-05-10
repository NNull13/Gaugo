package nativechat

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

func EvaluateJSON(ctx context.Context, cfg Config, req wire.EvalRequest) (wire.EvalResult, error) {
	var (
		err    error
		schema any
		body   []byte
		resp   wire.HTTPResponse
	)

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = provider.OllamaDefaultModel
	}
	endpoint := endpointURL(cfg.EndpointURL, cfg.BaseURL, provider.OllamaLocalBaseURL, provider.OllamaNativeChatPath)

	schema, err = wire.DecodeSchema(req.Schema)
	if err != nil {
		return wire.EvalResult{}, err
	}

	body, err = json.Marshal(requestBody{
		Model:  model,
		Stream: false,
		Messages: []message{
			{Role: "system", Content: req.Instructions},
			{Role: "user", Content: req.UserPrompt},
		},
		Format: schema,
		Options: optionsBody{
			Temperature: 0,
		},
	})
	if err != nil {
		return wire.EvalResult{}, fmt.Errorf("marshal ollama request: %w", err)
	}

	headers := map[string]string{
		wire.HeaderContentType: wire.ContentTypeJSON,
	}
	if apiKey := strings.TrimSpace(cfg.APIKey); apiKey != "" {
		headers[wire.HeaderAuthorization] = wire.AuthBearerPrefix + apiKey
	}

	resp, err = wire.PostJSONWithOptions(ctx, wire.NewHTTPClient(cfg.HTTPClient), endpoint, headers, body, wire.HTTPOptions{
		Retry:        cfg.Retry,
		MaxBodyBytes: cfg.MaxResponseBody,
	})
	if err != nil {
		return wire.EvalResult{}, fmt.Errorf("ollama judge request failed: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return wire.EvalResult{}, wire.StatusError(provider.Ollama, resp)
	}

	var parsed struct {
		Model   string `json:"model"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	err = json.Unmarshal(resp.Body, &parsed)
	if err != nil {
		return wire.EvalResult{}, fmt.Errorf("decode ollama response: %w", err)
	}

	content := wire.StripCodeFence(parsed.Message.Content)
	if strings.TrimSpace(content) == "" {
		return wire.EvalResult{}, fmt.Errorf("decode ollama response: empty message content")
	}
	rawJSON := []byte(strings.TrimSpace(content))
	if !json.Valid(rawJSON) {
		return wire.EvalResult{}, fmt.Errorf("decode ollama response: invalid json payload")
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
