package authaudit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	modelLog "github.com/neibniz/alpha/server/internal/model/log"
)

type captureInserter struct {
	entries []*modelLog.SystemLog
}

func (c *captureInserter) Insert(_ context.Context, entry *modelLog.SystemLog) error {
	c.entries = append(c.entries, entry)
	return nil
}

func TestRecordLoginBuildsCanonicalEvent(t *testing.T) {
	at := time.Date(2026, 7, 10, 12, 30, 0, 0, time.UTC)
	model := &captureInserter{}
	if err := RecordLogin(context.Background(), model, LoginEvent{
		UserID: 42, Method: "github", IP: "192.0.2.5", UserAgent: "Alpha/1", Success: true, At: at,
	}); err != nil {
		t.Fatal(err)
	}
	if len(model.entries) != 1 {
		t.Fatalf("entry count = %d, want 1", len(model.entries))
	}
	entry := model.entries[0]
	if entry.Type != modelLog.TypeLogin.Uint8() || entry.Date != "2026-07-10" || entry.ObjectID != 42 {
		t.Fatalf("unexpected system log: %#v", entry)
	}
	var content modelLog.Login
	if err := json.Unmarshal([]byte(entry.Content), &content); err != nil {
		t.Fatal(err)
	}
	if content.Method != "github" || content.LoginIP != "192.0.2.5" || !content.Success || content.Timestamp != at.UnixMilli() {
		t.Fatalf("unexpected login event: %#v", content)
	}
}

func TestRecordRegisterBuildsCanonicalEvent(t *testing.T) {
	model := &captureInserter{}
	if err := RecordRegister(context.Background(), model, RegisterEvent{
		UserID: 9, Method: "email", Identifier: "user@example.com", IP: "198.51.100.8",
	}); err != nil {
		t.Fatal(err)
	}
	if len(model.entries) != 1 || model.entries[0].Type != modelLog.TypeRegister.Uint8() {
		t.Fatalf("unexpected entries: %#v", model.entries)
	}
	var content modelLog.Register
	if err := json.Unmarshal([]byte(model.entries[0].Content), &content); err != nil {
		t.Fatal(err)
	}
	if content.AuthMethod != "email" || content.Identifier != "user@example.com" || content.RegisterIP != "198.51.100.8" {
		t.Fatalf("unexpected register event: %#v", content)
	}
}

func TestRecordRejectsInvalidEvent(t *testing.T) {
	if err := RecordLogin(context.Background(), &captureInserter{}, LoginEvent{}); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("RecordLogin() error = %v, want ErrInvalidEvent", err)
	}
	if err := RecordRegister(context.Background(), nil, RegisterEvent{UserID: 1, Method: "email"}); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("RecordRegister() error = %v, want ErrInvalidEvent", err)
	}
}
