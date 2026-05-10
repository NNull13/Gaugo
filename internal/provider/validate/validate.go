package validate

import (
	"fmt"
	"net/url"
	"strings"
)

// BaseURL validates an optional absolute HTTP(S) base URL.
func BaseURL(raw string) error {
	_, _, err := parseAbsoluteHTTPURL(raw)
	return err
}

// CloudURL validates cloud provider base/endpoint URLs.
//
// In strict mode (AllowUnsafeURL=false), URL must use https and one of the
// provided official hosts. In unsafe mode, host/scheme strictness is relaxed,
// but the URL must still be absolute and use http/https.
func CloudURL(raw string, allowUnsafeURL bool, officialHosts ...string) error {
	raw, u, err := parseAbsoluteHTTPURL(raw)
	if err != nil || u == nil {
		return err
	}
	if u.User != nil {
		return fmt.Errorf("invalid base URL %q: user info is not allowed", raw)
	}
	if allowUnsafeURL {
		return nil
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return fmt.Errorf("invalid base URL %q: scheme must be https", raw)
	}

	host := normalizeHost(u.Hostname())
	allowedHosts := normalizeHosts(officialHosts)
	if len(allowedHosts) == 0 {
		return fmt.Errorf("invalid base URL %q: no official hosts configured", raw)
	}
	for _, allowed := range allowedHosts {
		if host == allowed {
			return nil
		}
	}
	return fmt.Errorf("invalid base URL %q: host must be one of %s", raw, strings.Join(allowedHosts, ", "))
}

func parseAbsoluteHTTPURL(raw string) (string, *url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw, nil, fmt.Errorf("invalid base URL %q: %w", raw, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return raw, nil, fmt.Errorf("invalid base URL %q: must be absolute", raw)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return raw, u, nil
	default:
		return raw, nil, fmt.Errorf("invalid base URL %q: scheme must be http or https", raw)
	}
}

func normalizeHost(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}

func normalizeHosts(hosts []string) []string {
	out := make([]string, 0, len(hosts))
	for _, host := range hosts {
		host = normalizeHost(host)
		if host == "" {
			continue
		}
		out = append(out, host)
	}
	return out
}
