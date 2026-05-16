package custom

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/nnull13/gaugo"
	"github.com/nnull13/gaugo/internal/failure"
	"github.com/nnull13/gaugo/internal/provider"
	"github.com/nnull13/gaugo/internal/provider/request"
	"github.com/nnull13/gaugo/internal/provider/validate"
	"github.com/nnull13/gaugo/internal/provider/wire"
	"github.com/nnull13/gaugo/internal/provider/wire/anthropic/messages"
	"github.com/nnull13/gaugo/internal/provider/wire/ollama/nativechat"
	"github.com/nnull13/gaugo/internal/provider/wire/openai/chat"
	"github.com/nnull13/gaugo/internal/provider/wire/openai/responses"
	"github.com/nnull13/gaugo/internal/ratelimit"
)

// Mode selects the custom model service wire protocol.
type Mode string

const (
	// ModeNative uses Ollama's native /api/chat endpoint.
	ModeNative Mode = "native"
	// ModeOpenAI uses OpenAI-compatible endpoints.
	ModeOpenAI Mode = provider.OpenAI
	// ModeAnthropic uses an Anthropic-compatible endpoint.
	ModeAnthropic Mode = provider.Anthropic
)

// OpenAIEndpoint selects the OpenAI-compatible endpoint.
type OpenAIEndpoint string

const (
	// OpenAIEndpointChat uses /v1/chat/completions.
	OpenAIEndpointChat OpenAIEndpoint = "chat_completions"
	// OpenAIEndpointResponses uses /v1/responses.
	OpenAIEndpointResponses OpenAIEndpoint = provider.ResponsesWireName
)

// Config configures a custom model service judge.
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
	RateLimit            gaugo.RateLimitConfig
	MaxResponseBody      int64
}

// Judge evaluates metric prompts with a custom model service.
type Judge struct {
	cfg     Config
	mode    Mode
	limiter *ratelimit.Limiter
}

// New returns a configured custom judge.
func New(cfg Config) (*Judge, error) {
	mode, err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}
	j := &Judge{cfg: cfg, mode: mode}
	if cfg.RateLimit.RequestsPerMinute > 0 {
		j.limiter = ratelimit.New(cfg.RateLimit.RequestsPerMinute, cfg.RateLimit.Burst)
	}
	return j, nil
}

// EvaluateJSON evaluates one structured metric request.
func (j *Judge) EvaluateJSON(ctx context.Context, req gaugo.JudgeRequest) (gaugo.JudgeResponse, error) {
	if j.limiter != nil {
		if err := j.limiter.Wait(ctx); err != nil {
			return gaugo.JudgeResponse{}, err
		}
	}
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
		Model:           customModel(j.cfg.Model),
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
		Provider:  provider.Custom,
		Model:     res.Model,
		RequestID: res.RequestID,
		Latency:   res.Latency,
	}, nil
}

func (j *Judge) evalOpenAI(ctx context.Context, req wire.EvalRequest) (gaugo.JudgeResponse, error) {
	apiKey := strings.TrimSpace(j.cfg.APIKey)
	if apiKey == "" {
		apiKey = provider.CustomAPIKey
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
			Provider:        provider.Custom,
			APIKey:          apiKey,
			Model:           customModel(j.cfg.Model),
			BaseURL:         openAICompatBaseURL(j.cfg.BaseURL),
			EndpointURL:     j.cfg.EndpointURL,
			HTTPClient:      j.cfg.HTTPClient,
			Retry:           request.ToRetry(j.cfg.Retry),
			MaxResponseBody: j.cfg.MaxResponseBody,
		}, req)
	default:
		res, err = chat.EvaluateJSON(ctx, chat.Config{
			Provider:        provider.Custom,
			APIKey:          apiKey,
			Model:           customModel(j.cfg.Model),
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
		Provider:  provider.Custom,
		Model:     res.Model,
		RequestID: res.RequestID,
		Latency:   res.Latency,
	}, nil
}

func (j *Judge) evalAnthropic(ctx context.Context, req wire.EvalRequest) (gaugo.JudgeResponse, error) {
	apiKey := strings.TrimSpace(j.cfg.APIKey)
	if apiKey == "" {
		apiKey = provider.CustomAPIKey
	}

	res, err := messages.EvaluateJSON(ctx, messages.Config{
		Provider:        provider.Custom,
		APIKey:          apiKey,
		Model:           customModel(j.cfg.Model),
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
		Provider:  provider.Custom,
		Model:     res.Model,
		RequestID: res.RequestID,
		Latency:   res.Latency,
	}, nil
}

func customModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return provider.CustomDefaultModel
	}
	return model
}

func defaultBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return provider.CustomBaseURL
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
		return "", failure.Validation(
			failure.CodeProviderConfigInvalid,
			"custom.normalize_mode",
			"mode",
			fmt.Sprintf(provider.ConfigUnsupportedProviderWireMode, m),
			nil,
		)
	}
}

// Validate reports invalid custom provider configuration.
func (cfg Config) Validate() error {
	_, err := validateConfig(cfg)
	return err
}

func validateConfig(cfg Config) (Mode, error) {
	mode, err := normalizeMode(cfg.Mode)
	if err != nil {
		return "", provider.ConfigWrapError(provider.Custom, err)
	}

	err = validate.BaseURL(cfg.BaseURL)
	if err != nil {
		return "", provider.ConfigWrapError(provider.Custom, err)
	}
	err = validate.BaseURL(cfg.EndpointURL)
	if err != nil {
		return "", provider.ConfigFieldWrapError(provider.Custom, provider.FieldEndpointURL, err)
	}

	if mode != ModeOpenAI && strings.TrimSpace(string(cfg.OpenAICompatEndpoint)) != "" {
		return "", provider.ConfigError(provider.Custom, provider.ConfigOpenAIEndpointRequiresMode)
	}

	if mode == ModeOpenAI {
		switch cfg.OpenAICompatEndpoint {
		case "", OpenAIEndpointChat, OpenAIEndpointResponses:
		default:
			return "", provider.ConfigErrorf(provider.Custom, provider.ConfigUnsupportedOpenAIEndpoint, cfg.OpenAICompatEndpoint)
		}
	}
	err = cfg.Retry.Validate()
	if err != nil {
		return "", provider.ConfigWrapError(provider.Custom, err)
	}
	err = cfg.RateLimit.Validate()
	if err != nil {
		return "", provider.ConfigWrapError(provider.Custom, err)
	}
	if cfg.MaxResponseBody < 0 {
		return "", provider.ConfigError(provider.Custom, provider.ConfigMaxResponseBodyNonNegative)
	}

	return mode, nil
}
