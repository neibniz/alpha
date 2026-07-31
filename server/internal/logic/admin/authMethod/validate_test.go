package authMethod

import (
	"encoding/json"
	"testing"

	modelAuth "github.com/neibniz/alpha/server/internal/model/auth"
	"github.com/neibniz/alpha/server/pkg/email"
)

func TestNormalizeAuthPlatformConfigRaw(t *testing.T) {
	raw := json.RawMessage(`{"host":"smtp.example.com","port":587,"user":"mailer","pass":"secret","from":"noreply@example.com","ssl":true}`)

	platformConfig, err := normalizeAuthPlatformConfigRaw(email.SMTP.String(), raw)
	if err != nil {
		t.Fatalf("normalizeAuthPlatformConfigRaw returned error: %v", err)
	}

	var cfg modelAuth.SMTPConfig
	if err := json.Unmarshal(platformConfig, &cfg); err != nil {
		t.Fatalf("platform config is invalid json: %v", err)
	}
	if cfg.Host != "smtp.example.com" || cfg.Port != 587 {
		t.Fatalf("platform config not decoded: %+v", cfg)
	}
}

func TestAuthMethodConfigRawUsesDefaultForBlankInput(t *testing.T) {
	raw, err := authMethodConfigRaw("email", json.RawMessage("  \n\t "))
	if err != nil {
		t.Fatalf("authMethodConfigRaw returned error: %v", err)
	}
	if !json.Valid(raw) || len(raw) == 0 {
		t.Fatalf("default config raw = %q, want valid json", raw)
	}
}

func TestAuthMethodConfigRawValidInputAvoidsAllocation(t *testing.T) {
	raw := json.RawMessage(` {"enabled":true} `)
	allocs := testing.AllocsPerRun(1000, func() {
		got, err := authMethodConfigRaw("email", raw)
		if err != nil {
			t.Fatalf("authMethodConfigRaw returned error: %v", err)
		}
		if string(got) != `{"enabled":true}` {
			t.Fatalf("authMethodConfigRaw = %q, want trimmed json", got)
		}
	})
	if allocs != 0 {
		t.Fatalf("authMethodConfigRaw allocations = %v, want 0", allocs)
	}
}

func TestNormalizeEmailAuthMethodConfig(t *testing.T) {
	config, err := normalizeAuthMethodConfig("email", json.RawMessage(`{
		"platform": "smtp",
		"enable_verify": true,
		"platform_config": {
			"host": "smtp.example.com",
			"port": 587,
			"user": "mailer",
			"pass": "secret",
			"from": "noreply@example.com",
			"ssl": true
		}
	}`))
	if err != nil {
		t.Fatalf("normalizeAuthMethodConfig returned error: %v", err)
	}

	var payload struct {
		Platform                 string          `json:"platform"`
		PlatformConfig           json.RawMessage `json:"platform_config"`
		EnableVerify             bool            `json:"enable_verify"`
		MaintenanceEmailTemplate string          `json:"maintenance_email_template"`
	}
	if err := json.Unmarshal([]byte(config), &payload); err != nil {
		t.Fatalf("normalized config is invalid json: %v", err)
	}
	if payload.Platform != email.SMTP.String() || !payload.EnableVerify {
		t.Fatalf("normalized email config mismatch: %+v", payload)
	}
	if payload.MaintenanceEmailTemplate == "" {
		t.Fatal("maintenance template should be defaulted")
	}

	var smtp modelAuth.SMTPConfig
	if err := json.Unmarshal(payload.PlatformConfig, &smtp); err != nil {
		t.Fatalf("smtp config is invalid: %v", err)
	}
	if smtp.Host != "smtp.example.com" || smtp.Port != 587 || !smtp.SSL {
		t.Fatalf("smtp config mismatch: %+v", smtp)
	}
}

func TestRemovedAuthenticationMethodsAreRejected(t *testing.T) {
	for _, method := range []string{"mobile", "device"} {
		if _, err := normalizeAuthMethodConfig(method, json.RawMessage(`{}`)); err == nil {
			t.Fatalf("removed method %q should be rejected", method)
		}
	}
}
