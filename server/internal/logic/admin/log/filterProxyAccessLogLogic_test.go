package log

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/neibniz/alpha/server/internal/analytics"
	"github.com/neibniz/alpha/server/internal/types"
)

func TestProxyAccessFilterDefaultsAndCursor(t *testing.T) {
	now := time.Date(2026, 7, 9, 12, 0, 0, 987654321, time.Local)
	normalizedNow := time.UnixMilli(now.UnixMilli())
	eventID := uuid.MustParse("00000000-0000-5000-8000-000000000001")
	cursor := encodeProxyAccessCursor(normalizedNow.Add(-time.Hour), eventID)
	filter, err := proxyAccessFilter(&types.FilterProxyAccessLogRequest{Cursor: cursor, Size: 500}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !filter.Start.Equal(normalizedNow.Add(-24*time.Hour)) || !filter.End.Equal(normalizedNow) {
		t.Fatalf("range = %s to %s", filter.Start, filter.End)
	}
	if filter.Limit != 200 || filter.CursorID != eventID || !filter.CursorTime.Equal(normalizedNow.Add(-time.Hour)) {
		t.Fatalf("filter = %+v", filter)
	}
}

func TestProxyAccessFilterRejectsOversizedRange(t *testing.T) {
	now := time.Now()
	_, err := proxyAccessFilter(&types.FilterProxyAccessLogRequest{
		StartTime: now.Add(-32 * 24 * time.Hour).UnixMilli(),
		EndTime:   now.UnixMilli(),
	}, now)
	if err == nil {
		t.Fatal("proxyAccessFilter() error = nil")
	}
}

func TestProxyAccessFilterNormalizesNetworkIdentifiers(t *testing.T) {
	now := time.Now()
	filter, err := proxyAccessFilter(&types.FilterProxyAccessLogRequest{
		SourceIP:    "[2001:0db8::1]",
		Destination: "2001:0db8::2",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if filter.SourceIP != "2001:db8::1" || filter.Destination != "2001:db8::2" {
		t.Fatalf("filter = %+v", filter)
	}
	if _, err := proxyAccessFilter(&types.FilterProxyAccessLogRequest{SourceIP: "invalid"}, now); err == nil {
		t.Fatal("invalid source IP should be rejected")
	}
	if _, err := proxyAccessFilter(&types.FilterProxyAccessLogRequest{ServerId: -1}, now); err == nil {
		t.Fatal("negative filter ID should be rejected")
	}
}

func TestProxyAccessLogViewMasksNetworkIdentifiers(t *testing.T) {
	view := proxyAccessLogView(analytics.AccessEvent{
		EventID:         uuid.MustParse("00000000-0000-5000-8000-000000000001"),
		EventTime:       time.UnixMilli(1234),
		SourceIP:        "203.0.113.42",
		DestinationHost: "private.example.com",
		Reason:          "connection refused for user@example.com",
	})
	if view.SourceIP != "203.0.113.*" {
		t.Fatalf("source_ip = %q", view.SourceIP)
	}
	if view.Destination == "private.example.com" || view.Reason == "connection refused for user@example.com" {
		t.Fatalf("unmasked view = %+v", view)
	}
}

func TestProxyAccessLogViewReturnsAuditedRevealedFields(t *testing.T) {
	view := proxyAccessLogView(analytics.AccessEvent{
		EventID:         uuid.MustParse("00000000-0000-5000-8000-000000000002"),
		EventTime:       time.UnixMilli(1234),
		SourceIP:        "203.0.113.42",
		DestinationHost: "private.example.com",
		Reason:          "connection refused for user@example.com",
		Revealed:        true,
	})
	if view.SourceIP != "203.0.113.42" || view.Destination != "private.example.com" || view.Reason != "connection refused for user@example.com" || !view.Revealed {
		t.Fatalf("revealed view = %+v", view)
	}
}
