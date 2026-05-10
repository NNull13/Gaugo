package ollama

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/nnull13/gaugo"
	"github.com/nnull13/gaugo/internal/provider"
	"github.com/nnull13/gaugo/internal/provider/request"
	"github.com/nnull13/gaugo/internal/provider/validate"
	"github.com/nnull13/gaugo/internal/provider/wire"
	"github.com/nnull13/gaugo/internal/provider/wire/anthropic/messages"
	"github.com/nnull13/gaugo/internal/provider/wire/ollama/nativechat"
	"github.com/nnull13/gaugo/internal/provider/wire/openai/chat"
	"github.com/nnull13/gaugo/internal/provider/wire/openai/responses"
)

// Mode selects the Ollama wire protocol.
type Mode string

const (
	// ModeNative uses Ollama's native /api/chat endpoint.
	ModeNative Mode = "native"
	// ModeOpenAI uses Ollama's OpenAI-compatible endpoints.
	ModeOpenAI Mode = provider.OpenAI
	// ModeAnthropic uses Ollama's Anthropic-compatible endpoint.
	ModeAnthropic Mode = provider.Anthropic
)

// OpenAIEndpoint selects the OpenAI-compatible Ollama endpoint.
type OpenAIEndpoint string

const (
	// OpenAIEndpointChat uses /v1/chat/completions.
	OpenAIEndpointChat OpenAIEndpoint = "chat_completions"
	// OpenAIEndpointResponses uses /v1/responses.
	OpenAIEndpointResponses OpenAIEndpoint = provider.ResponsesWireName
)

// Config configures an Ollama judge.
type Config struct {
	APIKey               string
	Model                string
	BaseURL              string
	EndpointURL          string
	HTTPClient           *http.Client
	Mode                 Mode
	OpenAICompatEndpoint OpenAIEndpoint
	AnthropicAPIVersion  string
	AnthropicMaxTokens   int
	Retry                gaugo.RetryConfig
	MaxResponseBody      int64
}

// Judge evaluates metric prompts with Ollama.
type Judge struct {
	cfg  Config
	mode Mode
}

// New returns a configured Ollama judge.
func New(cfg Config) (*Judge, error) {
	mode, err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Judge{cfg: cfg, mode: mode}, nil
}

// EvaluateJSON evaluates one structured metric request.
func (j *Judge) EvaluateJSON(ctx context.Context, req gaugo.JudgeRequest) (gaugo.JudgeResponse, error) {
	wireReq := request.ToEval(req)
	switch j.mode {
	case ModeOpenAI:
		return j.evalOpenAI(ctx, wireReq)
	case ModeAnthropic:
		return j.evalAnthropic(ctx, wireReq)
	default:
		return j.evalNative(ctx, wireReq)
	}
}

func (j *Judge) evalNative(ctx context.Context, req wire.EvalRequest) (gaugo.JudgeResponse, error) {
	res, err := nativechat.EvaluateJSON(ctx, nativechat.Config{
		APIKey:          strings.TrimSpace(j.cfg.APIKey),
		Model:           j.cfg.Model,
		BaseURL:         defaultBaseURL(j.cfg.BaseURL),
		EndpointURL:     j.cfg.EndpointURL,
		HTTPClient:      j.cfg.HTTPClient,
		Retry:           request.ToRetry(j.cfg.Retry),
		MaxResponseBody: j.cfg.MaxResponseBody,
	}, req)
	if err != nil {
		return gaugo.JudgeResponse{}, err
	}
	return gaugo.JudgeResponse{
		RawJSON:  res.RawJSON,
		Provider: provider.Ollama,
		Model:    res.Model,
		Latency:  res.Latency,
	}, nil
}

