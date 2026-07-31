package subscribeview

import (
	"encoding/json"
	"testing"
	"time"

	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestSubscribeFromModelMapsSharedAPIShape(t *testing.T) {
	enabled := true
	createdAt := time.UnixMilli(1716192000000)
	updatedAt := time.UnixMilli(1716195600000)
	got, err := SubscribeFromModel(&modelSubscribe.Subscribe{
		Id:                9,
		Name:              "Pro",
		Language:          "en",
		Description:       "desc",
		UnitPrice:         100,
		UnitTime:          "Month",
		Discount:          `[{"quantity":3,"discount":0.8}]`,
		Replacement:       10,
		Inventory:         20,
		Traffic:           30,
		SpeedLimit:        40,
		DeviceLimit:       2,
		Quota:             5,
		Nodes:             "1,2,2,3",
		NodeTags:          " edge, premium, edge, ",
		Show:              &enabled,
		UserVisible:       &enabled,
		Sell:              &enabled,
		AllowDeduction:    &enabled,
		AllowRenewal:      &enabled,
		AllowResetTraffic: &enabled,
		RenewalReset:      &enabled,
		ShowOriginalPrice: true,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}, SubscribeOptions{IncludeTopology: true})
	if err != nil {
		t.Fatalf("SubscribeFromModel returned error: %v", err)
	}
	if got.Id != 9 || got.UnitPrice != 100 || got.DeviceLimit != 2 {
		t.Fatalf("subscribe fields mismatch: %+v", got)
	}
	if len(got.Discount) != 1 || got.Discount[0].Quantity != 3 || got.Discount[0].Discount != 0.8 {
		t.Fatalf("discount mismatch: %+v", got.Discount)
	}
	if len(got.Nodes) != 3 || got.Nodes[0] != 1 || got.Nodes[1] != 2 || got.Nodes[2] != 3 || got.NodeTags[0] != "edge" || got.NodeTags[1] != "premium" {
		t.Fatalf("topology mismatch: nodes=%+v tags=%+v", got.Nodes, got.NodeTags)
	}
	if !got.Show || !got.UserVisible || !got.AllowResetTraffic || !got.ShowOriginalPrice {
		t.Fatalf("boolean fields mismatch: %+v", got)
	}
	if got.CreatedAt != createdAt.UnixMilli() || got.UpdatedAt != updatedAt.UnixMilli() {
		t.Fatalf("timestamps mismatch: %+v", got)
	}
}

func TestSubscribeFromModelLenientDropsInvalidDiscount(t *testing.T) {
	got := SubscribeFromModelLenient(&modelSubscribe.Subscribe{Discount: "{bad"}, SubscribeOptions{})
	if got.Discount == nil || len(got.Discount) != 0 {
		t.Fatalf("invalid discount should be an empty array in lenient mode: %+v", got.Discount)
	}
	if _, err := SubscribeFromModel(&modelSubscribe.Subscribe{Discount: "{bad"}, SubscribeOptions{}); err == nil {
		t.Fatal("strict subscribe mapper should return invalid discount error")
	}
}

func TestParseDiscountFastPathsEmptyValues(t *testing.T) {
	for _, raw := range []string{"", "[]", " [] ", "null", " null "} {
		got, err := ParseDiscount(raw)
		if err != nil {
			t.Fatalf("ParseDiscount(%q) returned error: %v", raw, err)
		}
		if got == nil || len(got) != 0 {
			t.Fatalf("ParseDiscount(%q) should return non-nil empty discounts: %#v", raw, got)
		}
	}
}

func TestSubscribeRowsLenientSkipsNilAndKeepsShape(t *testing.T) {
	enabled := true
	rows := []*modelSubscribe.Subscribe{
		{
			Id:          10,
			Name:        "Portal Plan",
			Discount:    `[{"quantity":2,"discount":0.9}]`,
			Nodes:       "1,2",
			NodeTags:    "edge",
			Show:        &enabled,
			UserVisible: &enabled,
			Sell:        &enabled,
		},
		nil,
	}

	got := SubscribeRowsLenient(rows, SubscribeOptions{})
	if len(got) != 1 {
		t.Fatalf("subscribe rows = %d, want 1: %#v", len(got), got)
	}
	if got[0].Id != 10 || got[0].Name != "Portal Plan" || !got[0].Show || !got[0].UserVisible || !got[0].Sell {
		t.Fatalf("subscribe row not mapped: %#v", got[0])
	}
	if len(got[0].Discount) != 1 || got[0].Discount[0].Quantity != 2 {
		t.Fatalf("discount not parsed: %#v", got[0].Discount)
	}
	if len(got[0].Nodes) != 0 || len(got[0].NodeTags) != 0 {
		t.Fatalf("topology should be omitted unless requested: nodes=%#v tags=%#v", got[0].Nodes, got[0].NodeTags)
	}
	if got[0].Nodes == nil || got[0].NodeTags == nil {
		t.Fatalf("empty topology should keep array shape: nodes=%#v tags=%#v", got[0].Nodes, got[0].NodeTags)
	}
}

func TestSubscribeRowsLenientSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*modelSubscribe.Subscribe, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &modelSubscribe.Subscribe{Id: 1, Name: "first"}
	rows[len(rows)-1] = &modelSubscribe.Subscribe{Id: 2, Name: "second"}

	got := SubscribeRowsLenient(rows, SubscribeOptions{})
	if len(got) != 2 {
		t.Fatalf("subscribe rows = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy subscribe rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("subscribe row order changed: %#v", got)
	}
}

func TestSubscribeRowsLenientUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*modelSubscribe.Subscribe, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &modelSubscribe.Subscribe{Id: int64(index + 1), Name: "subscribe"}
	}

	got := SubscribeRowsLenient(rows, SubscribeOptions{})
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("subscribe rows = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("subscribe rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("subscribe row order changed: %#v", got)
	}
}

