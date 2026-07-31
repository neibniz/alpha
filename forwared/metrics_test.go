package main

import (
	"testing"
	"time"
)

func TestMetricsSnapshotIncludesPerRuleCountersAndRates(t *testing.T) {
	metrics := NewMetrics()
	item := ForwaredItem{
		RuleID: "rule-1", LocalPort: 1001, TargetHost: "10.0.0.1", TargetPort: 443,
	}
	metrics.Accepted(item.RuleID)
	metrics.Connected(item.RuleID)
	metrics.AddUpload(item.RuleID, 2048)
	metrics.AddDownload(item.RuleID, 4096)
	metrics.mu.Lock()
	metrics.lastAt = time.Now().Add(-time.Second)
	metrics.mu.Unlock()

	snapshot := metrics.Snapshot([]ForwaredItem{item})
	if snapshot.ActiveConnections != 1 || snapshot.UploadBytes != 2048 || snapshot.DownloadBytes != 4096 {
		t.Fatalf("unexpected global metrics: %#v", snapshot)
	}
	if snapshot.UploadBps < 1900 || snapshot.DownloadBps < 3900 {
		t.Fatalf("unexpected rates: upload=%d download=%d", snapshot.UploadBps, snapshot.DownloadBps)
	}
	if len(snapshot.Rules) != 1 || snapshot.Rules[0].AcceptedConnections != 1 {
		t.Fatalf("unexpected rule metrics: %#v", snapshot.Rules)
	}
}
