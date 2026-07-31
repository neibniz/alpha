package paymenturl

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"path"
	"strings"
)

// NotifyURL builds a payment callback URL from administrator-controlled base URLs.
func NotifyURL(paymentDomain, siteURL, platform, token string, gatewayMode bool) (string, error) {
	base := strings.TrimSpace(paymentDomain)
	if base == "" {
		base = strings.TrimSpace(siteURL)
	}
	parsed, err := parseBaseURL(base)
	if err != nil {
		return "", err
	}
	if err := validatePathSegment("platform", platform); err != nil {
		return "", err
	}
	if err := validatePathSegment("token", token); err != nil {
		return "", err
	}

	segments := []string{"v1", "notify", platform, token}
	if gatewayMode {
		segments = append([]string{"api"}, segments...)
	}
	parsed.Path = path.Join(append([]string{parsed.Path}, segments...)...)
	parsed.RawPath = ""
	return parsed.String(), nil
}

// NormalizeBaseURL validates and canonicalizes a configured payment callback base URL.
func NormalizeBaseURL(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	parsed, err := parseBaseURL(raw)
	if err != nil {
		return "", err
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String(), nil
}

// ValidateReturnURL prevents checkout requests from turning the payment provider into an open redirect.
func ValidateReturnURL(raw string, trustedBaseURLs ...string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	redirect, err := parseHTTPURL(raw, true)
	if err != nil {
		return "", fmt.Errorf("invalid payment return URL: %w", err)
	}

	for _, trusted := range trustedBaseURLs {
		base, parseErr := parseBaseURL(trusted)
		if parseErr == nil && sameOrigin(redirect, base) {
			return redirect.String(), nil
		}
	}
	return "", errors.New("payment return URL origin is not trusted")
}

func parseBaseURL(raw string) (*url.URL, error) {
	parsed, err := parseHTTPURL(strings.TrimSpace(raw), false)
	if err != nil {
		return nil, fmt.Errorf("invalid payment callback base URL: %w", err)
	}
	return parsed, nil
}

func parseHTTPURL(raw string, allowQueryAndFragment bool) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("URL must use http or https")
	}
	if parsed.Hostname() == "" || parsed.User != nil || parsed.Opaque != "" {
		return nil, errors.New("URL must be absolute and must not contain credentials")
	}
	if !allowQueryAndFragment && (parsed.RawQuery != "" || parsed.Fragment != "") {
		return nil, errors.New("base URL must not contain a query or fragment")
	}
	return parsed, nil
}

func validatePathSegment(name, value string) error {
	if value == "" || value == "." || value == ".." || len(value) > 255 || strings.ContainsAny(value, "/\\") {
		return fmt.Errorf("invalid payment callback %s", name)
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return fmt.Errorf("invalid payment callback %s", name)
		}
	}
	return nil
}

func sameOrigin(first, second *url.URL) bool {
	return canonicalOrigin(first) == canonicalOrigin(second)
}

func canonicalOrigin(value *url.URL) string {
	port := value.Port()
	if port == "" {
		if value.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return strings.ToLower(value.Scheme) + "://" + net.JoinHostPort(strings.ToLower(value.Hostname()), port)
}