func TestSubscribeRowsLenientCanIncludeTopology(t *testing.T) {
	got := SubscribeRowsLenient([]*modelSubscribe.Subscribe{{
		Id:       11,
		Nodes:    "1,2,2",
		NodeTags: "edge, edge, vip",
	}}, SubscribeOptions{IncludeTopology: true})
	if len(got) != 1 {
		t.Fatalf("subscribe rows = %d, want 1: %#v", len(got), got)
	}
	if len(got[0].Nodes) != 2 || got[0].Nodes[0] != 1 || got[0].Nodes[1] != 2 {
		t.Fatalf("nodes not normalized: %#v", got[0].Nodes)
	}
	if len(got[0].NodeTags) != 2 || got[0].NodeTags[0] != "edge" || got[0].NodeTags[1] != "vip" {
		t.Fatalf("node tags not normalized: %#v", got[0].NodeTags)
	}
}

func TestSubscribeRowsLenientReusesDuplicateParsedDiscountAndTopology(t *testing.T) {
	rawDiscount := `[{"quantity":2,"discount":0.9}]`
	rows := []*modelSubscribe.Subscribe{
		{Id: 1, Discount: rawDiscount, Nodes: "1,2,2", NodeTags: "edge, edge, vip"},
		{Id: 2, Discount: rawDiscount, Nodes: "1,2,2", NodeTags: "edge, edge, vip"},
	}

	got := SubscribeRowsLenient(rows, SubscribeOptions{IncludeTopology: true})
	if len(got) != 2 {
		t.Fatalf("subscribe rows = %d, want 2: %#v", len(got), got)
	}
	if len(got[0].Discount) != 1 || len(got[1].Discount) != 1 {
		t.Fatalf("discount rows not parsed: %#v", got)
	}
	if &got[0].Discount[0] != &got[1].Discount[0] {
		t.Fatalf("duplicate discount JSON should reuse parsed slice storage")
	}
	if len(got[0].Nodes) != 2 || len(got[1].Nodes) != 2 || &got[0].Nodes[0] != &got[1].Nodes[0] {
		t.Fatalf("duplicate node selectors should reuse parsed node storage: %#v %#v", got[0].Nodes, got[1].Nodes)
	}
	if len(got[0].NodeTags) != 2 || len(got[1].NodeTags) != 2 || &got[0].NodeTags[0] != &got[1].NodeTags[0] {
		t.Fatalf("duplicate node tag selectors should reuse parsed tag storage: %#v %#v", got[0].NodeTags, got[1].NodeTags)
	}
}

