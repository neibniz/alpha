package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestMaskSensitiveFieldsRecursively(t *testing.T) {
	input := []byte(`{
		"email":"user@example.com",
		"profile":{"old_password":"old","auth_identifier":"oauth-private-id","client_ip":"203.0.113.42","user_agent":"PrivateClient/1.0","name":"Alice"},
		"tokens":[{"access-token":"access"},{"nested":{"private_key":"secret"}}],
		"amount":123
	}`)

	got := maskSensitiveFields(input, []string{"custom_secret"})

	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("masked payload should remain valid json: %v", err)
	}
	if decoded["email"] != maskedLogValue {
		t.Fatalf("email should be masked: %s", got)
	}
	profile := decoded["profile"].(map[string]any)
	if profile["old_password"] != maskedLogValue || profile["auth_identifier"] != maskedLogValue || profile["client_ip"] != maskedLogValue || profile["user_agent"] != maskedLogValue || profile["name"] != "Alice" {
		t.Fatalf("profile masking mismatch: %#v", profile)
	}
	tokens := decoded["tokens"].([]any)
	firstToken := tokens[0].(map[string]any)
	if firstToken["access-token"] != maskedLogValue {
		t.Fatalf("access token should be masked: %#v", firstToken)
	}
	secondToken := tokens[1].(map[string]any)
	nested := secondToken["nested"].(map[string]any)
	if nested["private_key"] != maskedLogValue {
		t.Fatalf("private key should be masked: %#v", nested)
	}
	if decoded["amount"].(float64) != 123 {
		t.Fatalf("non-sensitive fields should be preserved: %#v", decoded)
	}
}

func TestLogBodyReadCloserCapturesOnlyBytesConsumedByHandler(t *testing.T) {
	buffer := newLogBodyBuffer(16)
	reader := &logBodyReadCloser{ReadCloser: io.NopCloser(bytes.NewBufferString("request-body")), body: buffer}
	payload := make([]byte, 7)
	if _, err := io.ReadFull(reader, payload); err != nil {
		t.Fatal(err)
	}
	if got := buffer.LogString(nil); got != "request" {
		t.Fatalf("captured request body = %q, want request", got)
	}
}

func TestMaskSensitiveFieldsKeepsInvalidJSON(t *testing.T) {
	input := []byte(`{"password":`)
	got := maskSensitiveFields(input, nil)
	if string(got) != string(input) {
		t.Fatalf("invalid json should be unchanged, got %q", got)
	}
}

func TestLogBodyStringOmitsLargePayload(t *testing.T) {
	got := logBodyString([]byte(strings.Repeat("x", 12)), nil, 8)
	if !strings.Contains(got, "omitted") || !strings.Contains(got, "12 bytes") {
		t.Fatalf("large payload should be omitted, got %q", got)
	}
}

func TestLogBodyBufferCapsStoredBody(t *testing.T) {
	buf := newLogBodyBuffer(5)
	n, err := buf.Write([]byte("123456789"))
	if err != nil || n != 9 {
		t.Fatalf("write should report original length, n=%d err=%v", n, err)
	}
	if buf.Buffer.Len() != 5 {
		t.Fatalf("buffer should retain only limit bytes, got %d", buf.Buffer.Len())
	}
	got := buf.LogString(nil)
	if !strings.Contains(got, "9 bytes") || strings.Contains(got, "12345") {
		t.Fatalf("truncated body should be omitted, got %q", got)
	}
}
