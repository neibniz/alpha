package logger

import (
	"fmt"
	"strings"
	"testing"
)

func TestSanitizeLogValueMasksNestedPrivacyFields(t *testing.T) {
	value := map[string]any{
		"auth_identifier": "user@example.com",
		"client_ip":       "203.0.113.42",
		"subscribe_token": "private-subscribe-token",
		"safe_count":      7,
	}
	got, ok := sanitizeLogValue("request", value).(map[string]any)
	if !ok {
		t.Fatalf("sanitized value has type %T", sanitizeLogValue("request", value))
	}
	for _, key := range []string{"auth_identifier", "client_ip", "subscribe_token"} {
		if got[key] != "***" {
			t.Fatalf("sanitized %s = %#v, want redacted", key, got[key])
		}
	}
	if fmt.Sprint(got["safe_count"]) != "7" {
		t.Fatalf("safe field changed: %#v", got["safe_count"])
	}
}

func TestSanitizeLogValueMasksTopLevelIdentity(t *testing.T) {
	got, ok := sanitizeLogValue("device_id", "device-fingerprint-abcdef").(string)
	if !ok || got == "device-fingerprint-abcdef" || !strings.Contains(got, "...") {
		t.Fatalf("sanitized device ID = %#v", got)
	}
}
