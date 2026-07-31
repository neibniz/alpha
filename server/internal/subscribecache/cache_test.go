package subscribecache

import (
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestClearBySnapshotUsesTopologyOnlyForServerUserKeys(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&node.Node{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*node.Node{
		{Id: 10, Name: "node-a", ServerId: 101, Protocol: "vless", InboundAlias: "vless", Sort: 10},
		{Id: 11, Name: "node-b", ServerId: 102, Protocol: "vless", InboundAlias: "vless", Sort: 11},
		{Id: 12, Name: "node-c", ServerId: 103, Protocol: "vless", InboundAlias: "vless", Sort: 12},
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, serverID := range []int64{101, 102, 103} {
		if err := rds.Set(ctx, fmt.Sprintf("%s%d", node.ServerUserListCacheKey, serverID), "cached", 0).Err(); err != nil {
			t.Fatal(err)
		}
	}

	raw := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:  ordersnapshot.KindSubscribe,
		Name:  "Snapshot Plan",
		Nodes: []int64{10, 0, 10, 11},
	})
	if err := ClearBySnapshot(ctx, db, rds, raw); err != nil {
		t.Fatal(err)
	}

	if got := rds.Exists(ctx, fmt.Sprintf("%s%d", node.ServerUserListCacheKey, 101)).Val(); got != 0 {
		t.Fatalf("server 101 cache exists = %d, want deleted", got)
	}
	if got := rds.Exists(ctx, fmt.Sprintf("%s%d", node.ServerUserListCacheKey, 102)).Val(); got != 0 {
		t.Fatalf("server 102 cache exists = %d, want deleted", got)
	}
	if got := rds.Exists(ctx, fmt.Sprintf("%s%d", node.ServerUserListCacheKey, 103)).Val(); got != 1 {
		t.Fatalf("server 103 cache exists = %d, want retained", got)
	}
}
