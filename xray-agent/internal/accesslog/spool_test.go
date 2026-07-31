package accesslog

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/neibniz/alpha/xray-agent/internal/config"
	"github.com/neibniz/alpha/xray-agent/internal/panel"
)

func TestSpoolPersistsAndAcknowledgesEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.jsonl")
	statePath := filepath.Join(dir, "state.json")
	queue, err := openSpool(path, statePath, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	events := []panel.AccessEvent{
		{EventID: "00000000-0000-5000-8000-000000000001", Timestamp: 1, Status: "accepted"},
		{EventID: "00000000-0000-5000-8000-000000000002", Timestamp: 2, Status: "rejected"},
	}
	if err := queue.appendAndAdvance(events, 10, 20, 300, 1); err != nil {
		t.Fatal(err)
	}

	queue, err = openSpool(path, statePath, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if queue.state.Pending != 2 || queue.state.FileOffset != 300 || queue.state.ParseFailures != 1 {
		t.Fatalf("state = %+v", queue.state)
	}
	batch, offset, err := queue.readBatch(1, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch) != 1 || batch[0].EventID != events[0].EventID {
		t.Fatalf("batch = %+v", batch)
	}
	if err := queue.acknowledge(offset, len(batch)); err != nil {
		t.Fatal(err)
	}

	queue, err = openSpool(path, statePath, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if queue.state.Pending != 1 {
		t.Fatalf("pending = %d", queue.state.Pending)
	}
	batch, offset, err = queue.readBatch(10, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch) != 1 || batch[0].EventID != events[1].EventID {
		t.Fatalf("batch = %+v", batch)
	}
	if err := queue.acknowledge(offset, len(batch)); err != nil {
		t.Fatal(err)
	}
	if queue.state.Pending != 0 || queue.state.AckOffset != 0 {
		t.Fatalf("state = %+v", queue.state)
	}
}

func TestCollectorDrainsRotatedFileBeforeCurrentFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "xray-access.log")
	statePath := filepath.Join(dir, "access-state.json")
	spoolPath := filepath.Join(dir, "access-spool.jsonl")
	first := "2026/07/09 12:34:56 from tcp:203.0.113.1:1000 accepted tcp:first.example:443 [proxy] email: u_1\n"
	second := "2026/07/09 12:34:57 from tcp:203.0.113.2:1001 accepted tcp:second.example:443 [proxy] email: u_1\n"
	third := "2026/07/09 12:34:58 from tcp:203.0.113.3:1002 accepted tcp:third.example:443 [proxy] email: u_1\n"
	if err := os.WriteFile(logPath, []byte(first), 0o600); err != nil {
		t.Fatal(err)
	}
	collector, err := NewCollector(9, logPath, testAccessConfig(spoolPath, statePath), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := collector.collect(); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(second); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(logPath, logPath+".1"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte(third), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := collector.collect(); err != nil {
		t.Fatal(err)
	}

	events, _, err := collector.spool.readBatch(10, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("events = %d, want 3", len(events))
	}
	if events[0].DestinationHost != "first.example" || events[1].DestinationHost != "second.example" || events[2].DestinationHost != "third.example" {
		t.Fatalf("unexpected rotation order: %+v", events)
	}
}

func TestCollectorFinalizesTrailingLineAfterRotation(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "xray-access.log")
	statePath := filepath.Join(dir, "access-state.json")
	spoolPath := filepath.Join(dir, "access-spool.jsonl")
	line := "2026/07/09 12:34:56 from tcp:203.0.113.1:1000 accepted tcp:old.example:443 [proxy] email: u_1"
	if err := os.WriteFile(logPath, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	collector, err := NewCollector(9, logPath, testAccessConfig(spoolPath, statePath), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := collector.collect(); err != nil {
		t.Fatal(err)
	}
	if collector.spool.state.Pending != 0 {
		t.Fatalf("pending before rotation = %d", collector.spool.state.Pending)
	}
	if err := os.Rename(logPath, logPath+".1"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := collector.collect(); err != nil {
		t.Fatal(err)
	}
	if collector.spool.state.Pending != 1 {
		t.Fatalf("pending after rotation = %d", collector.spool.state.Pending)
	}
}

func testAccessConfig(spoolPath, statePath string) config.AccessReportConfig {
	return config.AccessReportConfig{
		BatchSize:     1000,
		MaxBatchBytes: 1 << 20,
		MaxSpoolBytes: 1 << 20,
		SpoolPath:     spoolPath,
		StatePath:     statePath,
	}
}

func TestSpoolFullDropsNewestAndAdvancesSource(t *testing.T) {
	dir := t.TempDir()
	queue, err := openSpool(filepath.Join(dir, "access.jsonl"), filepath.Join(dir, "state.json"), 1)
	if err != nil {
		t.Fatal(err)
	}
	err = queue.appendAndAdvance([]panel.AccessEvent{{EventID: "event", Status: "accepted"}}, 1, 2, 88, 0)
	if !errors.Is(err, errSpoolFull) {
		t.Fatalf("error = %v", err)
	}
	if queue.state.FileOffset != 88 || queue.state.Dropped != 1 || queue.state.Pending != 0 {
		t.Fatalf("state = %+v", queue.state)
	}
}
