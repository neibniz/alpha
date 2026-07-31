package trafficreport

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/neibniz/alpha/xray-agent/internal/panel"
)

func TestSpoolPersistsReportsAndCursor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "traffic.jsonl")
	statePath := filepath.Join(dir, "traffic-state.json")
	queue, err := Open(path, statePath, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	first, err := queue.Append(panel.TrafficReport{
		SampledAt:       time.Now().UnixMilli(),
		IntervalSeconds: 5,
		Traffic:         []panel.UserTraffic{{SID: 7, Download: 20, Upload: 10}},
	}, map[string]int64{"user>>>u_7>>>traffic>>>downlink": 20})
	if err != nil {
		t.Fatal(err)
	}
	second, err := queue.Append(panel.TrafficReport{
		SampledAt:       time.Now().Add(5 * time.Second).UnixMilli(),
		IntervalSeconds: 5,
		Traffic:         []panel.UserTraffic{{SID: 7, Download: 30, Upload: 15}},
	}, map[string]int64{"user>>>u_7>>>traffic>>>downlink": 50})
	if err != nil {
		t.Fatal(err)
	}
	if first.ReportID == "" || first.Sequence != 1 || second.Sequence != 2 {
		t.Fatalf("report identities = %#v, %#v", first, second)
	}

	reports, nextOffset, err := queue.ReadBatch(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || reports[0].ReportID != first.ReportID {
		t.Fatalf("first batch = %#v", reports)
	}
	if err := queue.Acknowledge(nextOffset, len(reports)); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path, statePath, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Pending() != 1 {
		t.Fatalf("pending = %d, want 1", reopened.Pending())
	}
	if got := reopened.Checkpoint()["user>>>u_7>>>traffic>>>downlink"]; got != 50 {
		t.Fatalf("checkpoint = %d, want 50", got)
	}
	reports, _, err = reopened.ReadBatch(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || reports[0].ReportID != second.ReportID {
		t.Fatalf("remaining reports = %#v", reports)
	}
}

func TestSpoolDropsUncommittedTailOnOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "traffic.jsonl")
	statePath := filepath.Join(dir, "traffic-state.json")
	queue, err := Open(path, statePath, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queue.Append(panel.TrafficReport{
		SampledAt:       1,
		IntervalSeconds: 5,
		Traffic:         []panel.UserTraffic{{SID: 1, Download: 1}},
	}, map[string]int64{"cursor": 1}); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(`{"report_id":"uncommitted"}` + "\n"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path, statePath, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	reports, _, err := reopened.ReadBatch(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || reports[0].ReportID == "uncommitted" {
		t.Fatalf("reports after recovery = %#v", reports)
	}
}

func TestSpoolFlushPolicyUsesCountOrAge(t *testing.T) {
	dir := t.TempDir()
	queue, err := Open(filepath.Join(dir, "traffic.jsonl"), filepath.Join(dir, "state.json"), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queue.Append(panel.TrafficReport{
		SampledAt:       1,
		IntervalSeconds: 5,
		Traffic:         []panel.UserTraffic{{SID: 1, Download: 1}},
	}, map[string]int64{"cursor": 1}); err != nil {
		t.Fatal(err)
	}
	if queue.ShouldFlush(time.Now(), 2, time.Hour) {
		t.Fatal("one fresh report should not flush")
	}
	if !queue.ShouldFlush(time.Now().Add(2*time.Hour), 2, time.Hour) {
		t.Fatal("aged report should flush")
	}
	if _, err := queue.Append(panel.TrafficReport{
		SampledAt:       2,
		IntervalSeconds: 5,
		Traffic:         []panel.UserTraffic{{SID: 1, Download: 1}},
	}, map[string]int64{"cursor": 2}); err != nil {
		t.Fatal(err)
	}
	if !queue.ShouldFlush(time.Now(), 2, time.Hour) {
		t.Fatal("full batch should flush")
	}
}
