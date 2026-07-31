package config

import "testing"

func TestSessionCacheKeyKeepsExistingFormat(t *testing.T) {
	if got := SessionCacheKey("session-123"); got != "auth:session_id:session-123" {
		t.Fatalf("SessionCacheKey() = %q, want auth:session_id:session-123", got)
	}
}

func TestSessionUserValueKeepsExistingFormat(t *testing.T) {
	if got := SessionUserValue(42); got != "42" {
		t.Fatalf("SessionUserValue() = %q, want 42", got)
	}
}

func TestUserSessionCacheKey(t *testing.T) {
	if got := UserSessionCacheKey(42); got != "auth:user_sessions:42" {
		t.Fatalf("UserSessionCacheKey() = %q, want auth:user_sessions:42", got)
	}
}

func TestTelegramBindCacheKey(t *testing.T) {
	if got := TelegramBindCacheKey("token-123"); got != "auth:telegram_bind:token-123" {
		t.Fatalf("TelegramBindCacheKey() = %q", got)
	}
}

func TestAuthCodeCacheKeyHelpersKeepExistingFormat(t *testing.T) {
	if got := EmailAuthCodeCacheKey("security", "user@example.com"); got != "auth:verify:email:security:user@example.com" {
		t.Fatalf("EmailAuthCodeCacheKey() = %q, want auth:verify:email:security:user@example.com", got)
	}
}

func TestSendLimitCacheKeyHelpersKeepExistingFormat(t *testing.T) {
	if got := SendIntervalCacheKey("email", "register"); got != "send:interval::email:register" {
		t.Fatalf("SendIntervalCacheKey() = %q, want send:interval::email:register", got)
	}
	if got := SendCountLimitValue("email", "security", "user@example.com"); got != "email:security:user@example.com" {
		t.Fatalf("SendCountLimitValue() = %q, want email:security:user@example.com", got)
	}
}