func TestDiscountParseCacheCachesValidAndInvalidValues(t *testing.T) {
	cache := newDiscountParseCache(2)
	raw := `[{"quantity":2,"discount":0.9}]`
	first, err := cache.Parse(raw)
	if err != nil || len(first) != 1 {
		t.Fatalf("first Parse() = %#v, %v", first, err)
	}
	second, err := cache.Parse(raw)
	if err != nil || len(second) != 1 {
		t.Fatalf("second Parse() = %#v, %v", second, err)
	}
	if len(cache.rows) != 1 || cache.index != nil {
		t.Fatalf("small discount cache rows/index = %d/%#v, want one row and no map", len(cache.rows), cache.index)
	}
	if _, err := cache.Parse("{bad"); err == nil {
		t.Fatal("invalid discount should return error")
	}
	if _, err := cache.Parse("{bad"); err == nil {
		t.Fatal("invalid discount should stay invalid on cache hit")
	}
	if len(cache.rows) != 2 {
		t.Fatalf("discount cache rows = %d, want 2", len(cache.rows))
	}
}

func TestSubscribeViewCachesUpgradeLargeInputsToMap(t *testing.T) {
	discountCache := newDiscountParseCache(smallSubscribeViewCacheLimit + 1)
	if discountCache == nil || discountCache.rows != nil || discountCache.index != nil {
		t.Fatalf("large discount cache should not preallocate storage from hint alone: %#v", discountCache)
	}
	topologyCache := newTopologyParseCache(smallSubscribeViewCacheLimit+1, true)
	if topologyCache == nil || topologyCache.rows != nil || topologyCache.index != nil {
		t.Fatalf("large topology cache should not preallocate storage from hint alone: %#v", topologyCache)
	}

	rawDiscount := `[{"quantity":2,"discount":0.9}]`
	firstDiscount, err := discountCache.Parse(rawDiscount)
	if err != nil {
		t.Fatal(err)
	}
	secondDiscount, err := discountCache.Parse(rawDiscount)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstDiscount) != len(secondDiscount) || len(discountCache.rows) != 1 || discountCache.index != nil {
		t.Fatalf("duplicate discounts should stay on linear path: rows=%#v index=%#v", discountCache.rows, discountCache.index)
	}
	firstNodes, firstTags := topologyCache.Parse("1,2,2", "edge, edge")
	secondNodes, secondTags := topologyCache.Parse("1,2,2", "edge, edge")
	if len(firstNodes) != len(secondNodes) || len(firstTags) != len(secondTags) || len(topologyCache.rows) != 1 || topologyCache.index != nil {
		t.Fatalf("duplicate topology should stay on linear path: rows=%#v index=%#v", topologyCache.rows, topologyCache.index)
	}

	discountCache = newDiscountParseCache(smallSubscribeViewCacheLimit + 1)
	topologyCache = newTopologyParseCache(smallSubscribeViewCacheLimit+1, true)
	for i := 0; i < smallSubscribeViewCacheLimit+1; i++ {
		rawDiscount := `[{"quantity":` + string(rune('1'+i)) + `,"discount":0.9}]`
		if _, err := discountCache.Parse(rawDiscount); err != nil {
			t.Fatalf("discount Parse(%d): %v", i, err)
		}
		topologyCache.Parse(string(rune('1'+i)), "edge")
	}
	if len(discountCache.rows) != smallSubscribeViewCacheLimit+1 || len(discountCache.index) != smallSubscribeViewCacheLimit+1 {
		t.Fatalf("discount cache rows/index = %d/%d", len(discountCache.rows), len(discountCache.index))
	}
	if len(topologyCache.rows) != smallSubscribeViewCacheLimit+1 || len(topologyCache.index) != smallSubscribeViewCacheLimit+1 {
		t.Fatalf("topology cache rows/index = %d/%d", len(topologyCache.rows), len(topologyCache.index))
	}
}

