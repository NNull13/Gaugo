package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nnull13/gaugo"
)

func TestEvaluateJSONInvalidConfig(t *testing.T) {
	t.Parallel()

	_, err := New(Config{})
	if err == nil || !strings.Contains(err.Error(), "invalid gemini config") {
		t.Fatalf("expected invalid config error, got: %v", err)
	}
}

func TestEvaluateJSONInvalidBaseURL(t *testing.T) {
	t.Parallel()

	_, err := New(Config{APIKey: "k", BaseURL: "://bad"})
	if err == nil || !strings.Contains(err.Error(), "invalid gemini config") {
		t.Fatalf("expected invalid base url error, got: %v", err)
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
				APIKey:  "k",
				BaseURL: "https://generativelanguage.googleapis.com",
			},
		},
		{
			name: "strict denies unofficial host",
			cfg: Config{
				APIKey:  "k",
				BaseURL: "https://example.com",
			},
			wantErr: true,
		},
		{
			name: "strict denies http",
			cfg: Config{
				APIKey:  "k",
				BaseURL: "http://generativelanguage.googleapis.com",
			},
			wantErr: true,
		},
		{
			name: "strict denies userinfo",
			cfg: Config{
				APIKey:  "k",
				BaseURL: "https://user:pass@generativelanguage.googleapis.com",
			},
			wantErr: true,
		},
		{
			name: "unsafe override allows custom http endpoint",
			cfg: Config{
				APIKey:         "k",
				EndpointURL:    "http://127.0.0.1:8080/v1beta/models/test:generateContent",
				AllowUnsafeURL: true,
			},
		},
		{
			name: "unsafe still requires http or https",
			cfg: Config{
				APIKey:         "k",
				EndpointURL:    "ftp://127.0.0.1:8080/v1beta/models/test:generateContent",
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
		_, _ = w.Write([]byte(`{"modelVersion":"gemini-test","candidates":[{"content":{"parts":[{"text":"{\"ok\":true}"}]}}]}`))
	}))
	defer srv.Close()

	j, err := New(Config{
		APIKey:         "gemini-key",
		EndpointURL:    srv.URL,
		Model:          "gemini-test",
		AllowUnsafeURL: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	res, err := j.EvaluateJSON(context.Background(), sampleReq())
	if err != nil {
		t.Fatalf("EvaluateJSON error: %v", err)
	}
	if res.Provider != "gemini" {
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
