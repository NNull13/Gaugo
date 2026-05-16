package nvidia

import (
	"context"
	"net/http"
	"strings"

	"github.com/nnull13/gaugo"
	"github.com/nnull13/gaugo/internal/provider"
	"github.com/nnull13/gaugo/internal/provider/request"
	"github.com/nnull13/gaugo/internal/provider/validate"
	"github.com/nnull13/gaugo/internal/provider/wire/openai/chat"
	"github.com/nnull13/gaugo/internal/ratelimit"
)

// Config configures an NVIDIA NIM judge.
type Config struct {
	APIKey          string
	Model           string
	BaseURL         string
	EndpointURL     string
	AllowUnsafeURL  bool
	HTTPClient      *http.Client
	Retry           gaugo.RetryConfig
	RateLimit       gaugo.RateLimitConfig
	MaxResponseBody int64
}

// Judge evaluates metric prompts with NVIDIA NIM.
type Judge struct {
	cfg     Config
	limiter *ratelimit.Limiter
}

// New returns a configured NVIDIA NIM judge.
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
	res, err := chat.EvaluateJSON(ctx, chat.Config{
		Provider:        provider.NVIDIA,
		APIKey:          j.cfg.APIKey,
		Model:           defaultModel(j.cfg.Model),
		BaseURL:         defaultBaseURL(j.cfg.BaseURL),
		EndpointURL:     j.cfg.EndpointURL,
		HTTPClient:      j.cfg.HTTPClient,
		Retry:           request.ToRetry(j.cfg.Retry),
		MaxResponseBody: j.cfg.MaxResponseBody,
	}, wireReq)
	if err != nil {
		return gaugo.JudgeResponse{}, err
	}
	return gaugo.JudgeResponse{
		RawJSON:   res.RawJSON,
		Provider:  provider.NVIDIA,
		Model:     res.Model,
		RequestID: res.RequestID,
		Latency:   res.Latency,
	}, nil
}

func defaultModel(m string) string {
	m = strings.TrimSpace(m)
	if m == "" {
		return provider.NVIDIADefaultModel
	}
	return m
}

func defaultBaseURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return provider.NVIDIABaseURL
	}
	return baseURL
}

// Validate reports invalid NVIDIA NIM configuration.
func (cfg Config) Validate() error {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return provider.ConfigError(provider.NVIDIA, provider.ConfigAPIKeyRequired)
	}
	err := validate.CloudURL(cfg.BaseURL, cfg.AllowUnsafeURL, provider.NVIDIAHost)
	if err != nil {
		return provider.ConfigWrapError(provider.NVIDIA, err)
	}
	err = validate.CloudURL(cfg.EndpointURL, cfg.AllowUnsafeURL, provider.NVIDIAHost)
	if err != nil {
		return provider.ConfigFieldWrapError(provider.NVIDIA, provider.FieldEndpointURL, err)
	}
	err = cfg.Retry.Validate()
	if err != nil {
		return provider.ConfigWrapError(provider.NVIDIA, err)
	}
	err = cfg.RateLimit.Validate()
	if err != nil {
		return provider.ConfigWrapError(provider.NVIDIA, err)
	}
	if cfg.MaxResponseBody < 0 {
		return provider.ConfigError(provider.NVIDIA, provider.ConfigMaxResponseBodyNonNegative)
	}
	return nil
}