func TestSubscribeViewCachesBoundSparseUpgradeCapacity(t *testing.T) {
	discountCache := newDiscountParseCache(smallSubscribeViewCacheLimit + 20)
	topologyCache := newTopologyParseCache(smallSubscribeViewCacheLimit+20, true)
	for i := 0; i < smallSubscribeViewCacheLimit+1; i++ {
		rawDiscount := `[{"quantity":` + string(rune('1'+i)) + `,"discount":0.9}]`
		if _, err := discountCache.Parse(rawDiscount); err != nil {
			t.Fatalf("discount Parse(%d): %v", i, err)
		}
		topologyCache.Parse(string(rune('1'+i)), "edge")
	}
	if discountCache.index == nil || topologyCache.index == nil {
		t.Fatalf("subscribe view caches should allocate indexes after actual unique values exceed threshold")
	}
	if maxCap := len(discountCache.rows) * 2; cap(discountCache.rows) > maxCap {
		t.Fatalf("discount cache row cap = %d, want at most %d", cap(discountCache.rows), maxCap)
	}
	if maxCap := len(topologyCache.rows) * 2; cap(topologyCache.rows) > maxCap {
		t.Fatalf("topology cache row cap = %d, want at most %d", cap(topologyCache.rows), maxCap)
	}
	for i := 0; i < smallSubscribeViewCacheLimit+20; i++ {
		if _, err := discountCache.Parse(`[{"quantity":1,"discount":0.9}]`); err != nil {
			t.Fatalf("duplicate discount Parse(%d): %v", i, err)
		}
		topologyCache.Parse("1", "edge")
	}
	if maxCap := len(discountCache.rows) * 2; cap(discountCache.rows) > maxCap {
		t.Fatalf("discount cache row cap after duplicates = %d, want at most %d", cap(discountCache.rows), maxCap)
	}
	if maxCap := len(topologyCache.rows) * 2; cap(topologyCache.rows) > maxCap {
		t.Fatalf("topology cache row cap after duplicates = %d, want at most %d", cap(topologyCache.rows), maxCap)
	}
}

func TestSubscribeRowsLenientKeepsEmptyListShape(t *testing.T) {
	got := SubscribeRowsLenient(nil, SubscribeOptions{})
	if got == nil || len(got) != 0 {
		t.Fatalf("empty subscribe rows should return non-nil empty slice: %#v", got)
	}
}

func TestSubscribeFromModelKeepsArrayJSONShape(t *testing.T) {
	got := SubscribeFromModelLenient(&modelSubscribe.Subscribe{}, SubscribeOptions{})
	if got.Discount == nil || got.Nodes == nil || got.NodeTags == nil {
		t.Fatalf("empty subscribe arrays should be non-nil: discount=%#v nodes=%#v node_tags=%#v", got.Discount, got.Nodes, got.NodeTags)
	}
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json marshal subscribe: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json unmarshal subscribe: %v", err)
	}
	for _, key := range []string{"discount", "nodes", "node_tags"} {
		if decoded[key] == nil {
			t.Fatalf("%s must encode as [] instead of null: %s", key, payload)
		}
		if _, ok := decoded[key].([]any); !ok {
			t.Fatalf("%s must encode as array: %T %s", key, decoded[key], payload)
		}
	}
}

func TestSubscribeFromModelKeepsArrayJSONShapeWithEmptyTopology(t *testing.T) {
	got := SubscribeFromModelLenient(&modelSubscribe.Subscribe{}, SubscribeOptions{IncludeTopology: true})
	if got.Nodes == nil || got.NodeTags == nil {
		t.Fatalf("empty topology should stay non-nil: nodes=%#v node_tags=%#v", got.Nodes, got.NodeTags)
	}
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json marshal subscribe: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json unmarshal subscribe: %v", err)
	}
	if _, ok := decoded["nodes"].([]any); !ok {
		t.Fatalf("nodes must encode as array: %T %s", decoded["nodes"], payload)
	}
	if _, ok := decoded["node_tags"].([]any); !ok {
		t.Fatalf("node_tags must encode as array: %T %s", decoded["node_tags"], payload)
	}
}

func TestSplitCommaListTrimsEmptyValues(t *testing.T) {
	got := SplitCommaList(" a,, b , a ")
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("SplitCommaList mismatch: %+v", got)
	}
}

func TestSplitCommaListKeepsEmptyArrayShape(t *testing.T) {
	got := SplitCommaList("")
	if got == nil || len(got) != 0 {
		t.Fatalf("empty comma list should be a non-nil empty slice: %#v", got)
	}
}
