package gemini

import (
	"context"
	"net/http"
	"strings"

	"github.com/nnull13/gaugo"
	"github.com/nnull13/gaugo/internal/provider"
	"github.com/nnull13/gaugo/internal/provider/request"
	"github.com/nnull13/gaugo/internal/provider/validate"
	"github.com/nnull13/gaugo/internal/provider/wire/gemini/generatecontent"
)

// Config configures a Gemini GenerateContent judge.
type Config struct {
	APIKey          string
	Model           string
	BaseURL         string
	EndpointURL     string
	AllowUnsafeURL  bool
	HTTPClient      *http.Client
	Retry           gaugo.RetryConfig
	MaxResponseBody int64
}

// Judge evaluates metric prompts with Gemini.
type Judge struct {
	cfg Config
}

// New returns a configured Gemini judge.
func New(cfg Config) (*Judge, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Judge{cfg: cfg}, nil
}

// EvaluateJSON evaluates one structured metric request.
func (j *Judge) EvaluateJSON(ctx context.Context, req gaugo.JudgeRequest) (gaugo.JudgeResponse, error) {
	wireReq := request.ToEval(req)
	res, err := generatecontent.EvaluateJSON(ctx, generatecontent.Config{
		APIKey:          j.cfg.APIKey,
		Model:           j.cfg.Model,
		BaseURL:         j.cfg.BaseURL,
		EndpointURL:     j.cfg.EndpointURL,
		HTTPClient:      j.cfg.HTTPClient,
		Retry:           request.ToRetry(j.cfg.Retry),
		MaxResponseBody: j.cfg.MaxResponseBody,
	}, wireReq)
	if err != nil {
		return gaugo.JudgeResponse{}, err
	}
	return gaugo.JudgeResponse{
		RawJSON:  res.RawJSON,
		Provider: provider.Gemini,
		Model:    res.Model,
		Latency:  res.Latency,
	}, nil
}

// Validate reports invalid Gemini configuration.
func (cfg Config) Validate() error {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return provider.ConfigError(provider.Gemini, provider.ConfigAPIKeyRequired)
	}
	if err := validate.CloudURL(cfg.BaseURL, cfg.AllowUnsafeURL, provider.GeminiHost); err != nil {
		return provider.ConfigWrapError(provider.Gemini, err)
	}
	if err := validate.CloudURL(cfg.EndpointURL, cfg.AllowUnsafeURL, provider.GeminiHost); err != nil {
		return provider.ConfigFieldWrapError(provider.Gemini, provider.FieldEndpointURL, err)
	}
	if err := cfg.Retry.Validate(); err != nil {
		return provider.ConfigWrapError(provider.Gemini, err)
	}
	if cfg.MaxResponseBody < 0 {
		return provider.ConfigError(provider.Gemini, provider.ConfigMaxResponseBodyNonNegative)
	}
	return nil
}
