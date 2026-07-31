package types

import (
	"encoding/json"
	"testing"
)

func TestEncodeDecodeVerifyEmailContent(t *testing.T) {
	raw, err := EncodeEmailContent(VerifyEmailContent{
		Type:     1,
		SiteLogo: "logo",
		SiteName: "Alpha",
		Expire:   5,
		Code:     "123456",
	})
	if err != nil {
		t.Fatalf("encode verify email content: %v", err)
	}

	payload, err := json.Marshal(SendEmailPayload{
		Type:    EmailTypeVerify,
		Email:   "user@example.com",
		Subject: "Verification code",
		Content: raw,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	var decoded struct {
		Content VerifyEmailContent `json:"content"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if decoded.Content.Type != 1 || decoded.Content.SiteName != "Alpha" || decoded.Content.Code != "123456" {
		t.Fatalf("unexpected decoded content: %+v", decoded.Content)
	}

	content, err := DecodeEmailContent[VerifyEmailContent](raw)
	if err != nil {
		t.Fatalf("decode verify email content: %v", err)
	}
	if content != decoded.Content {
		t.Fatalf("decoded content mismatch: %+v != %+v", content, decoded.Content)
	}
}

func TestDecodeEmailContentRejectsEmptyContent(t *testing.T) {
	if _, err := DecodeEmailContent[VerifyEmailContent](nil); err == nil {
		t.Fatalf("empty content should fail")
	}
}
