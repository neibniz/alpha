package server

import (
	"context"
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/analytics"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/google/uuid"
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

func TestNormalizeAccessEventsRejectsUnknownAcceptedSubscription(t *testing.T) {
	now := time.Now()
	items := []types.XrayAccessEvent{{
		EventId:         uuid.NewString(),
		Timestamp:       now.UnixMilli(),
		UserSubscribeId: 999,
		SourceIP:        "203.0.113.1",
		Status:          "accepted",
	}}
	events, rejected := normalizeAccessEvents(1, items, map[int64]int64{}, now)
	if len(events) != 0 || rejected != 1 {
		t.Fatalf("events = %v, rejected = %d", events, rejected)
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
