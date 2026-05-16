package nvidia

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nnull13/gaugo"
)

func TestEvaluateJSONInvalidConfig(t *testing.T) {
	t.Parallel()

	_, err := New(Config{})
	if err == nil || !strings.Contains(err.Error(), "invalid nvidia config") {
		t.Fatalf("expected invalid config error, got: %v", err)
	}
	if !errors.Is(err, gaugo.ErrConfig) {
		t.Fatalf("expected config sentinel, got: %v", err)
	}
	info := gaugo.ClassifyError(err)
	if info.Kind != gaugo.ErrorKindConfig || string(info.Code) != "provider_api_key_required" || info.Provider != "nvidia" {
		t.Fatalf("unexpected error info: %+v", info)
	}
}

func TestEvaluateJSONInvalidBaseURL(t *testing.T) {
	t.Parallel()

	_, err := New(Config{APIKey: "nvapi-test", BaseURL: "://bad"})
	if err == nil || !strings.Contains(err.Error(), "invalid nvidia config") {
		t.Fatalf("expected invalid base url error, got: %v", err)
	}
	info := gaugo.ClassifyError(err)
	if info.Kind != gaugo.ErrorKindConfig || info.Provider != "nvidia" {
		t.Fatalf("unexpected error info: %+v", info)
	}
}

func TestEvaluateJSONInvalidEndpointURLRedactsUserInfo(t *testing.T) {
	t.Parallel()

	_, err := New(Config{
		APIKey:      "nvapi-test",
		EndpointURL: "https://user:secret@integrate.api.nvidia.com/v1/chat/completions",
	})
	if err == nil {
		t.Fatalf("expected invalid endpoint URL")
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "user:secret") {
		t.Fatalf("endpoint URL error leaked user info: %v", err)
	}
	info := gaugo.ClassifyError(err)
	if info.Kind != gaugo.ErrorKindConfig || info.Provider != "nvidia" || info.Field != "endpoint URL" {
		t.Fatalf("unexpected error info: %+v", info)
	}
}

func TestValidateURLPolicy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "strict allows official https",
			cfg: Config{
				APIKey:  "nvapi-test",
				BaseURL: "https://integrate.api.nvidia.com/v1",
			},
		},
		{
			name: "strict denies unofficial host",
			cfg: Config{
				APIKey:  "nvapi-test",
				BaseURL: "https://example.com/v1",
			},
			wantErr: true,
		},
		{
			name: "strict denies http",
			cfg: Config{
				APIKey:  "nvapi-test",
				BaseURL: "http://integrate.api.nvidia.com/v1",
			},
			wantErr: true,
		},
		{
			name: "strict denies userinfo",
			cfg: Config{
				APIKey:  "nvapi-test",
				BaseURL: "https://user:pass@integrate.api.nvidia.com/v1",
			},
			wantErr: true,
		},
		{
			name: "unsafe override allows custom http endpoint",
			cfg: Config{
				APIKey:         "nvapi-test",
				EndpointURL:    "http://127.0.0.1:8080/v1/chat/completions",
				AllowUnsafeURL: true,
			},
		},
		{
			name: "unsafe still requires http or https",
			cfg: Config{
				APIKey:         "nvapi-test",
				EndpointURL:    "ftp://127.0.0.1:8080/v1/chat/completions",
				AllowUnsafeURL: true,
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := New(tc.cfg)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEvaluateJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"model":"meta/llama-3.3-70b-instruct","choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
	}))
	defer srv.Close()

	j, err := New(Config{
		APIKey:         "nvapi-test",
		EndpointURL:    srv.URL,
		Model:          "meta/llama-3.3-70b-instruct",
		AllowUnsafeURL: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	res, err := j.EvaluateJSON(context.Background(), sampleReq())
	if err != nil {
		t.Fatalf("EvaluateJSON error: %v", err)
	}
	if res.Provider != "nvidia" {
		t.Fatalf("provider got=%q", res.Provider)
	}
}

func sampleReq() gaugo.JudgeRequest {
	return gaugo.JudgeRequest{
		Metric:       "Faithfulness",
		Question:     "Q",
		Answer:       "A",
		Instructions: "Return JSON",
		Schema:       json.RawMessage(`{"type":"object"}`),
	}
}