func (j *Judge) evalOpenAI(ctx context.Context, req wire.EvalRequest) (gaugo.JudgeResponse, error) {
	apiKey := strings.TrimSpace(j.cfg.APIKey)
	if apiKey == "" {
		// Ollama's local OpenAI-compatible endpoint accepts but ignores auth.
		apiKey = provider.OllamaLocalAPIKey
	}
	endpoint := j.cfg.OpenAICompatEndpoint
	if endpoint == "" {
		endpoint = OpenAIEndpointChat
	}

	var (
		res wire.EvalResult
		err error
	)
	switch endpoint {
	case OpenAIEndpointResponses:
		res, err = responses.EvaluateJSON(ctx, responses.Config{
			APIKey:          apiKey,
			Model:           j.cfg.Model,
			BaseURL:         defaultBaseURL(j.cfg.BaseURL) + provider.OpenAIV1Path,
			EndpointURL:     j.cfg.EndpointURL,
			HTTPClient:      j.cfg.HTTPClient,
			Retry:           request.ToRetry(j.cfg.Retry),
			MaxResponseBody: j.cfg.MaxResponseBody,
		}, req)
	default:
		res, err = chat.EvaluateJSON(ctx, chat.Config{
			APIKey:          apiKey,
			Model:           j.cfg.Model,
			BaseURL:         defaultBaseURL(j.cfg.BaseURL) + provider.OpenAIV1Path,
			EndpointURL:     j.cfg.EndpointURL,
			HTTPClient:      j.cfg.HTTPClient,
			Retry:           request.ToRetry(j.cfg.Retry),
			MaxResponseBody: j.cfg.MaxResponseBody,
		}, req)
	}
	if err != nil {
		return gaugo.JudgeResponse{}, err
	}
	return gaugo.JudgeResponse{
		RawJSON:  res.RawJSON,
		Provider: provider.Ollama,
		Model:    res.Model,
		Latency:  res.Latency,
	}, nil
}

func (j *Judge) evalAnthropic(ctx context.Context, req wire.EvalRequest) (gaugo.JudgeResponse, error) {
	apiKey := strings.TrimSpace(j.cfg.APIKey)
	if apiKey == "" {
		apiKey = provider.OllamaLocalAPIKey
	}

	res, err := messages.EvaluateJSON(ctx, messages.Config{
		APIKey:          apiKey,
		Model:           j.cfg.Model,
		BaseURL:         defaultBaseURL(j.cfg.BaseURL),
		EndpointURL:     j.cfg.EndpointURL,
		APIVersion:      j.cfg.AnthropicAPIVersion,
		MaxTokens:       j.cfg.AnthropicMaxTokens,
		HTTPClient:      j.cfg.HTTPClient,
		Retry:           request.ToRetry(j.cfg.Retry),
		MaxResponseBody: j.cfg.MaxResponseBody,
	}, req)
	if err != nil {
		return gaugo.JudgeResponse{}, err
	}
	return gaugo.JudgeResponse{
		RawJSON:  res.RawJSON,
		Provider: provider.Ollama,
		Model:    res.Model,
		Latency:  res.Latency,
	}, nil
}

func defaultBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return provider.OllamaLocalBaseURL
	}
	return baseURL
}

func normalizeMode(m Mode) (Mode, error) {
	switch strings.ToLower(strings.TrimSpace(string(m))) {
	case "":
		return ModeNative, nil
	case string(ModeOpenAI):
		return ModeOpenAI, nil
	case string(ModeAnthropic):
		return ModeAnthropic, nil
	case string(ModeNative):
		return ModeNative, nil
	default:
		return "", fmt.Errorf(provider.ConfigUnsupportedProviderWireMode, m)
	}
}

// Validate reports invalid Ollama configuration.
func (cfg Config) Validate() error {
	_, err := validateConfig(cfg)
	return err
}

func validateConfig(cfg Config) (Mode, error) {
	mode, err := normalizeMode(cfg.Mode)
	if err != nil {
		return "", provider.ConfigWrapError(provider.Ollama, err)
	}

	if err := validate.BaseURL(cfg.BaseURL); err != nil {
		return "", provider.ConfigWrapError(provider.Ollama, err)
	}
	if err := validate.BaseURL(cfg.EndpointURL); err != nil {
		return "", provider.ConfigFieldWrapError(provider.Ollama, provider.FieldEndpointURL, err)
	}

	if mode != ModeOpenAI && strings.TrimSpace(string(cfg.OpenAICompatEndpoint)) != "" {
		return "", provider.ConfigError(provider.Ollama, provider.ConfigOpenAIEndpointRequiresMode)
	}

	if mode == ModeOpenAI {
		switch cfg.OpenAICompatEndpoint {
		case "", OpenAIEndpointChat, OpenAIEndpointResponses:
		default:
			return "", provider.ConfigErrorf(provider.Ollama, provider.ConfigUnsupportedOpenAIEndpoint, cfg.OpenAICompatEndpoint)
		}
	}
	if err := cfg.Retry.Validate(); err != nil {
		return "", provider.ConfigWrapError(provider.Ollama, err)
	}
	if cfg.MaxResponseBody < 0 {
		return "", provider.ConfigError(provider.Ollama, provider.ConfigMaxResponseBodyNonNegative)
	}

	return mode, nil
}
