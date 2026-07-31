package order

import (
	"testing"
	"time"

	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/subscriptionstate"
)

func TestSubscriptionActionPolicy(t *testing.T) {
	now := time.Now()
	enabled := true
	product := &modelSubscribe.Subscribe{AllowRenewal: &enabled, AllowResetTraffic: &enabled}
	expiredAt := now.Add(-time.Hour)
	if err := ensureRenewalActionAllowed(subscriptionstate.StatusExpired, expiredAt, product, now); err != nil {
		t.Fatalf("expired subscription should be renewable: %v", err)
	}
	if err := ensureResetTrafficActionAllowed(subscriptionstate.StatusExpired, expiredAt, product, now); err == nil {
		t.Fatal("expired subscription must not reset traffic")
	}
	if err := ensureResetTrafficActionAllowed(subscriptionstate.StatusExhausted, now.Add(time.Hour), product, now); err != nil {
		t.Fatalf("exhausted subscription should reset traffic: %v", err)
	}
}
