package authlimit

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestLoginLimitsBlockAndResetAccount(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()
	limits := LoginLimits("email", "User@Example.com", "203.0.113.10")

	for i := int64(0); i < accountFailureMax; i++ {
		if err := RecordFailure(ctx, client, limits); err != nil {
			t.Fatal(err)
		}
	}
	blocked, err := Blocked(ctx, client, limits)
	if err != nil || !blocked {
		t.Fatalf("Blocked() = %v, %v; want true, nil", blocked, err)
	}
	if err := ResetSuccessful(ctx, client, limits); err != nil {
		t.Fatal(err)
	}
	if client.Exists(ctx, limits[0].Key, limits[1].Key).Val() != 0 {
		t.Fatal("account and account/IP failure counters were not reset")
	}
	if client.Exists(ctx, limits[2].Key).Val() != 1 {
		t.Fatal("IP spray counter must survive a successful login")
	}
}

func TestLoginLimitKeysDoNotContainIdentity(t *testing.T) {
	limits := LoginLimits("email", "private@example.com", "203.0.113.10")
	for _, limit := range limits {
		if limit.Key == "" || containsAny(limit.Key, "private@example.com", "203.0.113.10") {
			t.Fatalf("limiter key contains private input: %q", limit.Key)
		}
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if len(needle) > 0 && len(value) >= len(needle) {
			for i := 0; i+len(needle) <= len(value); i++ {
				if value[i:i+len(needle)] == needle {
					return true
				}
			}
		}
	}
	return false
}
