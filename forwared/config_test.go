package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigValidateAppliesTelemetryDefaults(t *testing.T) {
	cfg := Config{
		BaseURL:   "https://panel.example.com",
		SecretKey: "0123456789abcdef0123456789abcdef",
		GeoIPFile: "/tmp/geoip.bin",
		LogFile:   "/tmp/forwared.log",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.EventBatchSize != 500 || cfg.EventFlushInterval != 5 {
		t.Fatalf("unexpected event defaults: size=%d interval=%d", cfg.EventBatchSize, cfg.EventFlushInterval)
	}
	if cfg.MaxSpoolBytes != 64<<20 {
		t.Fatalf("unexpected spool limit: %d", cfg.MaxSpoolBytes)
	}
}

func TestConfigValidateRejectsWeakSecret(t *testing.T) {
	cfg := Config{
		BaseURL:   "https://panel.example.com",
		SecretKey: "too-short",
		GeoIPFile: "/tmp/geoip.bin",
		LogFile:   "/tmp/forwared.log",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected a weak secret to be rejected")
	}
}

func TestConfigLoadsEnvironmentWithoutFile(t *testing.T) {
	t.Setenv("ALPHA_FORWARED_BASE_URL", "https://api.example.test")
	t.Setenv("ALPHA_FORWARED_SECRET_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("ALPHA_FORWARED_DATA_DIR", t.TempDir())
	t.Setenv("ALPHA_FORWARED_IP2LOCATION_TOKEN", "test-download-token")
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogFile != "" || cfg.GeoIPUpdateInterval != 86400 || cfg.GeoIPDownloadCode != "DB3LITEBINIPV6" || cfg.GeoIPDownloadToken != "test-download-token" {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}
	for _, path := range []string{cfg.GeoIPFile, cfg.StateFile, cfg.SpoolFile} {
		if !filepath.IsLocal(mustRelative(t, cfg.DataDir, path)) {
			t.Fatalf("data file outside data directory: %s", path)
		}
	}
}

func mustRelative(t *testing.T, base, path string) string {
	t.Helper()
	rel, err := filepath.Rel(base, path)
	if err != nil {
		t.Fatal(err)
	}
	return rel
}

func TestConfigPreservesExplicitLegacyPathsAndEnvironmentPrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.yaml")
	err := os.WriteFile(path, []byte("base_url: https://old.example.test\nsecret_key: 0123456789abcdef0123456789abcdef\ngeoipfile: /app/config/old.BIN\nstate_file: /app/state/identity.json\nspool_file: /app/state/events.jsonl\nlogfile: /app/log/old.log\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("ALPHA_FORWARED_BASE_URL", "https://new.example.test")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "https://new.example.test" || cfg.GeoIPFile != "/app/config/old.BIN" || cfg.StateFile != "/app/state/identity.json" || cfg.LogFile != "/app/log/old.log" {
		t.Fatalf("legacy configuration changed: %#v", cfg)
	}
	if _, err := LoadConfig(path + ".missing"); err == nil {
		t.Fatal("explicit missing config accepted")
	}
}

func TestConfigRejectsUnsafeUpdaterSettings(t *testing.T) {
	for _, source := range []string{"http://example.test/db.zip", "https://user:password@example.test/db.zip", "https://example.test/db.zip#fragment"} {
		cfg := Config{BaseURL: "https://api.example.test", SecretKey: "0123456789abcdef0123456789abcdef", GeoIPDownloadURL: source}
		if err := cfg.Validate(); err == nil {
			t.Fatalf("unsafe source accepted: %s", source)
		}
	}
	for _, interval := range []uint{1, 3599, 30*86400 + 1} {
		cfg := Config{BaseURL: "https://api.example.test", SecretKey: "0123456789abcdef0123456789abcdef", GeoIPUpdateInterval: interval}
		if err := cfg.Validate(); err == nil {
			t.Fatalf("invalid interval accepted: %d", interval)
		}
	}
}
