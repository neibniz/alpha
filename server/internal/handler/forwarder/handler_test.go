package forwarder

import (
	"bytes"
	"compress/gzip"
	"net"
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/forwarderapi"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/google/uuid"
)

func TestParseConnectionEventAcceptsClosedConnection(t *testing.T) {
	now := time.Now().UTC()
	event, err := parseConnectionEvent(uuid.New(), forwarderapi.ConnectionEvent{
		EventID:              uuid.NewString(),
		ConnectionID:         uuid.NewString(),
		BootID:               uuid.NewString(),
		EventTime:            now.UnixMilli(),
		EventType:            "closed",
		SourceIP:             "203.0.113.10",
		SourcePort:           54321,
		ListenPort:           8443,
		ConfiguredTargetHost: "198.51.100.8",
		ConfiguredTargetPort: 443,
		ResolvedTargetIP:     "198.51.100.8",
		ResolvedTargetPort:   443,
		RuleID:               "rule-1",
		ConnectedAt:          now.Add(-time.Second).UnixMilli(),
		ClosedAt:             now.UnixMilli(),
		DurationMS:           1000,
		UploadBytes:          1024,
		DownloadBytes:        2048,
		CloseInitiator:       "client",
		CloseReason:          "eof",
	})
	if err != nil {
		t.Fatal(err)
	}
	if event.EventType != "closed" || event.UploadBytes != 1024 || event.DownloadBytes != 2048 {
		t.Fatalf("unexpected event: %#v", event)
	}
}

func TestParseConnectionEventRejectsFutureEvent(t *testing.T) {
	_, err := parseConnectionEvent(uuid.New(), forwarderapi.ConnectionEvent{
		EventID:              uuid.NewString(),
		ConnectionID:         uuid.NewString(),
		BootID:               uuid.NewString(),
		EventTime:            time.Now().Add(time.Hour).UnixMilli(),
		EventType:            "blocked",
		SourceIP:             "203.0.113.10",
		SourcePort:           54321,
		ListenPort:           8443,
		ConfiguredTargetHost: "198.51.100.8",
		ConfiguredTargetPort: 443,
		RuleID:               "rule-1",
	})
	if err == nil {
		t.Fatal("expected future event to be rejected")
	}
}

func TestNormalizeRuleStatusesRejectsDuplicateRule(t *testing.T) {
	rule := realtime.ForwarderRuleStatus{
		RuleID: "rule-1", ListenPort: 8443, TargetHost: "198.51.100.8", TargetPort: 443,
	}
	if _, err := normalizeRuleStatuses([]realtime.ForwarderRuleStatus{rule, rule}); err == nil {
		t.Fatal("expected duplicate rule to be rejected")
	}
}

func TestReadEventsRequestBodyGzip(t *testing.T) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte(`{"events":[]}`)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	payload, err := readEventsRequestBody(&compressed, "gzip")
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"events":[]}` {
		t.Fatalf("unexpected payload: %s", payload)
	}
}

func TestRegistrationPublicIPPrefersObservedPublicAddress(t *testing.T) {
	observed := "8.8.8.8"
	if net.ParseIP(observed) == nil {
		t.Fatal("invalid test address")
	}
	value, err := registrationPublicIP("1.1.1.1", observed)
	if err != nil {
		t.Fatal(err)
	}
	if value != observed {
		t.Fatalf("expected %s, got %s", observed, value)
	}
}

func TestRegistrationPublicIPAllowsReportedAddressBehindPrivateProxy(t *testing.T) {
	value, err := registrationPublicIP("1.1.1.1", "10.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if value != "1.1.1.1" {
		t.Fatalf("unexpected address: %s", value)
	}
}
