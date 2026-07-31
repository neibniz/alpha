package auth

import (
	"encoding/json"
	"testing"
)

type authConfigMarshaler interface {
	Marshal() string
}

func TestSimpleAuthConfigMarshalShapes(t *testing.T) {
	tests := []struct {
		name  string
		value authConfigMarshaler
		want  string
	}{
		{
			name:  "apple",
			value: &AppleAuthConfig{TeamID: "team", KeyID: "key", ClientId: "client", ClientSecret: "secret", RedirectURL: "https://example.test/apple"},
			want:  `{"team_id":"team","key_id":"key","client_id":"client","client_secret":"secret","redirect_url":"https://example.test/apple"}`,
		},
		{
			name:  "google",
			value: &GoogleAuthConfig{ClientId: "client", ClientSecret: "secret", RedirectURL: "https://example.test/google"},
			want:  `{"client_id":"client","client_secret":"secret","redirect_url":"https://example.test/google"}`,
		},
		{
			name:  "github",
			value: &GithubAuthConfig{ClientId: "client", ClientSecret: "secret", RedirectURL: "https://example.test/github"},
			want:  `{"client_id":"client","client_secret":"secret","redirect_url":"https://example.test/github"}`,
		},
		{
			name:  "facebook",
			value: &FacebookAuthConfig{ClientId: "client", ClientSecret: "secret", RedirectURL: "https://example.test/facebook"},
			want:  `{"client_id":"client","client_secret":"secret","redirect_url":"https://example.test/facebook"}`,
		},
		{
			name:  "telegram",
			value: &TelegramAuthConfig{BotToken: "bot", EnableNotify: true, WebHookDomain: "https://example.test/hook"},
			want:  `{"bot_token":"bot","enable_notify":true,"webhook_domain":"https://example.test/hook"}`,
		},
		{
			name:  "smtp",
			value: &SMTPConfig{Host: "smtp.example.test", Port: 587, User: "user", Pass: "pass", From: "noreply@example.test", SSL: true},
			want:  `{"host":"smtp.example.test","port":587,"user":"user","pass":"pass","from":"noreply@example.test","ssl":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.Marshal(); got != tt.want {
				t.Fatalf("Marshal() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestSimpleAuthConfigMarshalNilPointerShape(t *testing.T) {
	var cfg *AppleAuthConfig
	if got := cfg.Marshal(); got != "null" {
		t.Fatalf("nil AppleAuthConfig.Marshal() = %s, want null", got)
	}
}

func TestEmailAuthConfigKeepsPlatformConfigRaw(t *testing.T) {
	var cfg EmailAuthConfig
	cfg.Unmarshal(`{"platform":"smtp","platform_config":{"host":"smtp.example.com","port":587}}`)
	if cfg.Platform != "smtp" {
		t.Fatalf("platform = %q", cfg.Platform)
	}
	if string(cfg.PlatformConfig) != `{"host":"smtp.example.com","port":587}` {
		t.Fatalf("platform config = %s", cfg.PlatformConfig)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		PlatformConfig json.RawMessage `json:"platform_config"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if string(payload.PlatformConfig) != `{"host":"smtp.example.com","port":587}` {
		t.Fatalf("marshaled platform config = %s", payload.PlatformConfig)
	}
}

func TestAuthConfigDefaultsUseStableObjects(t *testing.T) {
	var emailPayload struct {
		Platform       string          `json:"platform"`
		PlatformConfig json.RawMessage `json:"platform_config"`
	}
	if err := json.Unmarshal([]byte(new(EmailAuthConfig).Marshal()), &emailPayload); err != nil {
		t.Fatal(err)
	}
	if emailPayload.Platform != "smtp" || !json.Valid(emailPayload.PlatformConfig) || string(emailPayload.PlatformConfig) == "null" {
		t.Fatalf("default email config shape mismatch: %+v platform_config=%s", emailPayload, emailPayload.PlatformConfig)
	}
}

func TestAuthConfigWhitespaceNullPlatformConfigDefaults(t *testing.T) {
	emailCfg := EmailAuthConfig{Platform: "smtp", PlatformConfig: json.RawMessage(" \n null\t")}
	emailCfg.applyDefaults()
	var smtp SMTPConfig
	if err := json.Unmarshal(emailCfg.PlatformConfig, &smtp); err != nil {
		t.Fatalf("default email platform config is invalid: %v", err)
	}
}

func TestAuthConfigFallbacksKeepPlatformConfigObject(t *testing.T) {
	var emailCfg EmailAuthConfig
	emailCfg.Unmarshal(`{`)
	if !json.Valid(emailCfg.PlatformConfig) || string(emailCfg.PlatformConfig) == "null" {
		t.Fatalf("email fallback platform config = %s", emailCfg.PlatformConfig)
	}
}

func TestAuthSummaryColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(authSummaryColumns))
	for _, column := range authSummaryColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "method", "enabled"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("auth summary columns must include %q", column)
		}
	}

	if _, ok := columns["config"]; ok {
		t.Fatal("auth summary columns must not include config")
	}
}

func TestAuthAdminSelectColumnsIncludeConfigOnlyOnDemand(t *testing.T) {
	withoutConfig := make(map[string]struct{}, len(authAdminSelectColumns(false)))
	for _, column := range authAdminSelectColumns(false) {
		withoutConfig[column] = struct{}{}
	}
	if _, ok := withoutConfig["config"]; ok {
		t.Fatal("auth admin summary columns must not include config by default")
	}

	withConfig := make(map[string]struct{}, len(authAdminSelectColumns(true)))
	for _, column := range authAdminSelectColumns(true) {
		withConfig[column] = struct{}{}
	}
	if _, ok := withConfig["config"]; !ok {
		t.Fatal("auth admin detail columns must include config when requested")
	}
}
