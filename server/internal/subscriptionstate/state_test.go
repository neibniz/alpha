package subscriptionstate

import (
	"math"
	"testing"
	"time"
)

func TestTrafficExhaustedAvoidsOverflow(t *testing.T) {
	if !TrafficExhausted(math.MaxInt64, math.MaxInt64-1, math.MaxInt64) {
		t.Fatal("large non-negative counters must be treated as exhausted")
	}
	if TrafficExhausted(0, math.MaxInt64, math.MaxInt64) {
		t.Fatal("zero traffic limit must remain unlimited")
	}
}

func TestEffectiveStatusPrioritizesExpiry(t *testing.T) {
	now := time.Now()
	status, reason := EffectiveStatus(StatusActive, now, 100, 100, 0, now)
	if status != StatusExpired || reason != ReasonExpired {
		t.Fatalf("effective status = %d/%s, want expired", status, reason)
	}
	status, reason = EffectiveStatus(StatusExhausted, now, 100, 100, 0, now)
	if status != StatusExpired || reason != ReasonExpired {
		t.Fatalf("exhausted effective status = %d/%s, want expired", status, reason)
	}
}

func TestUnlimitedExpirySupportsLegacyEpoch(t *testing.T) {
	if !IsUnlimitedExpiry(time.UnixMilli(0)) || !IsUnlimitedExpiry(time.Time{}) {
		t.Fatal("zero time and Unix epoch must both be unlimited")
	}
}

func TestRemainingTrafficSaturates(t *testing.T) {
	if got := UsedTraffic(math.MaxInt64, 1); got != math.MaxInt64 {
		t.Fatalf("used traffic = %d, want MaxInt64", got)
	}
	if got := RemainingTraffic(100, 30, 20); got != 50 {
		t.Fatalf("remaining traffic = %d, want 50", got)
	}
}
