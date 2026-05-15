package validate

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nnull13/gaugo/internal/failure"
)

const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"
)

const (
	baseURLErrorUserInfoNotAllowed = "user info is not allowed"
	baseURLErrorSchemeHTTPS        = "scheme must be https"
	baseURLErrorNoOfficialHosts    = "no official hosts configured"
	baseURLErrorMustBeAbsolute     = "must be absolute"
	baseURLErrorSchemeHTTPHTTPS    = "scheme must be http or https"
	baseURLErrorHostOneOfFormat    = "host must be one of %s"
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
		return baseURLError(raw, baseURLErrorUserInfoNotAllowed)
	}
	if allowUnsafeURL {
		return nil
	}
	if !strings.EqualFold(u.Scheme, schemeHTTPS) {
		return baseURLError(raw, baseURLErrorSchemeHTTPS)
	}

	host := normalizeHost(u.Hostname())
	allowedHosts := normalizeHosts(officialHosts)
	if len(allowedHosts) == 0 {
		return baseURLError(raw, baseURLErrorNoOfficialHosts)
	}
	for _, allowed := range allowedHosts {
		if host == allowed {
			return nil
		}
	}
	return baseURLErrorf(raw, baseURLErrorHostOneOfFormat, strings.Join(allowedHosts, ", "))
}

func parseAbsoluteHTTPURL(raw string) (string, *url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw, nil, baseURLWrapError(raw, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return raw, nil, baseURLError(raw, baseURLErrorMustBeAbsolute)
	}
	switch strings.ToLower(u.Scheme) {
	case schemeHTTP, schemeHTTPS:
		return raw, u, nil
	default:
		return raw, nil, baseURLError(raw, baseURLErrorSchemeHTTPHTTPS)
	}
}

func baseURLError(raw, detail string) error {
	return failure.Validation(
		failure.CodeValidationInvalid,
		"provider.validate_url",
		"base_url",
		failure.FormatInvalidURL(raw, detail),
		nil,
	)
}

func baseURLErrorf(raw, format string, args ...any) error {
	return baseURLError(raw, fmt.Sprintf(format, args...))
}

func baseURLWrapError(raw string, err error) error {
	return failure.Validation(
		failure.CodeValidationInvalid,
		"provider.validate_url",
		"base_url",
		failure.FormatInvalidURL(raw, err.Error()),
		err,
	)
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
