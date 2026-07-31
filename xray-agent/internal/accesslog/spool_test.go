package accesslog

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	records := []collectedRecord{
		{Event: events[0], EndOffset: 100},
		{Event: events[1], EndOffset: 200},
		{EndOffset: 300, ParseFailure: true},
	}
	if _, err := queue.appendRecords(records, 10, 20, 0); err != nil {
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

func TestSpoolFullPausesBeforeUnqueuedSourceRecord(t *testing.T) {
	dir := t.TempDir()
	queue, err := openSpool(filepath.Join(dir, "access.jsonl"), filepath.Join(dir, "state.json"), 1)
	if err != nil {
		t.Fatal(err)
	}
	committed, err := queue.appendRecords([]collectedRecord{{
		Event:     panel.AccessEvent{EventID: "event", Status: "accepted"},
		EndOffset: 88,
	}}, 1, 2, 0)
	if !errors.Is(err, errSpoolFull) {
		t.Fatalf("error = %v", err)
	}
	if committed != 0 || queue.state.FileOffset != 0 || queue.state.Dropped != 0 || queue.state.Pending != 0 {
		t.Fatalf("state = %+v", queue.state)
	}
}

func TestSpoolFullCommitsOnlyDurablePrefix(t *testing.T) {
	dir := t.TempDir()
	first := panel.AccessEvent{EventID: "first", Status: "accepted"}
	second := panel.AccessEvent{EventID: "second", Status: "accepted"}
	payload, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	secondPayload, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	maxPayload := max(len(payload), len(secondPayload)) + 1
	queue, err := openSpool(filepath.Join(dir, "access.jsonl"), filepath.Join(dir, "state.json"), int64(maxPayload))
	if err != nil {
		t.Fatal(err)
	}
	records := []collectedRecord{{Event: first, EndOffset: 10}, {Event: second, EndOffset: 20}}
	committed, err := queue.appendRecords(records, 1, 2, 0)
	if !errors.Is(err, errSpoolFull) || committed != 1 {
		t.Fatalf("committed=%d error=%v", committed, err)
	}
	if queue.state.FileOffset != 10 || queue.state.Pending != 1 || queue.state.Dropped != 0 {
		t.Fatalf("state = %+v", queue.state)
	}

	batch, nextOffset, err := queue.readBatch(10, 1<<20)
	if err != nil || len(batch) != 1 || batch[0].EventID != "first" {
		t.Fatalf("batch=%+v offset=%d error=%v", batch, nextOffset, err)
	}
	if err := queue.acknowledge(nextOffset, 1); err != nil {
		t.Fatal(err)
	}
	committed, err = queue.appendRecords(records[1:], 1, 2, 10)
	if err != nil || committed != 1 || queue.state.FileOffset != 20 || queue.state.Pending != 1 {
		t.Fatalf("committed=%d state=%+v error=%v", committed, queue.state, err)
	}
}

func TestSpoolFullDoesNotCrossUnqueuedEventAfterParseFailure(t *testing.T) {
	dir := t.TempDir()
	queue, err := openSpool(filepath.Join(dir, "access.jsonl"), filepath.Join(dir, "state.json"), 1)
	if err != nil {
		t.Fatal(err)
	}
	committed, err := queue.appendRecords([]collectedRecord{
		{EndOffset: 10, ParseFailure: true},
		{Event: panel.AccessEvent{EventID: "event", Status: "accepted"}, EndOffset: 20},
	}, 1, 2, 0)
	if !errors.Is(err, errSpoolFull) || committed != 1 {
		t.Fatalf("committed=%d error=%v", committed, err)
	}
	if queue.state.FileOffset != 10 || queue.state.ParseFailures != 1 || queue.state.Pending != 0 {
		t.Fatalf("state = %+v", queue.state)
	}
}

func TestOpenSpoolRejectsCorruptState(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	if err := os.WriteFile(statePath, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := openSpool(filepath.Join(dir, "access.jsonl"), statePath, 1<<20)
	if err == nil || !strings.Contains(err.Error(), "decode access spool state") {
		t.Fatalf("error = %v", err)
	}
}

func TestOpenSpoolRejectsCorruptRecord(t *testing.T) {
	dir := t.TempDir()
	spoolPath := filepath.Join(dir, "access.jsonl")
	if err := os.WriteFile(spoolPath, []byte("not-json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := openSpool(spoolPath, filepath.Join(dir, "state.json"), 1<<20)
	if err == nil || !strings.Contains(err.Error(), "decode access spool record") {
		t.Fatalf("error = %v", err)
	}
}

func TestCollectorBoundsOversizedAccessLine(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "xray-access.log")
	oversized := strings.Repeat("x", maxAccessLogLineBytes+1) + "\n"
	valid := "2026/07/09 12:34:56 from tcp:203.0.113.1:1000 accepted tcp:valid.example:443 [proxy] email: u_1\n"
	if err := os.WriteFile(logPath, []byte(oversized+valid), 0o600); err != nil {
		t.Fatal(err)
	}
	collector, err := NewCollector(9, logPath, testAccessConfig(
		filepath.Join(dir, "access.jsonl"), filepath.Join(dir, "state.json"),
	), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := collector.collect(); err != nil {
		t.Fatal(err)
	}
	if collector.spool.state.ParseFailures != 1 || collector.spool.state.Pending != 1 {
		t.Fatalf("state = %+v", collector.spool.state)
	}
	if collector.spool.state.FileOffset != int64(len(oversized)+len(valid)) {
		t.Fatalf("file offset = %d", collector.spool.state.FileOffset)
	}
}

func TestCollectorAdvancesOversizedIncompleteLineIncrementally(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "xray-access.log")
	oversized := strings.Repeat("x", maxAccessLogLineBytes+(16<<10))
	if err := os.WriteFile(logPath, []byte(oversized), 0o600); err != nil {
		t.Fatal(err)
	}
	collector, err := NewCollector(9, logPath, testAccessConfig(
		filepath.Join(dir, "access.jsonl"), filepath.Join(dir, "state.json"),
	), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := collector.collect(); err != nil {
		t.Fatal(err)
	}
	firstOffset := collector.spool.state.FileOffset
	if firstOffset <= 0 || firstOffset >= int64(len(oversized)) || !collector.spool.state.DiscardingLine {
		t.Fatalf("state after first chunk = %+v", collector.spool.state)
	}
	if collector.spool.state.ParseFailures != 1 {
		t.Fatalf("parse failures = %d", collector.spool.state.ParseFailures)
	}
	if err := collector.collect(); err != nil {
		t.Fatal(err)
	}
	if collector.spool.state.FileOffset <= firstOffset || collector.spool.state.ParseFailures != 1 {
		t.Fatalf("state after second chunk = %+v", collector.spool.state)
	}

	valid := "\n2026/07/09 12:34:56 from tcp:203.0.113.1:1000 accepted tcp:valid.example:443 [proxy] email: u_1\n"
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(valid); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := collector.collect(); err != nil {
		t.Fatal(err)
	}
	if collector.spool.state.DiscardingLine || collector.spool.state.ParseFailures != 1 || collector.spool.state.Pending != 1 {
		t.Fatalf("final state = %+v", collector.spool.state)
	}
}
