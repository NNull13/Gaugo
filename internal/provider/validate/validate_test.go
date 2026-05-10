package validate

import "testing"

func TestBaseURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{name: "empty ok", in: "", wantErr: false},
		{name: "https ok", in: "https://api.example.com/v1", wantErr: false},
		{name: "http ok", in: "http://127.0.0.1:11434/v1", wantErr: false},
		{name: "missing scheme", in: "api.example.com/v1", wantErr: true},
		{name: "unsupported scheme", in: "ftp://example.com", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := BaseURL(tc.in)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCloudURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		in          string
		allowUnsafe bool
		hosts       []string
		wantErr     bool
	}{
		{name: "empty ok", in: "", hosts: []string{"api.openai.com"}, wantErr: false},
		{name: "strict https official host ok", in: "https://api.openai.com/v1", hosts: []string{"api.openai.com"}, wantErr: false},
		{name: "strict http denied", in: "http://api.openai.com/v1", hosts: []string{"api.openai.com"}, wantErr: true},
		{name: "strict unofficial host denied", in: "https://example.com/v1", hosts: []string{"api.openai.com"}, wantErr: true},
		{name: "strict userinfo denied", in: "https://user:pass@api.openai.com/v1", hosts: []string{"api.openai.com"}, wantErr: true},
		{name: "unsafe override allows http unofficial host", in: "http://localhost:8080/v1", allowUnsafe: true, hosts: []string{"api.openai.com"}, wantErr: false},
		{name: "unsafe keeps absolute http https validation", in: "ftp://localhost:8080/v1", allowUnsafe: true, hosts: []string{"api.openai.com"}, wantErr: true},
		{name: "unsafe still denies userinfo", in: "https://user:pass@localhost:8080/v1", allowUnsafe: true, hosts: []string{"api.openai.com"}, wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := CloudURL(tc.in, tc.allowUnsafe, tc.hosts...)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
