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
	"github.com/nnull13/gaugo/internal/ratelimit"
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
	RateLimit       gaugo.RateLimitConfig
	MaxResponseBody int64
}

// Judge evaluates metric prompts with Anthropic.
type Judge struct {
	cfg     Config
	limiter *ratelimit.Limiter
}

// New returns a configured Anthropic judge.
func New(cfg Config) (*Judge, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	j := &Judge{cfg: cfg}
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
	res, err := messages.EvaluateJSON(ctx, messages.Config{
		Provider:        provider.Anthropic,
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
		RawJSON:   res.RawJSON,
		Provider:  provider.Anthropic,
		Model:     res.Model,
		RequestID: res.RequestID,
		Latency:   res.Latency,
	}, nil
}

// Validate reports invalid Anthropic configuration.
func (cfg Config) Validate() error {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return provider.ConfigError(provider.Anthropic, provider.ConfigAPIKeyRequired)
	}
	err := validate.CloudURL(cfg.BaseURL, cfg.AllowUnsafeURL, provider.AnthropicHost)
	if err != nil {
		return provider.ConfigWrapError(provider.Anthropic, err)
	}
	err = validate.CloudURL(cfg.EndpointURL, cfg.AllowUnsafeURL, provider.AnthropicHost)
	if err != nil {
		return provider.ConfigFieldWrapError(provider.Anthropic, provider.FieldEndpointURL, err)
	}
	err = cfg.Retry.Validate()
	if err != nil {
		return provider.ConfigWrapError(provider.Anthropic, err)
	}
	err = cfg.RateLimit.Validate()
	if err != nil {
		return provider.ConfigWrapError(provider.Anthropic, err)
	}
	if cfg.MaxResponseBody < 0 {
		return provider.ConfigError(provider.Anthropic, provider.ConfigMaxResponseBodyNonNegative)
	}
	return nil
}
