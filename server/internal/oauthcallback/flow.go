package oauthcallback

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path"
	"strings"
)

const (
	GitHubLoginPath       = "/oauth/github"
	GitHubBindPath        = "/bind/github"
	GitHubLoginBridgePath = "/oauth/github/index.html"
	GitHubBindBridgePath  = "/bind/github/index.html"
	GitHubLoginStateScope = "github:login"
	GitHubBindStateScope  = "github:bind"
)

func NewState() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func ValidateBaseURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("parse callback base URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("callback base URL must use http or https")
	}
	if parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" || parsed.RawQuery != "" {
		return nil, errors.New("callback base URL must be an absolute URL without credentials, query, or fragment")
	}
	return parsed, nil
}

func ValidateRedirect(raw, callbackBase string, expectedPaths ...string) (string, error) {
	redirect, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("parse oauth redirect URL: %w", err)
	}
	base, err := ValidateBaseURL(callbackBase)
	if err != nil {
		return "", err
	}
	if redirect.Scheme != "http" && redirect.Scheme != "https" {
		return "", errors.New("oauth redirect URL must use http or https")
	}
	if redirect.Hostname() == "" || redirect.User != nil || redirect.Fragment != "" || redirect.RawQuery != "" {
		return "", errors.New("oauth redirect URL must be absolute and must not contain credentials, query, or fragment")
	}
	if origin(redirect) != origin(base) {
		return "", errors.New("oauth redirect origin does not match configured callback URL")
	}

	redirectPath := cleanURLPath(redirect.Path)
	pathAllowed := false
	for _, expectedPath := range expectedPaths {
		if redirectPath == cleanURLPath(expectedPath) {
			pathAllowed = true
			break
		}
	}
	if !pathAllowed {
		return "", fmt.Errorf("oauth redirect path is not allowed: %s", redirectPath)
	}
	basePath := cleanURLPath(base.Path)
	if basePath != "/" && redirectPath != basePath && !strings.HasPrefix(redirectPath, strings.TrimSuffix(basePath, "/")+"/") {
		return "", errors.New("oauth redirect path is outside configured callback URL")
	}
	return redirect.String(), nil
}

func origin(value *url.URL) string {
	host := strings.ToLower(value.Hostname())
	port := value.Port()
	if port == "" {
		if value.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return strings.ToLower(value.Scheme) + "://" + net.JoinHostPort(host, port)
}

func cleanURLPath(value string) string {
	cleaned := path.Clean("/" + strings.TrimSpace(value))
	if cleaned == "." {
		return "/"
	}
	return cleaned
}
