package oauthcallback

import (
	"encoding/json"
	"testing"
)

func TestDecodeStateCode(t *testing.T) {
	got, err := Decode[StateCode](json.RawMessage(`{"code":"abc","state":"xyz"}`))
	if err != nil {
		t.Fatalf("decode state code: %v", err)
	}
	if got.Code != "abc" || got.State != "xyz" {
		t.Fatalf("unexpected state code: %+v", got)
	}
}

func TestDecodeTelegram(t *testing.T) {
	got, err := Decode[Telegram](json.RawMessage(`{"state":"xyz","tgAuthResult":"payload"}`))
	if err != nil {
		t.Fatalf("decode telegram: %v", err)
	}
	if got.State != "xyz" || got.TgAuthResult != "payload" {
		t.Fatalf("unexpected telegram callback: %+v", got)
	}
}

func TestNewStateCodeValidatesRequiredFields(t *testing.T) {
	got, err := NewStateCode("abc", "xyz")
	if err != nil {
		t.Fatalf("new state code: %v", err)
	}
	if got.Code != "abc" || got.State != "xyz" {
		t.Fatalf("unexpected state code: %+v", got)
	}
	if _, err := NewStateCode("", "xyz"); err == nil {
		t.Fatal("empty code should be rejected")
	}
	if _, err := NewStateCode("abc", ""); err == nil {
		t.Fatal("empty state should be rejected")
	}
}

func TestNewTelegramValidatesRequiredFields(t *testing.T) {
	got, err := NewTelegram("xyz", "payload")
	if err != nil {
		t.Fatalf("new telegram callback: %v", err)
	}
	if got.State != "xyz" || got.TgAuthResult != "payload" {
		t.Fatalf("unexpected telegram callback: %+v", got)
	}
	if _, err := NewTelegram("", "payload"); err == nil {
		t.Fatal("empty state should be rejected")
	}
	if _, err := NewTelegram("xyz", ""); err == nil {
		t.Fatal("empty telegram auth result should be rejected")
	}
}

func TestDecodeRejectsEmptyAndInvalidJSON(t *testing.T) {
	if _, err := Decode[StateCode](nil); err == nil {
		t.Fatalf("empty callback should be rejected")
	}
	if _, err := Decode[StateCode](json.RawMessage(`{`)); err == nil {
		t.Fatalf("invalid json should be rejected")
	}
}
