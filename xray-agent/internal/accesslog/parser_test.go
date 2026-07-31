package accesslog

import (
	"testing"
	"time"
)

func TestParseAcceptedAccessLine(t *testing.T) {
	line := "2026/07/09 12:34:56.123456 from tcp:203.0.113.42:54321 accepted tcp:example.com:443 [proxy] email: u_123"
	event, err := Parse(line)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	wantTime, _ := time.ParseInLocation("2006/01/02 15:04:05.999999999", "2026/07/09 12:34:56.123456", time.Local)
	if event.Timestamp != wantTime.UnixMilli() {
		t.Fatalf("Timestamp = %d, want %d", event.Timestamp, wantTime.UnixMilli())
	}
	if event.SourceIP != "203.0.113.42" || event.SourcePort != 54321 {
		t.Fatalf("source = %s:%d", event.SourceIP, event.SourcePort)
	}
	if event.Network != "tcp" || event.DestinationHost != "example.com" || event.DestinationPort != 443 {
		t.Fatalf("destination = %s:%s:%d", event.Network, event.DestinationHost, event.DestinationPort)
	}
	if event.Status != "accepted" || event.OutboundTag != "proxy" || event.UserSubscribeID != 123 {
		t.Fatalf("event = %+v", event)
	}
}

func TestParseRejectedAccessLineWithoutDestination(t *testing.T) {
	line := "2026/07/09 12:34:56 from tcp:[2001:db8::1]:3210 rejected  proxy/socks: unknown SOCKS version: 67 email: u_44"
	event, err := Parse(line)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if event.SourceIP != "2001:db8::1" || event.SourcePort != 3210 {
		t.Fatalf("source = %s:%d", event.SourceIP, event.SourcePort)
	}
	if event.DestinationHost != "" || event.DestinationIP != "" || event.DestinationPort != 0 {
		t.Fatalf("unexpected destination: %+v", event)
	}
	if event.Status != "rejected" || event.Reason != "proxy/socks: unknown SOCKS version: 67" || event.UserSubscribeID != 44 {
		t.Fatalf("event = %+v", event)
	}
}

func TestParseIPAddressDestination(t *testing.T) {
	event, err := Parse("2026/07/09 12:34:56 from udp:127.0.0.1:53 accepted udp:198.51.100.8:5353 [direct]")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if event.DestinationHost != "" || event.DestinationIP != "198.51.100.8" {
		t.Fatalf("destination host=%q ip=%q", event.DestinationHost, event.DestinationIP)
	}
}

func TestParseRejectsMalformedLine(t *testing.T) {
	if _, err := Parse("not an xray access record"); err == nil {
		t.Fatal("Parse() error = nil")
	}
}
