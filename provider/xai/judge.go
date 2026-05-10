package xai

import (
	"context"
	"net/http"
	"strings"

	"github.com/nnull13/gaugo"
	"github.com/nnull13/gaugo/internal/provider"
	"github.com/nnull13/gaugo/internal/provider/request"
	"github.com/nnull13/gaugo/internal/provider/validate"
	"github.com/nnull13/gaugo/internal/provider/wire"
	"github.com/nnull13/gaugo/internal/provider/wire/openai/chat"
	"github.com/nnull13/gaugo/internal/provider/wire/openai/responses"
)

// Config configures an xAI judge.
type Config struct {
	APIKey             string
	Model              string
	BaseURL            string
	EndpointURL        string
	AllowUnsafeURL     bool
	HTTPClient         *http.Client
	UseChatCompletions bool
	Retry              gaugo.RetryConfig
	MaxResponseBody    int64
}

// Judge evaluates metric prompts with xAI.
type Judge struct {
	cfg Config
}

// New returns a configured xAI judge.
func New(cfg Config) (*Judge, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Judge{cfg: cfg}, nil
}

// EvaluateJSON evaluates one structured metric request.
func (j *Judge) EvaluateJSON(ctx context.Context, req gaugo.JudgeRequest) (gaugo.JudgeResponse, error) {
	wireReq := request.ToEval(req)
	if j.cfg.UseChatCompletions {
		return j.evalChat(ctx, wireReq)
	}
	return j.evalResponses(ctx, wireReq)
}

func (j *Judge) evalResponses(ctx context.Context, req wire.EvalRequest) (gaugo.JudgeResponse, error) {
	res, err := responses.EvaluateJSON(ctx, responses.Config{
		APIKey:          j.cfg.APIKey,
		Model:           defaultModel(j.cfg.Model),
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
		Provider: provider.XAI,
		Model:    res.Model,
		Latency:  res.Latency,
	}, nil
}

func (j *Judge) evalChat(ctx context.Context, req wire.EvalRequest) (gaugo.JudgeResponse, error) {
	res, err := chat.EvaluateJSON(ctx, chat.Config{
		APIKey:          j.cfg.APIKey,
		Model:           defaultModel(j.cfg.Model),
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
		Provider: provider.XAI,
		Model:    res.Model,
		Latency:  res.Latency,
	}, nil
}

func defaultModel(m string) string {
	m = strings.TrimSpace(m)
	if m == "" {
		return provider.XAIDefaultModel
	}
	return m
}

func defaultBaseURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return provider.XAIBaseURL
	}
	return baseURL
}

// Validate reports invalid xAI configuration.
func (cfg Config) Validate() error {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return provider.ConfigError(provider.XAI, provider.ConfigAPIKeyRequired)
	}
	if err := validate.CloudURL(cfg.BaseURL, cfg.AllowUnsafeURL, provider.XAIHost); err != nil {
		return provider.ConfigWrapError(provider.XAI, err)
	}
	if err := validate.CloudURL(cfg.EndpointURL, cfg.AllowUnsafeURL, provider.XAIHost); err != nil {
		return provider.ConfigFieldWrapError(provider.XAI, provider.FieldEndpointURL, err)
	}
	if err := cfg.Retry.Validate(); err != nil {
		return provider.ConfigWrapError(provider.XAI, err)
	}
	if cfg.MaxResponseBody < 0 {
		return provider.ConfigError(provider.XAI, provider.ConfigMaxResponseBodyNonNegative)
	}
	return nil
}
