package main

import "testing"

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
