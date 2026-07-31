package initialize

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/redis/go-redis/v9"
)

func TestPurgeSystemConfigCache(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	for _, key := range systemConfigCacheKeys {
		if err := client.Set(ctx, key, "cached", 0).Err(); err != nil {
			t.Fatalf("seed cache %q: %v", key, err)
		}
	}
	if err := client.Set(ctx, "unrelated", "keep", 0).Err(); err != nil {
		t.Fatalf("seed unrelated cache: %v", err)
	}

	if err := purgeSystemConfigCache(ctx, &svc.ServiceContext{Redis: client}); err != nil {
		t.Fatalf("purge system config cache: %v", err)
	}

	for _, key := range systemConfigCacheKeys {
		if mini.Exists(key) {
			t.Fatalf("cache key %q was not removed", key)
		}
	}
	if !mini.Exists("unrelated") {
		t.Fatal("unrelated cache key was removed")
	}
}
