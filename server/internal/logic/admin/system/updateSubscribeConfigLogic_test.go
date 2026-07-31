package system

import (
	"strings"
	"testing"

	"github.com/neibniz/alpha/server/internal/types"
)

func TestValidateSubscribePath(t *testing.T) {
	t.Parallel()

	valid := []string{"", "/v1/subscribe/config", "/alpha-sub", "/sub_v2"}
	for _, path := range valid {
		if err := validateSubscribePath(path); err != nil {
			t.Fatalf("validateSubscribePath(%q) returned error: %v", path, err)
		}
	}

	invalid := []string{
		"/",
		"sub/config",
		"/sub/",
		"/sub//config",
		"/sub/:token",
		"/sub/*path",
		"/sub?token=1",
		"/sub config",
	}
	for _, path := range invalid {
		if err := validateSubscribePath(path); err == nil {
			t.Fatalf("validateSubscribePath(%q) returned nil", path)
		}
	}
}

func TestNormalizeSubscribeConfigNormalizesLists(t *testing.T) {
	req := &types.SubscribeConfig{
		SubscribePath:   " /v1/subscribe/config ",
		SubscribeDomain: " sub.example.com \nsub.example.com\nedge.example.com ",
		UserAgentList:   " Clash Verge \nsing-box\nClash Verge ",
	}
	if err := normalizeSubscribeConfig(req); err != nil {
		t.Fatal(err)
	}
	if req.SubscribePath != "/v1/subscribe/config" {
		t.Fatalf("subscribe path = %q", req.SubscribePath)
	}
	if req.SubscribeDomain != "sub.example.com\nedge.example.com" {
		t.Fatalf("subscribe domain = %q", req.SubscribeDomain)
	}
	if req.UserAgentList != "Clash Verge\nsing-box" {
		t.Fatalf("user agent list = %q", req.UserAgentList)
	}
}

func TestNormalizeSubscribeConfigRejectsInvalidDomainsAndUserAgents(t *testing.T) {
	for _, domain := range []string{
		"*.example.com",
		"https://example.com",
		"example.com:8443",
		"edge..example.com",
		"999.1.1.1",
	} {
		req := &types.SubscribeConfig{SubscribeDomain: domain}
		if err := normalizeSubscribeConfig(req); err == nil {
			t.Fatalf("normalizeSubscribeConfig() accepted domain %q", domain)
		}
	}

	for _, userAgent := range []string{"Clash\x00Meta", strings.Repeat("x", 256)} {
		req := &types.SubscribeConfig{UserAgentList: userAgent}
		if err := normalizeSubscribeConfig(req); err == nil {
			t.Fatalf("normalizeSubscribeConfig() accepted user agent %q", userAgent)
		}
	}
}
