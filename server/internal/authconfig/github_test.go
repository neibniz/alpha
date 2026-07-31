package authconfig

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/auth"
	"github.com/neibniz/alpha/server/internal/privacy"
)

func TestGitHubConfigEncryptsAndRedactsSecret(t *testing.T) {
	codec, err := privacy.NewCodec(privacy.Keyring{
		ActiveKeyID:   "test",
		EncryptionKey: map[string]string{"test": base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 32)))},
		IndexKey:      base64.StdEncoding.EncodeToString([]byte(strings.Repeat("b", 32))),
	})
	if err != nil {
		t.Fatal(err)
	}

	stored, err := StoreGitHub(auth.GithubAuthConfig{
		ClientId:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "https://panel.example/",
	}, codec)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, "client-secret") || !strings.Contains(stored, "enc:v1:test:") {
		t.Fatalf("secret was not encrypted: %s", stored)
	}

	opened, err := LoadGitHub(stored, codec)
	if err != nil {
		t.Fatal(err)
	}
	if opened.ClientSecret != "client-secret" {
		t.Fatalf("opened secret = %q", opened.ClientSecret)
	}

	redacted, err := RedactGitHub(stored)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(redacted), "enc:v1:") || strings.Contains(string(redacted), "client-secret") {
		t.Fatalf("redacted config leaked secret: %s", redacted)
	}
	var view GitHubAdminConfig
	if err := json.Unmarshal(redacted, &view); err != nil {
		t.Fatal(err)
	}
	if view.ClientID != "client-id" || !view.ClientSecretConfigured || view.RedirectURL != "https://panel.example/" {
		t.Fatalf("unexpected redacted config: %+v", view)
	}
}

func TestValidateGitHubConfig(t *testing.T) {
	valid := auth.GithubAuthConfig{
		ClientId:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "https://panel.example/",
	}
	if err := ValidateGitHub(valid); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	tests := []auth.GithubAuthConfig{
		{ClientSecret: "secret", RedirectURL: "https://panel.example/"},
		{ClientId: "id", RedirectURL: "https://panel.example/"},
		{ClientId: "id", ClientSecret: "secret", RedirectURL: "javascript:alert(1)"},
		{ClientId: "id", ClientSecret: "secret", RedirectURL: "https://panel.example/oauth/github/"},
	}
	for _, config := range tests {
		if err := ValidateGitHub(config); err == nil {
			t.Fatalf("invalid config accepted: %+v", config)
		}
	}
}
