package server

import (
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/types"
)

func TestNormalizeServerNetworkHistoryRequest(t *testing.T) {
	now := time.UnixMilli(1_000_000)
	ids, start, end, interval, err := normalizeServerNetworkHistoryRequest(&types.QueryServerNetworkRateHistoryRequest{
		ServerIds: "7, 8,7",
	}, now)
	if err != nil {
		t.Fatalf("normalize request: %v", err)
	}
	if len(ids) != 2 || ids[0] != 7 || ids[1] != 8 {
		t.Fatalf("server ids = %#v, want [7 8]", ids)
	}
	if end != now.UnixMilli() || start != end-defaultServerNetworkHistoryRange.Milliseconds() {
		t.Fatalf("range = %d..%d, want default five-minute range", start, end)
	}
	if interval != 5*time.Second {
		t.Fatalf("interval = %v, want 5s", interval)
	}
}

func TestNormalizeServerNetworkHistoryRequestRejectsOversizedSeries(t *testing.T) {
	now := time.UnixMilli(100_000_000)
	_, _, _, _, err := normalizeServerNetworkHistoryRequest(&types.QueryServerNetworkRateHistoryRequest{
		ServerIds: "7",
		StartTime: now.Add(-24 * time.Hour).UnixMilli(),
		EndTime:   now.UnixMilli(),
		Interval:  "5s",
	}, now)
	if err == nil {
		t.Fatal("expected oversized five-second series to be rejected")
	}
}

func TestNormalizeServerNetworkHistoryRequestAcceptsSeventyTwoHourAggregate(t *testing.T) {
	now := time.UnixMilli(300_000_000)
	_, start, end, interval, err := normalizeServerNetworkHistoryRequest(&types.QueryServerNetworkRateHistoryRequest{
		ServerIds: "7",
		StartTime: now.Add(-72 * time.Hour).UnixMilli(),
		EndTime:   now.UnixMilli(),
		Interval:  "5m",
	}, now)
	if err != nil {
		t.Fatalf("normalize 72-hour request: %v", err)
	}
	if end-start != (72*time.Hour).Milliseconds() || interval != 5*time.Minute {
		t.Fatalf("normalized range = %d, interval = %v", end-start, interval)
	}
}
