package local

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

// Mode selects the local model service wire protocol.
type Mode string

const (
	// ModeNative uses Ollama's native /api/chat endpoint.
	ModeNative Mode = "native"
	// ModeOpenAI uses OpenAI-compatible local endpoints.
	ModeOpenAI Mode = provider.OpenAI
	// ModeAnthropic uses an Anthropic-compatible local endpoint.
	ModeAnthropic Mode = provider.Anthropic
)

// OpenAIEndpoint selects the OpenAI-compatible local endpoint.
type OpenAIEndpoint string

const (
	// OpenAIEndpointChat uses /v1/chat/completions.
	OpenAIEndpointChat OpenAIEndpoint = "chat_completions"
	// OpenAIEndpointResponses uses /v1/responses.
	OpenAIEndpointResponses OpenAIEndpoint = provider.ResponsesWireName
)

// Config configures a local model service judge.
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

// Judge evaluates metric prompts with a local model service.
type Judge struct {
	cfg  Config
	mode Mode
}

// New returns a configured local judge.
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
		Model:           localModel(j.cfg.Model),
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
		RawJSON:   res.RawJSON,
		Provider:  provider.Local,
		Model:     res.Model,
		RequestID: res.RequestID,
		Latency:   res.Latency,
	}, nil
}

func (j *Judge) evalOpenAI(ctx context.Context, req wire.EvalRequest) (gaugo.JudgeResponse, error) {
	apiKey := strings.TrimSpace(j.cfg.APIKey)
	if apiKey == "" {
		// Local OpenAI-compatible endpoints commonly accept but ignore auth.
		apiKey = provider.LocalAPIKey
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
			Provider:        provider.Local,
			APIKey:          apiKey,
			Model:           localModel(j.cfg.Model),
			BaseURL:         openAICompatBaseURL(j.cfg.BaseURL),
			EndpointURL:     j.cfg.EndpointURL,
			HTTPClient:      j.cfg.HTTPClient,
			Retry:           request.ToRetry(j.cfg.Retry),
			MaxResponseBody: j.cfg.MaxResponseBody,
		}, req)
	default:
		res, err = chat.EvaluateJSON(ctx, chat.Config{
			Provider:        provider.Local,
			APIKey:          apiKey,
			Model:           localModel(j.cfg.Model),
			BaseURL:         openAICompatBaseURL(j.cfg.BaseURL),
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
		RawJSON:   res.RawJSON,
		Provider:  provider.Local,
		Model:     res.Model,
		RequestID: res.RequestID,
		Latency:   res.Latency,
	}, nil
}

func (j *Judge) evalAnthropic(ctx context.Context, req wire.EvalRequest) (gaugo.JudgeResponse, error) {
	apiKey := strings.TrimSpace(j.cfg.APIKey)
	if apiKey == "" {
		apiKey = provider.LocalAPIKey
	}

	res, err := messages.EvaluateJSON(ctx, messages.Config{
		Provider:        provider.Local,
		APIKey:          apiKey,
		Model:           localModel(j.cfg.Model),
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
		RawJSON:   res.RawJSON,
		Provider:  provider.Local,
		Model:     res.Model,
		RequestID: res.RequestID,
		Latency:   res.Latency,
	}, nil
}

func localModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return provider.LocalDefaultModel
	}
	return model
}

func defaultBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return provider.LocalBaseURL
	}
	return baseURL
}

func openAICompatBaseURL(baseURL string) string {
	base := defaultBaseURL(baseURL)
	if strings.HasSuffix(base, provider.OpenAIV1Path) {
		return base
	}
	return base + provider.OpenAIV1Path
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

// Validate reports invalid local provider configuration.
func (cfg Config) Validate() error {
	_, err := validateConfig(cfg)
	return err
}

func validateConfig(cfg Config) (Mode, error) {
	mode, err := normalizeMode(cfg.Mode)
	if err != nil {
		return "", provider.ConfigWrapError(provider.Local, err)
	}

	err = validate.BaseURL(cfg.BaseURL)
	if err != nil {
		return "", provider.ConfigWrapError(provider.Local, err)
	}
	err = validate.BaseURL(cfg.EndpointURL)
	if err != nil {
		return "", provider.ConfigFieldWrapError(provider.Local, provider.FieldEndpointURL, err)
	}

	if mode != ModeOpenAI && strings.TrimSpace(string(cfg.OpenAICompatEndpoint)) != "" {
		return "", provider.ConfigError(provider.Local, provider.ConfigOpenAIEndpointRequiresMode)
	}

	if mode == ModeOpenAI {
		switch cfg.OpenAICompatEndpoint {
		case "", OpenAIEndpointChat, OpenAIEndpointResponses:
		default:
			return "", provider.ConfigErrorf(provider.Local, provider.ConfigUnsupportedOpenAIEndpoint, cfg.OpenAICompatEndpoint)
		}
	}
	err = cfg.Retry.Validate()
	if err != nil {
		return "", provider.ConfigWrapError(provider.Local, err)
	}
	if cfg.MaxResponseBody < 0 {
		return "", provider.ConfigError(provider.Local, provider.ConfigMaxResponseBodyNonNegative)
	}

	return mode, nil
}
