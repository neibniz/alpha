package accesspolicy

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/redis/go-redis/v9"
)

func TestStoreEnforcesAndRefreshesActiveNetworks(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	store := NewStore(client, privacy.DisabledCodec())
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	first, err := store.Evaluate(context.Background(), 41, "203.0.113.10", 2)
	if err != nil || first.Blocked || first.Active != 1 {
		t.Fatalf("first decision = %+v, err = %v", first, err)
	}
	second, err := store.Evaluate(context.Background(), 41, "203.0.113.11", 2)
	if err != nil || second.Blocked || second.Active != 2 {
		t.Fatalf("second decision = %+v, err = %v", second, err)
	}
	blocked, err := store.Evaluate(context.Background(), 41, "203.0.113.12", 2)
	if err != nil || !blocked.Blocked || blocked.Active != 2 {
		t.Fatalf("blocked decision = %+v, err = %v", blocked, err)
	}

	now = now.Add(time.Minute)
	refreshed, err := store.Evaluate(context.Background(), 41, "203.0.113.10", 2)
	if err != nil || refreshed.Blocked || refreshed.Active != 2 {
		t.Fatalf("refresh decision = %+v, err = %v", refreshed, err)
	}
	if !refreshed.ExpiresAt.Equal(now.Add(defaultActiveTTL)) {
		t.Fatalf("refreshed expiry = %v", refreshed.ExpiresAt)
	}
}

func TestStoreTreatsIPv6AddressesInSame64AsOneNetwork(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	store := NewStore(client, privacy.DisabledCodec())

	first, err := store.Evaluate(context.Background(), 42, "2001:db8:1:2::1", 1)
	if err != nil || first.Blocked {
		t.Fatalf("first decision = %+v, err = %v", first, err)
	}
	second, err := store.Evaluate(context.Background(), 42, "2001:db8:1:2::ffff", 1)
	if err != nil || second.Blocked || second.Active != 1 {
		t.Fatalf("same /64 decision = %+v, err = %v", second, err)
	}
	third, err := store.Evaluate(context.Background(), 42, "2001:db8:1:3::1", 1)
	if err != nil || !third.Blocked {
		t.Fatalf("different /64 decision = %+v, err = %v", third, err)
	}
}

func TestStoreDoesNotKeepPlaintextNetworks(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	store := NewStore(client, privacy.DisabledCodec())

	if _, err := store.Evaluate(context.Background(), 43, "203.0.113.77", 1); err != nil {
		t.Fatal(err)
	}
	members, err := client.ZRange(context.Background(), "access:ip-limit:43", 0, -1).Result()
	if err != nil || len(members) != 1 {
		t.Fatalf("members = %v, err = %v", members, err)
	}
	if members[0] == "203.0.113.77/32" {
		t.Fatal("Redis member contains plaintext source network")
	}
}

func TestStoreAcceptsNormalizedAccessLogScopes(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewStore(client, privacy.DisabledCodec())
	ctx := context.Background()
	for _, address := range []string{"203.0.113.10", "2001:db8:1:2::1"} {
		t.Run(address, func(t *testing.T) {
			network, ok := privacy.NormalizeIPNetwork(address)
			if !ok {
				t.Fatal("invalid test address")
			}
			first, err := store.Evaluate(ctx, 44, network, 2)
			if err != nil || first.Blocked {
				t.Fatalf("normalized scope decision = %+v, err = %v", first, err)
			}
			refreshed, err := store.Evaluate(ctx, 44, address, 2)
			if err != nil || refreshed.Blocked || refreshed.Active != first.Active {
				t.Fatalf("raw address and scope must share a member: %+v, err = %v", refreshed, err)
			}
		})
	}
}
