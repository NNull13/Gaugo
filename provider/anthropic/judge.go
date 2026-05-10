package anthropic

import (
	"context"
	"net/http"
	"strings"

	"github.com/nnull13/gaugo"
	"github.com/nnull13/gaugo/internal/provider"
	"github.com/nnull13/gaugo/internal/provider/request"
	"github.com/nnull13/gaugo/internal/provider/validate"
	"github.com/nnull13/gaugo/internal/provider/wire/anthropic/messages"
)

// Config configures an Anthropic Messages judge.
type Config struct {
	APIKey          string
	Model           string
	BaseURL         string
	EndpointURL     string
	AllowUnsafeURL  bool
	APIVersion      string
	MaxTokens       int
	HTTPClient      *http.Client
	Retry           gaugo.RetryConfig
	MaxResponseBody int64
}

// Judge evaluates metric prompts with Anthropic.
type Judge struct {
	cfg Config
}

// New returns a configured Anthropic judge.
func New(cfg Config) (*Judge, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Judge{cfg: cfg}, nil
}

// EvaluateJSON evaluates one structured metric request.
func (j *Judge) EvaluateJSON(ctx context.Context, req gaugo.JudgeRequest) (gaugo.JudgeResponse, error) {
	wireReq := request.ToEval(req)
	res, err := messages.EvaluateJSON(ctx, messages.Config{
		APIKey:          j.cfg.APIKey,
		Model:           j.cfg.Model,
		BaseURL:         j.cfg.BaseURL,
		EndpointURL:     j.cfg.EndpointURL,
		APIVersion:      j.cfg.APIVersion,
		MaxTokens:       j.cfg.MaxTokens,
		HTTPClient:      j.cfg.HTTPClient,
		Retry:           request.ToRetry(j.cfg.Retry),
		MaxResponseBody: j.cfg.MaxResponseBody,
	}, wireReq)
	if err != nil {
		return gaugo.JudgeResponse{}, err
	}
	return gaugo.JudgeResponse{
		RawJSON:  res.RawJSON,
		Provider: provider.Anthropic,
		Model:    res.Model,
		Latency:  res.Latency,
	}, nil
}

// Validate reports invalid Anthropic configuration.
func (cfg Config) Validate() error {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return provider.ConfigError(provider.Anthropic, provider.ConfigAPIKeyRequired)
	}
	if err := validate.CloudURL(cfg.BaseURL, cfg.AllowUnsafeURL, provider.AnthropicHost); err != nil {
		return provider.ConfigWrapError(provider.Anthropic, err)
	}
	if err := validate.CloudURL(cfg.EndpointURL, cfg.AllowUnsafeURL, provider.AnthropicHost); err != nil {
		return provider.ConfigFieldWrapError(provider.Anthropic, provider.FieldEndpointURL, err)
	}
	if err := cfg.Retry.Validate(); err != nil {
		return provider.ConfigWrapError(provider.Anthropic, err)
	}
	if cfg.MaxResponseBody < 0 {
		return provider.ConfigError(provider.Anthropic, provider.ConfigMaxResponseBodyNonNegative)
	}
	return nil
}
