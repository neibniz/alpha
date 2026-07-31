package subscriptionstate

import (
	"testing"
	"time"
)

func TestRenewalCapability(t *testing.T) {
	now := time.Now()
	for _, status := range []uint8{StatusActive, StatusExhausted, StatusExpired} {
		if got := RenewalCapability(status, now.Add(time.Hour), true, true, now); !got.Allowed {
			t.Fatalf("status %d should be renewable: %#v", status, got)
		}
	}
	if got := RenewalCapability(StatusPending, now.Add(time.Hour), true, true, now); got.Allowed || got.Reason != ReasonPending {
		t.Fatalf("pending subscription capability = %#v", got)
	}
	if got := RenewalCapability(StatusActive, now.Add(time.Hour), true, false, now); got.Allowed || got.Reason != ReasonRenewalDisabled {
		t.Fatalf("disabled renewal capability = %#v", got)
	}
}

func TestResetTrafficCapability(t *testing.T) {
	now := time.Now()
	for _, status := range []uint8{StatusActive, StatusExhausted} {
		if got := ResetTrafficCapability(status, now.Add(time.Hour), true, true, now); !got.Allowed {
			t.Fatalf("status %d should allow reset: %#v", status, got)
		}
	}
	if got := ResetTrafficCapability(StatusActive, now.Add(-time.Second), true, true, now); got.Allowed || got.Reason != ReasonExpired {
		t.Fatalf("expired subscription capability = %#v", got)
	}
	if got := ResetTrafficCapability(StatusStopped, now.Add(time.Hour), true, true, now); got.Allowed || got.Reason != ReasonStopped {
		t.Fatalf("stopped subscription capability = %#v", got)
	}
}
