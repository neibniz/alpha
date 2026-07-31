package server

import (
	"context"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/neibniz/alpha/server/internal/analytics"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/types"
)

func TestServerPushAccessLogsRejectsNilRequest(t *testing.T) {
	logic := NewServerPushAccessLogsLogic(context.Background(), nil)
	if _, err := logic.ServerPushAccessLogs(nil); err == nil {
		t.Fatal("nil access log request should be rejected")
	}
}

func TestEffectiveDeviceLimitUsesOrderSnapshot(t *testing.T) {
	raw := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:        ordersnapshot.KindSubscribe,
		Name:        "frozen plan",
		DeviceLimit: 3,
	})
	if got := effectiveDeviceLimit(raw, 9); got != 3 {
		t.Fatalf("effectiveDeviceLimit() = %d, want 3", got)
	}
	if got := effectiveDeviceLimit("{}", 9); got != 9 {
		t.Fatalf("fallback effectiveDeviceLimit() = %d, want 9", got)
	}
}

func TestNormalizeAccessEventsPreservesUnknownAcceptedSubscription(t *testing.T) {
	now := time.Now()
	items := []types.XrayAccessEvent{{
		EventId:         uuid.NewString(),
		Timestamp:       now.UnixMilli(),
		UserSubscribeId: 999,
		SourceIP:        "203.0.113.1",
		Status:          "accepted",
	}}
	events, rejected := normalizeAccessEvents(1, items, map[int64]int64{}, now)
	if len(events) != 1 || rejected != 0 {
		t.Fatalf("events = %v, rejected = %d", events, rejected)
	}
	if events[0].UserID != 0 || events[0].UserSubscribeID != 0 {
		t.Fatalf("unknown subscription should be stored without user association: %+v", events[0])
	}
}

func TestNormalizeAccessEventRejectsInvalidSourceAndFutureTimestamp(t *testing.T) {
	now := time.Now()
	base := types.XrayAccessEvent{
		EventId: uuid.NewString(), Timestamp: now.UnixMilli(), SourceIP: "not-an-ip", Status: "rejected",
	}
	if _, err := normalizeAccessEvent(7, base, nil, now); err == nil {
		t.Fatal("invalid source IP should be rejected")
	}
	base.SourceIP = "203.0.113.9"
	base.Timestamp = now.Add(maxAccessClockSkew + time.Second).UnixMilli()
	if _, err := normalizeAccessEvent(7, base, nil, now); err == nil {
		t.Fatal("future timestamp outside clock skew should be rejected")
	}
}

func TestNormalizeAccessEventCanonicalizesIPAndUTF8Truncation(t *testing.T) {
	now := time.Now()
	event, err := normalizeAccessEvent(7, types.XrayAccessEvent{
		EventId:         uuid.NewString(),
		Timestamp:       now.UnixMilli(),
		SourceIP:        "[2001:0db8::1]",
		DestinationHost: "198.51.100.8",
		Reason:          "原因原因原因",
		Status:          "rejected",
	}, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if event.SourceIP != "2001:db8::1" || event.DestinationHost != "" || event.DestinationIP != "198.51.100.8" {
		t.Fatalf("normalized event = %+v", event)
	}
	if got := truncateAccessValue("原因原因", 5); got != "原" || !utf8.ValidString(got) {
		t.Fatalf("UTF-8 truncation = %q", got)
	}
}

func TestNormalizeAccessEventPreservesPolicyInput(t *testing.T) {
	now := time.Now()
	event, err := normalizeAccessEvent(7, types.XrayAccessEvent{
		EventId:         uuid.NewString(),
		Timestamp:       now.UnixMilli(),
		UserSubscribeId: 12,
		SourceIP:        "2001:db8:1:2::9",
		Status:          "accepted",
	}, map[int64]int64{12: 4}, now)
	if err != nil {
		t.Fatal(err)
	}
	want := analytics.AccessEvent{ServerID: 7, UserID: 4, UserSubscribeID: 12, SourceIP: "2001:db8:1:2::9", Status: "accepted"}
	if event.ServerID != want.ServerID || event.UserID != want.UserID || event.UserSubscribeID != want.UserSubscribeID || event.SourceIP != want.SourceIP || event.Status != want.Status {
		t.Fatalf("normalized event = %+v", event)
	}
}
