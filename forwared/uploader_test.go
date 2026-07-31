package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestEventUploaderPersistsThenRemovesSuccessfulBatch(t *testing.T) {
	cfg := &Config{
		SpoolFile:      filepath.Join(t.TempDir(), "events.jsonl"),
		MaxSpoolBytes:  1 << 20,
		EventBatchSize: 2,
	}
	sent := 0
	uploader := NewEventUploader(cfg, func(_ context.Context, events []ConnectionEvent) error {
		sent += len(events)
		return nil
	})
	events := []ConnectionEvent{
		{EventID: "event-1", ConnectionID: "connection-1"},
		{EventID: "event-2", ConnectionID: "connection-2"},
	}
	for _, event := range events {
		if err := uploader.appendSpool(event); err != nil {
			t.Fatal(err)
		}
	}
	pending, err := uploader.readSpool()
	if err != nil {
		t.Fatal(err)
	}
	pending = uploader.flush(pending, false)
	if sent != 2 || len(pending) != 0 {
		t.Fatalf("unexpected upload result: sent=%d pending=%d", sent, len(pending))
	}
	reloaded, err := uploader.readSpool()
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded) != 0 {
		t.Fatalf("successful events remained in spool: %d", len(reloaded))
	}
}

func TestEventUploaderBacksOffAfterFailure(t *testing.T) {
	cfg := &Config{
		SpoolFile:      filepath.Join(t.TempDir(), "events.jsonl"),
		MaxSpoolBytes:  1 << 20,
		EventBatchSize: 1,
	}
	attempts := 0
	uploader := NewEventUploader(cfg, func(_ context.Context, _ []ConnectionEvent) error {
		attempts++
		return errors.New("unavailable")
	})
	pending := []ConnectionEvent{{EventID: "event-1"}}
	pending = uploader.flush(pending, false)
	pending = uploader.flush(pending, false)
	if attempts != 1 || len(pending) != 1 {
		t.Fatalf("backoff was not honored: attempts=%d pending=%d", attempts, len(pending))
	}
	if uploader.retryDelay < time.Second || !uploader.nextAttempt.After(time.Now()) {
		t.Fatalf("retry state was not scheduled: delay=%s next=%s", uploader.retryDelay, uploader.nextAttempt)
	}
}
