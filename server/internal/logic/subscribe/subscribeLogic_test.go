package subscribe

import (
	"testing"

	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/xraytemplate"
)

func TestGetSubscribeProductMarksSnapshotProduct(t *testing.T) {
	raw := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:     ordersnapshot.KindSubscribe,
		Name:     "Snapshot Plan",
		Nodes:    []int64{10, 11},
		NodeTags: []string{"edge"},
	})

	product, fromSnapshot, err := (&SubscribeLogic{}).getSubscribeProduct(&modelUser.Subscribe{
		ProductSnapshot: raw,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !fromSnapshot {
		t.Fatal("snapshot product should be marked fromSnapshot")
	}
	if product == nil || product.Name != "Snapshot Plan" || product.Nodes != "10,11" || product.NodeTags != "edge" {
		t.Fatalf("snapshot product = %#v", product)
	}
}

func TestBuildXraySubscriptionProxiesSkipsEmptyServers(t *testing.T) {
	got := (&SubscribeLogic{}).buildXraySubscriptionProxies(nil, nil, nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("empty xray subscription proxies = %#v, want non-nil empty slice", got)
	}
}

func TestSubscribeInboundBindingIndexCacheUsesLinearPathForSmallInput(t *testing.T) {
	cache := newSubscribeInboundBindingIndexCache(smallSubscribeInboundBindingIndexCacheThreshold + 20)
	first := xraytemplate.NewInboundBindingIndex(nil)
	updated := xraytemplate.NewInboundBindingIndex(nil)
	cache.Set(2, first)
	cache.Set(4, xraytemplate.NewInboundBindingIndex(nil))
	cache.Set(2, updated)

	if cache.index != nil {
		t.Fatal("small subscribe inbound binding index cache should not allocate map")
	}
	got, ok := cache.Get(2)
	if !ok || got != updated {
		t.Fatalf("Get(2) = %#v, %v; want updated index", got, ok)
	}
	if _, ok := cache.Get(9); ok {
		t.Fatal("Get(9) found missing index")
	}
}

func TestSubscribeInboundBindingIndexCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := newSubscribeInboundBindingIndexCache(smallSubscribeInboundBindingIndexCacheThreshold + 20)
	for id := int64(1); id <= smallSubscribeInboundBindingIndexCacheThreshold+1; id++ {
		cache.Set(id, xraytemplate.NewInboundBindingIndex(nil))
	}
	if cache.index == nil {
		t.Fatal("large subscribe inbound binding index cache should allocate map")
	}
	lastID := int64(smallSubscribeInboundBindingIndexCacheThreshold + 1)
	if got, ok := cache.Get(lastID); !ok || got == nil {
		t.Fatalf("Get(%d) = %#v, %v", lastID, got, ok)
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("cache rows cap = %d, want <= %d", cap(cache.rows), maxCap)
	}
}

func TestSubscribeInboundBindingIndexCacheAvoidsMapForLargeDuplicateInput(t *testing.T) {
	cache := newSubscribeInboundBindingIndexCache(smallSubscribeInboundBindingIndexCacheThreshold + 20)
	updated := xraytemplate.NewInboundBindingIndex(nil)
	for i := 0; i < smallSubscribeInboundBindingIndexCacheThreshold+20; i++ {
		cache.Set(2, updated)
	}

	if cache.index != nil {
		t.Fatal("duplicate-heavy subscribe inbound binding index cache should not allocate map")
	}
	if len(cache.rows) != 1 {
		t.Fatalf("cache rows len = %d, want 1", len(cache.rows))
	}
	if cap(cache.rows) != smallSubscribeInboundBindingIndexCacheThreshold+1 {
		t.Fatalf("cache rows cap = %d, want %d", cap(cache.rows), smallSubscribeInboundBindingIndexCacheThreshold+1)
	}
	if got, ok := cache.Get(2); !ok || got != updated {
		t.Fatalf("Get(2) = %#v, %v; want updated index", got, ok)
	}
}
