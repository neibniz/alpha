package oauthcallback

import (
	"strings"
	"testing"
)

func TestNewStateUsesURLSafeEntropy(t *testing.T) {
	first, err := NewState()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewState()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 43 || first == second || strings.ContainsAny(first, "+/=") {
		t.Fatalf("unexpected state values: %q %q", first, second)
	}
}

func TestValidateRedirect(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		base    string
		path    string
		wantErr bool
	}{
		{name: "login", raw: "https://panel.example/oauth/github/callback", base: "https://panel.example/", path: GitHubLoginPath},
		{name: "bind", raw: "https://panel.example/bind/github/callback", base: "https://panel.example/", path: GitHubBindPath},
		{name: "default https port", raw: "https://panel.example:443/oauth/github/callback", base: "https://panel.example/", path: GitHubLoginPath},
		{name: "legacy login path", raw: "https://panel.example/oauth/github", base: "https://panel.example/", path: GitHubLoginPath, wantErr: true},
		{name: "legacy bind path", raw: "https://panel.example/bind/github", base: "https://panel.example/", path: GitHubBindPath, wantErr: true},
		{name: "wrong origin", raw: "https://evil.example/oauth/github/callback", base: "https://panel.example/", path: GitHubLoginPath, wantErr: true},
		{name: "subdomain", raw: "https://evil.panel.example/oauth/github/callback", base: "https://panel.example/", path: GitHubLoginPath, wantErr: true},
		{name: "wrong path", raw: "https://panel.example/profile", base: "https://panel.example/", path: GitHubLoginPath, wantErr: true},
		{name: "query injection", raw: "https://panel.example/oauth/github/callback?next=https://evil.example", base: "https://panel.example/", path: GitHubLoginPath, wantErr: true},
		{name: "outside base path", raw: "https://panel.example/oauth/github/callback", base: "https://panel.example/callback", path: GitHubLoginPath, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateRedirect(tt.raw, tt.base, tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateRedirect() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBaseURLRejectsUnsafeValues(t *testing.T) {
	for _, raw := range []string{"", "javascript:alert(1)", "https://user@panel.example/", "https://panel.example/?next=evil", "https://panel.example/#fragment"} {
		if _, err := ValidateBaseURL(raw); err == nil {
			t.Fatalf("ValidateBaseURL(%q) should fail", raw)
		}
	}
}
