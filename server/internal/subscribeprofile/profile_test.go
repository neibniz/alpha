package subscribeprofile

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/model/node"
	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/internal/xraytemplate"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestQueryAvailableNodesForSubscribeDetailsUsesPreloadedProduct(t *testing.T) {
	nodes, err := QueryAvailableNodesForSubscribeDetails(context.Background(), nil, &user.SubscribeDetails{
		Subscribe: &modelSubscribe.Subscribe{Id: 1},
	})
	if err != nil {
		t.Fatalf("QueryAvailableNodesForSubscribeDetails() error = %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("QueryAvailableNodesForSubscribeDetails() returned %d nodes, want 0", len(nodes))
	}
}

func TestQueryAvailableNodesForSubscribeDetailsUsesSnapshotFallback(t *testing.T) {
	nodes, err := QueryAvailableNodesForSubscribeDetails(context.Background(), nil, &user.SubscribeDetails{
		Subscribe:       &modelSubscribe.Subscribe{},
		ProductSnapshot: `{"version":1,"kind":"subscribe","name":"legacy"}`,
	})
	if err != nil {
		t.Fatalf("QueryAvailableNodesForSubscribeDetails() error = %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("QueryAvailableNodesForSubscribeDetails() returned %d nodes, want 0", len(nodes))
	}
}

func TestHydrateSubscribeProductsForDetailsBatchesMissingSelectors(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelSubscribe.Subscribe{}); err != nil {
		t.Fatal(err)
	}
	products := []modelSubscribe.Subscribe{
		{Id: 10, Name: "current-ten", Nodes: "1,2", NodeTags: "standard", Sort: 1},
		{Id: 11, Name: "current-eleven", Nodes: "3", NodeTags: "premium", Sort: 2},
		{Id: 12, Name: "current-twelve", Nodes: "current", NodeTags: "current", Sort: 3},
		{Id: 13, Name: "current-thirteen", Nodes: "13", NodeTags: "fallback", Sort: 4},
	}
	if err := db.Select("*").Create(&products).Error; err != nil {
		t.Fatal(err)
	}

	queryCount := 0
	if err := db.Callback().Query().Before("gorm:query").Register("test_count_hydrate_product_selector_queries", func(tx *gorm.DB) {
		queryCount++
	}); err != nil {
		t.Fatal(err)
	}

	snapshot := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:     ordersnapshot.KindSubscribe,
		Name:     "snapshot",
		Nodes:    []int64{99},
		NodeTags: []string{"snapshot"},
	})
	items := []*user.SubscribeDetails{
		{Id: 1, SubscribeId: 10},
		{Id: 2, SubscribeId: 10},
		{Id: 3, SubscribeId: 11},
		{Id: 4, SubscribeId: 12, ProductSnapshot: snapshot},
		{Id: 5, SubscribeId: 13, Subscribe: &modelSubscribe.Subscribe{}},
		{Id: 6, SubscribeId: 12, ProductSnapshot: " " + snapshot + " "},
	}

	if err := HydrateSubscribeProductsForDetails(ctx, &svc.ServiceContext{DB: db}, items); err != nil {
		t.Fatal(err)
	}
	if queryCount != 1 {
		t.Fatalf("hydrate product selector query count = %d, want 1", queryCount)
	}
	if items[0].Subscribe == nil || items[0].Subscribe.Id != 10 || items[0].Subscribe.Nodes != "1,2" || items[0].Subscribe.NodeTags != "standard" {
		t.Fatalf("first product selector not hydrated: %#v", items[0].Subscribe)
	}
	if items[1].Subscribe != items[0].Subscribe {
		t.Fatalf("duplicate subscribe id should reuse hydrated selector pointer")
	}
	if items[2].Subscribe == nil || items[2].Subscribe.Id != 11 || items[2].Subscribe.Nodes != "3" || items[2].Subscribe.NodeTags != "premium" {
		t.Fatalf("second product selector not hydrated: %#v", items[2].Subscribe)
	}
	if items[3].Subscribe == nil || items[3].Subscribe.Nodes != "99" || items[3].Subscribe.NodeTags != "snapshot" {
		t.Fatalf("snapshot selector should take precedence over current product: %#v", items[3].Subscribe)
	}
	if items[4].Subscribe == nil || items[4].Subscribe.Id != 13 || items[4].Subscribe.Nodes != "13" || items[4].Subscribe.NodeTags != "fallback" {
		t.Fatalf("empty preloaded product should fall back to current selector: %#v", items[4].Subscribe)
	}
	if items[5].Subscribe != items[3].Subscribe {
		t.Fatalf("duplicate snapshot selector should reuse cached product pointer")
	}
}

func TestSubscribeSnapshotProductCacheReusesTrimmedSnapshot(t *testing.T) {
	raw := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:     ordersnapshot.KindSubscribe,
		Name:     "snapshot",
		Nodes:    []int64{99},
		NodeTags: []string{"snapshot"},
	})
	cache := newSubscribeSnapshotProductCache(2)

	first, ok := cache.Product(" " + raw + " ")
	if !ok || first == nil || first.Nodes != "99" || first.NodeTags != "snapshot" {
		t.Fatalf("snapshot product = %#v ok=%v", first, ok)
	}
	second, ok := cache.Product(raw)
	if !ok || second != first {
		t.Fatalf("cached snapshot product = %#v ok=%v, want first pointer", second, ok)
	}
	if len(cache.rows) != 1 {
		t.Fatalf("duplicate trimmed snapshot should use one cache row, got %d", len(cache.rows))
	}
	if cache.index != nil {
		t.Fatal("small snapshot product cache should not allocate map")
	}
}

func TestSubscribeSnapshotProductCacheCachesInvalidSnapshot(t *testing.T) {
	cache := newSubscribeSnapshotProductCache(2)

	if product, ok := cache.Product(`{"kind":`); ok || product != nil {
		t.Fatalf("invalid snapshot product = %#v ok=%v, want nil/false", product, ok)
	}
	if product, ok := cache.Product(` {"kind": `); ok || product != nil {
		t.Fatalf("trim-equivalent invalid snapshot product = %#v ok=%v, want nil/false", product, ok)
	}
	if len(cache.rows) != 1 {
		t.Fatalf("duplicate invalid snapshot should use one cache row, got %d", len(cache.rows))
	}
}

func TestSubscribeSnapshotProductCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := newSubscribeSnapshotProductCache(subscribeProfileLinearLimit + 1)
	if cache == nil || cache.rows != nil || cache.index != nil {
		t.Fatalf("large snapshot product cache should not preallocate storage from hint alone: %#v", cache)
	}
	raw := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:  ordersnapshot.KindSubscribe,
		Name:  "snapshot",
		Nodes: []int64{99},
	})
	first, ok := cache.Product(" " + raw + " ")
	if !ok || first == nil {
		t.Fatalf("snapshot product = %#v ok=%v", first, ok)
	}
	second, ok := cache.Product(raw)
	if !ok || second != first || len(cache.rows) != 1 || cache.index != nil {
		t.Fatalf("duplicate snapshot should stay on linear path: rows=%#v index=%#v", cache.rows, cache.index)
	}

	cache = newSubscribeSnapshotProductCache(subscribeProfileLinearLimit + 1)
	for i := 1; i <= subscribeProfileLinearLimit+1; i++ {
		raw := ordersnapshot.Marshal(ordersnapshot.Snapshot{
			Kind:  ordersnapshot.KindSubscribe,
			Name:  "snapshot-" + strconv.Itoa(i),
			Nodes: []int64{int64(i)},
		})
		product, ok := cache.Product(raw)
		if !ok || product == nil || product.Nodes != strconv.Itoa(i) {
			t.Fatalf("snapshot product %d = %#v ok=%v", i, product, ok)
		}
	}

	if len(cache.rows) != subscribeProfileLinearLimit+1 || len(cache.index) != subscribeProfileLinearLimit+1 {
		t.Fatalf("snapshot product cache rows/index = %d/%d, want %d", len(cache.rows), len(cache.index), subscribeProfileLinearLimit+1)
	}
}

func TestSubscribeSnapshotProductCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := newSubscribeSnapshotProductCache(subscribeProfileLinearLimit + 20)
	for i := 1; i <= subscribeProfileLinearLimit+1; i++ {
		raw := ordersnapshot.Marshal(ordersnapshot.Snapshot{
			Kind:  ordersnapshot.KindSubscribe,
			Name:  "snapshot-" + strconv.Itoa(i),
			Nodes: []int64{int64(i)},
		})
		product, ok := cache.Product(raw)
		if !ok || product == nil || product.Nodes != strconv.Itoa(i) {
			t.Fatalf("snapshot product %d = %#v ok=%v", i, product, ok)
		}
	}
	if cache.index == nil {
		t.Fatal("snapshot product cache should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("snapshot product cache row cap = %d, want <= %d", cap(cache.rows), maxCap)
	}
}

func TestSubscribeProductSelectorIndexUsesLinearPathForSmallInput(t *testing.T) {
	index := newSubscribeProductSelectorIndex([]modelSubscribe.Subscribe{
		{Id: 10, Nodes: "1"},
		{Id: 11, Nodes: "2"},
	})
	if index.byID != nil {
		t.Fatal("small product selector index should not allocate map")
	}
	if got, ok := index.Find(11); !ok || got.Nodes != "2" {
		t.Fatalf("Find(11) = %#v, %v", got, ok)
	}
	if _, ok := index.Find(99); ok {
		t.Fatal("Find(99) found missing product")
	}
}

func TestSubscribeProductSelectorIndexUpgradesLargeInputToMap(t *testing.T) {
	products := make([]modelSubscribe.Subscribe, 0, subscribeProfileLinearLimit+1)
	for id := int64(1); id <= subscribeProfileLinearLimit+1; id++ {
		products = append(products, modelSubscribe.Subscribe{Id: id, Nodes: strconv.FormatInt(id, 10)})
	}
	index := newSubscribeProductSelectorIndex(products)
	if index.byID == nil {
		t.Fatal("large product selector index should allocate map")
	}
	lastID := int64(subscribeProfileLinearLimit + 1)
	if got, ok := index.Find(lastID); !ok || got.Nodes != strconv.FormatInt(lastID, 10) {
		t.Fatalf("Find(%d) = %#v, %v", lastID, got, ok)
	}
}

func TestProductNodeSelectorCacheKeyCanonicalizesSelectors(t *testing.T) {
	got := productNodeSelectorCacheKey(&modelSubscribe.Subscribe{
		Nodes:    " 2,1,2 ",
		NodeTags: " premium,edge,premium ",
	})
	want := productNodeSelectorCacheKey(&modelSubscribe.Subscribe{
		Nodes:    "1,2",
		NodeTags: "edge,premium",
	})
	if got != want {
		t.Fatalf("productNodeSelectorCacheKey() = %q, want %q", got, want)
	}
	if got != "1,2\x00edge,premium" {
		t.Fatalf("canonical selector key = %q", got)
	}
}

func TestNormalizedNodeSelectorCacheKeyCapacity(t *testing.T) {
	nodeIDs := []int64{10, 2, 300}
	tags := []string{"edge", "premium"}
	key := normalizedNodeSelectorCacheKey(nodeIDs, tags)
	if got, want := normalizedNodeSelectorCacheKeyCapacity(nodeIDs, tags), len(key); got != want {
		t.Fatalf("normalizedNodeSelectorCacheKeyCapacity() = %d, want %d", got, want)
	}
}

func TestDecimalInt64Len(t *testing.T) {
	tests := []struct {
		value int64
		want  int
	}{
		{value: 0, want: 1},
		{value: 9, want: 1},
		{value: 10, want: 2},
		{value: -10, want: 3},
	}
	for _, tt := range tests {
		if got := decimalInt64Len(tt.value); got != tt.want {
			t.Fatalf("decimalInt64Len(%d) = %d, want %d", tt.value, got, tt.want)
		}
	}
}

func TestToTypePresetKeepsZeroTimesAtZero(t *testing.T) {
	got := ToTypePreset(user.SubscribePreset{Items: "[]", IsDefault: true})
	if got.CreatedAt != 0 || got.UpdatedAt != 0 {
		t.Fatalf("zero preset times should stay zero: %#v", got)
	}
	if !got.Enabled {
		t.Fatalf("nil enabled should default to true: %#v", got)
	}
	if !got.IsDefault {
		t.Fatalf("default preset flag should be preserved: %#v", got)
	}
}

func TestToTypePresetKeepsEmptyItemsArrayShape(t *testing.T) {
	got := ToTypePreset(user.SubscribePreset{})
	if got.Items == nil || len(got.Items) != 0 {
		t.Fatalf("empty preset items = %#v, want non-nil empty slice", got.Items)
	}
}

func TestNormalizeQueryNodeSelectorsCleansTagsInOnePass(t *testing.T) {
	nodeIDs, tags := normalizeQueryNodeSelectors(
		[]int64{3, 3, 0, -1, 4},
		[]string{" edge ", "", "premium", "edge", "  "},
	)
	if want := []int64{3, 4}; !reflect.DeepEqual(nodeIDs, want) {
		t.Fatalf("node ids = %#v, want %#v", nodeIDs, want)
	}
	if want := []string{"edge", "premium"}; !reflect.DeepEqual(tags, want) {
		t.Fatalf("tags = %#v, want %#v", tags, want)
	}
}

func TestNormalizeQueryNodeSelectorsKeepsSparseNodeIDsCompact(t *testing.T) {
	ids := make([]int64, subscribeProfileLinearLimit+20)
	ids[2] = 3
	ids[subscribeProfileLinearLimit+10] = 3

	nodeIDs, tags := normalizeQueryNodeSelectors(ids, []string{" edge ", "edge"})
	if want := []int64{3}; !reflect.DeepEqual(nodeIDs, want) {
		t.Fatalf("node ids = %#v, want %#v", nodeIDs, want)
	}
	if maxCap := subscribeProfileLinearLimit + 1; cap(nodeIDs) > maxCap {
		t.Fatalf("node ids cap = %d, want <= %d", cap(nodeIDs), maxCap)
	}
	if want := []string{"edge"}; !reflect.DeepEqual(tags, want) {
		t.Fatalf("tags = %#v, want %#v", tags, want)
	}
}

func TestSubscribePresetListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(subscribePresetListColumns))
	for _, column := range subscribePresetListColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"user_subscribe_id",
		"name",
		"preset_key",
		"mode",
		"profile_key",
		"items",
		"enabled",
		"is_default",
		"created_at",
		"updated_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("subscribe preset list columns must include %q", column)
		}
	}
	if _, ok := columns["user_id"]; ok {
		t.Fatal("subscribe preset list columns must not include user_id in API projection")
	}
}

func TestSubscribePresetSelectionColumnsStayLightweight(t *testing.T) {
	want := []string{"mode", "profile_key", "items"}
	if !reflect.DeepEqual(subscribePresetSelectionColumns, want) {
		t.Fatalf("subscribe preset selection columns = %#v, want %#v", subscribePresetSelectionColumns, want)
	}
}

func TestSubscribeProfileNodeColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(subscribeProfileNodeColumns))
	for _, column := range subscribeProfileNodeColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"name",
		"tags",
		"port",
		"address",
		"server_id",
		"protocol",
		"inbound_alias",
		"sort",
		"created_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("subscribe profile node columns must include %q", column)
		}
	}

	forbidden := []string{
		"enabled",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("subscribe profile node columns must not include %q", column)
		}
	}
}

func TestSubscribeProfileServerColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(subscribeProfileServerColumns))
	for _, column := range subscribeProfileServerColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"name",
		"country",
		"city",
		"address",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("subscribe profile server columns must include %q", column)
		}
	}

	forbidden := []string{
		"sort",
		"last_reported_at",
		"created_at",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("subscribe profile server columns must not include %q", column)
		}
	}
}

func TestNormalizeUserSubscribeIDs(t *testing.T) {
	got := normalizeUserSubscribeIDs([]int64{12, 0, 12, -1, 11})
	want := []int64{12, 11}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeUserSubscribeIDs() = %#v, want %#v", got, want)
	}
}

func TestNormalizeUserSubscribeIDsSmallInputAvoidsMapAllocation(t *testing.T) {
	ids := []int64{12, 0, 12, -1, 11}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = normalizeUserSubscribeIDs(ids)
	})
	if allocs > 1 {
		t.Fatalf("normalizeUserSubscribeIDs small input allocations = %f, want at most 1", allocs)
	}
}

func TestNormalizeUserSubscribeIDsKeepsSparseInputCompact(t *testing.T) {
	ids := make([]int64, subscribeProfileLinearLimit+20)
	ids[1] = 12
	ids[subscribeProfileLinearLimit+10] = 12

	got := normalizeUserSubscribeIDs(ids)
	want := []int64{12}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeUserSubscribeIDs() = %#v, want %#v", got, want)
	}
	if maxCap := subscribeProfileLinearLimit + 1; cap(got) > maxCap {
		t.Fatalf("normalizeUserSubscribeIDs cap = %d, want <= %d", cap(got), maxCap)
	}
}

func TestNormalizeUserSubscribeIDsBoundsSparseUpgradeCapacity(t *testing.T) {
	ids := make([]int64, subscribeProfileLinearLimit+40)
	for i := 0; i <= subscribeProfileLinearLimit; i++ {
		ids[i] = int64(i + 1)
	}

	got := normalizeUserSubscribeIDs(ids)
	if len(got) != subscribeProfileLinearLimit+1 {
		t.Fatalf("normalizeUserSubscribeIDs len = %d, want %d", len(got), subscribeProfileLinearLimit+1)
	}
	if maxCap := (subscribeProfileLinearLimit + 1) * 2; cap(got) > maxCap {
		t.Fatalf("normalizeUserSubscribeIDs cap = %d, want <= %d", cap(got), maxCap)
	}
}

func TestMissingServerXrayTemplateCacheIDsDeduplicatesAndSkipsCached(t *testing.T) {
	got := missingServerXrayTemplateCacheIDs([]*node.Node{
		nil,
		{ServerId: 7},
		{ServerId: 7},
		{ServerId: 8},
		{ServerId: 0},
		{ServerId: 9},
	}, map[int64][]*node.ServerXrayTemplate{
		8: nil,
	})
	want := []int64{7, 9}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("missingServerXrayTemplateCacheIDs() = %#v, want %#v", got, want)
	}
}

func TestMissingServerXrayTemplateCacheIDsSmallInputAvoidsMapAllocation(t *testing.T) {
	nodes := []*node.Node{{ServerId: 7}, {ServerId: 7}, {ServerId: 8}}
	cache := map[int64][]*node.ServerXrayTemplate{8: nil}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = missingServerXrayTemplateCacheIDs(nodes, cache)
	})
	if allocs > 1 {
		t.Fatalf("missingServerXrayTemplateCacheIDs small input allocations = %f, want at most 1", allocs)
	}
}

func TestServerXrayTemplateCacheIDCollectorBoundsSparseUpgradeCapacity(t *testing.T) {
	collector := newServerXrayTemplateCacheIDCollector(subscribeProfileLinearLimit + 20)
	if collector.ids != nil || collector.seen != nil {
		t.Fatalf("hinted template cache id collector should not allocate before first id: %#v", collector)
	}
	collector.Add(1)
	collector.Add(1)
	if collector.seen != nil || len(collector.ids) != 1 {
		t.Fatalf("duplicate first id should stay on linear path: %#v", collector)
	}
	for id := int64(2); id <= int64(subscribeProfileLinearLimit+1); id++ {
		collector.Add(id)
	}
	if collector.seen == nil {
		t.Fatal("template cache id collector should allocate map after actual unique ids exceed threshold")
	}
	if maxCap := (subscribeProfileLinearLimit + 1) * 2; cap(collector.IDs()) > maxCap {
		t.Fatalf("collector ids cap = %d, want <= %d", cap(collector.IDs()), maxCap)
	}
}

func TestQueryEnabledServerXrayTemplateCacheSkipsEmptyInputMap(t *testing.T) {
	got, err := QueryEnabledServerXrayTemplateCache(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("empty node list should not allocate template cache map: %#v", got)
	}
}

func TestServerXrayTemplateCacheIDCollectorUpgradesLargeInputToMap(t *testing.T) {
	var collector serverXrayTemplateCacheIDCollector
	for id := int64(1); id <= subscribeProfileLinearLimit+1; id++ {
		collector.Add(id)
	}
	collector.Add(1)
	if collector.seen == nil {
		t.Fatal("large server template cache id collector should allocate map")
	}
	wantLen := subscribeProfileLinearLimit + 1
	if len(collector.IDs()) != wantLen {
		t.Fatalf("collector ids len = %d, want %d: %#v", len(collector.IDs()), wantLen, collector.IDs())
	}
}

func TestServerXrayTemplateCacheIDCollectorUsesCapacityHintLazily(t *testing.T) {
	collector := newServerXrayTemplateCacheIDCollector(subscribeProfileLinearLimit + 1)
	if collector.ids != nil || collector.seen != nil {
		t.Fatalf("hinted collector should not allocate before first id: %#v", collector)
	}
	collector.Add(7)
	if cap(collector.IDs()) != subscribeProfileLinearLimit+1 {
		t.Fatalf("collector ids cap = %d, want %d", cap(collector.IDs()), subscribeProfileLinearLimit+1)
	}
	if collector.seen != nil {
		t.Fatal("large hinted collector should not allocate map for one unique id")
	}
	collector.Add(7)
	if len(collector.IDs()) != 1 {
		t.Fatalf("collector ids = %#v, want one unique id", collector.IDs())
	}
}

func TestFilterNodesSkipsFilteredWithoutInputSizedCapacity(t *testing.T) {
	nodes := make([]*node.Node, viewutil.SmallRowsCapacityThreshold+20)
	nodes[0] = &node.Node{Id: 1, InboundAlias: "main"}
	for index := 1; index < len(nodes); index++ {
		nodes[index] = &node.Node{Id: int64(index + 1), InboundAlias: "other"}
	}

	got := FilterNodes(nodes, &Selection{ProfileKey: "main"})
	if len(got) != 1 {
		t.Fatalf("filtered nodes len = %d, want 1", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("filtered nodes cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestFilterNodesUsesCapacityHintAfterActualThreshold(t *testing.T) {
	nodes := make([]*node.Node, viewutil.SmallRowsCapacityThreshold+20)
	for index := range nodes {
		nodes[index] = &node.Node{Id: int64(index + 1), InboundAlias: "other"}
	}
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		nodes[index].InboundAlias = "main"
	}

	got := FilterNodes(nodes, &Selection{ProfileKey: "main"})
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("filtered nodes len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("filtered nodes cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestBuildNodeInfoAndProfilesUsesServerBindingIndex(t *testing.T) {
	enabled := true
	disabled := false
	server := &node.Server{Id: 9, Name: "sg", Address: "node.example.com"}
	nodes := []*node.Node{
		{
			Id:           1,
			Name:         "Singapore",
			ServerId:     server.Id,
			Server:       server,
			Protocol:     "vless",
			InboundAlias: "VLESS Reality gRPC",
			Port:         443,
			Address:      "node.example.com",
			Tags:         "edge, premium",
			CreatedAt:    time.Unix(1700000000, 0).UTC(),
		},
		{
			Id:           2,
			Name:         "Disabled",
			ServerId:     server.Id,
			Server:       server,
			Protocol:     "vless",
			InboundAlias: "disabled-inbound",
		},
	}
	bindingCache := map[int64][]*node.ServerXrayTemplate{
		server.Id: {
			{
				ServerId: server.Id,
				Alias:    "main-grpc",
				Template: &node.XrayTemplate{
					Type: node.XrayTemplateInbound,
					Name: "VLESS Reality gRPC",
					Config: `{
						"tag": "vless-reality-grpc",
						"port": 443,
						"protocol": "vless",
						"settings": {"decryption": "none"},
						"streamSettings": {"network": "tcp", "security": "none"}
					}`,
					DefaultVariables: "{}",
					Enabled:          &enabled,
				},
				Variables:           "{}",
				SubscriptionEnabled: &enabled,
				Enabled:             &enabled,
				Sort:                5,
			},
			{
				ServerId: server.Id,
				Alias:    "disabled-inbound",
				Template: &node.XrayTemplate{
					Type:             node.XrayTemplateInbound,
					Name:             "Disabled",
					Config:           `{"tag":"disabled-inbound","protocol":"vless","settings":{"decryption":"none"},"streamSettings":{"network":"tcp","security":"none"}}`,
					DefaultVariables: "{}",
					Enabled:          &enabled,
				},
				Variables:           "{}",
				SubscriptionEnabled: &disabled,
				Enabled:             &enabled,
				Sort:                1,
			},
		},
	}

	nodeInfos, profiles, err := BuildNodeInfoAndProfilesWithCache(context.Background(), nil, nodes, "user-uuid", bindingCache)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodeInfos) != 1 {
		t.Fatalf("nodeInfos len = %d, want 1: %#v", len(nodeInfos), nodeInfos)
	}
	if nodeInfos[0].Id != 1 || nodeInfos[0].ProfileKey != "VLESS Reality gRPC" || nodeInfos[0].Protocol != "vless" {
		t.Fatalf("node info = %#v", nodeInfos[0])
	}
	if !reflect.DeepEqual(nodeInfos[0].Tags, []string{"edge", "premium"}) {
		t.Fatalf("node tags = %#v", nodeInfos[0].Tags)
	}
	if len(profiles) != 1 {
		t.Fatalf("profiles len = %d, want 1: %#v", len(profiles), profiles)
	}
	if profiles[0].Key != "VLESS Reality gRPC" || profiles[0].NodeCount != 1 || profiles[0].Sort != 5 {
		t.Fatalf("profile = %#v", profiles[0])
	}
}

func TestBuildNodeInfoAndProfilesRuntimeCacheReusesRenderedServerIndex(t *testing.T) {
	enabled := true
	server := &node.Server{Id: 9, Name: "sg", Address: "node.example.com"}
	nodes := []*node.Node{profileTestNode(1, server, "main")}
	cache := NewNodeProfileRuntimeCache(2)
	cache.bindings[server.Id] = []*node.ServerXrayTemplate{
		profileTestInboundBinding(server.Id, "main", &enabled),
	}

	first, _, err := BuildNodeInfoAndProfilesWithRuntimeCache(context.Background(), nil, nodes, "first", cache)
	if err != nil {
		t.Fatal(err)
	}
	firstIndex, ok := cache.bindingIndexes.Get(server.Id)
	if !ok || firstIndex == nil {
		t.Fatal("first render did not populate server binding index cache")
	}

	second, _, err := BuildNodeInfoAndProfilesWithRuntimeCache(context.Background(), nil, nodes, "second", cache)
	if err != nil {
		t.Fatal(err)
	}
	secondIndex, ok := cache.bindingIndexes.Get(server.Id)
	if !ok || secondIndex != firstIndex {
		t.Fatal("second render rebuilt the cached server binding index")
	}
	if len(first) != 1 || first[0].Uuid != "first" || len(second) != 1 || second[0].Uuid != "second" {
		t.Fatalf("runtime cache changed per-subscription output: first=%#v second=%#v", first, second)
	}
}

func BenchmarkBuildNodeInfoAndProfilesRuntimeCache(b *testing.B) {
	enabled := true
	server := &node.Server{Id: 9, Name: "sg", Address: "node.example.com"}
	bindingCache := map[int64][]*node.ServerXrayTemplate{
		server.Id: {profileTestInboundBinding(server.Id, "main", &enabled)},
	}
	nodes := make([]*node.Node, 32)
	for i := range nodes {
		nodes[i] = profileTestNode(int64(i+1), server, "main")
	}

	b.Run("shared-render-index", func(b *testing.B) {
		cache := NewNodeProfileRuntimeCache(1)
		cache.bindings = bindingCache
		if _, _, err := BuildNodeInfoAndProfilesWithRuntimeCache(context.Background(), nil, nodes, "uuid", cache); err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, _, err := BuildNodeInfoAndProfilesWithRuntimeCache(context.Background(), nil, nodes, "uuid", cache); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("rebuild-render-index", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, _, err := BuildNodeInfoAndProfilesWithCache(context.Background(), nil, nodes, "uuid", bindingCache); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestBuildNodeInfoAndProfilesSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	enabled := true
	server := &node.Server{Id: 9, Name: "sg", Address: "node.example.com"}
	nodes := make([]*node.Node, viewutil.SmallRowsCapacityThreshold+20)
	nodes[0] = profileTestNode(1, server, "main")
	for index := 1; index < len(nodes); index++ {
		nodes[index] = profileTestNode(int64(index+1), server, "missing")
	}
	bindingCache := map[int64][]*node.ServerXrayTemplate{
		server.Id: {profileTestInboundBinding(server.Id, "main", &enabled)},
	}

	nodeInfos, _, err := BuildNodeInfoAndProfilesWithCache(context.Background(), nil, nodes, "user-uuid", bindingCache)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodeInfos) != 1 {
		t.Fatalf("node infos len = %d, want 1", len(nodeInfos))
	}
	if cap(nodeInfos) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy node infos cap = %d, want %d", cap(nodeInfos), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestBuildNodeInfoAndProfilesUsesCapacityHintAfterActualThreshold(t *testing.T) {
	enabled := true
	server := &node.Server{Id: 9, Name: "sg", Address: "node.example.com"}
	nodes := make([]*node.Node, viewutil.SmallRowsCapacityThreshold+20)
	for index := range nodes {
		nodes[index] = profileTestNode(int64(index+1), server, "missing")
	}
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		nodes[index].InboundAlias = "main"
	}
	bindingCache := map[int64][]*node.ServerXrayTemplate{
		server.Id: {profileTestInboundBinding(server.Id, "main", &enabled)},
	}

	nodeInfos, _, err := BuildNodeInfoAndProfilesWithCache(context.Background(), nil, nodes, "user-uuid", bindingCache)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodeInfos) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("node infos len = %d, want %d", len(nodeInfos), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(nodeInfos) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("node infos cap = %d, want %d", cap(nodeInfos), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestBuildNodeInfoAndProfilesEmptyInputKeepsArrayShape(t *testing.T) {
	nodeInfos, profiles, err := BuildNodeInfoAndProfilesWithCache(context.Background(), nil, nil, "user-uuid", nil)
	if err != nil {
		t.Fatal(err)
	}
	if nodeInfos == nil || len(nodeInfos) != 0 {
		t.Fatalf("empty node infos = %#v, want non-nil empty slice", nodeInfos)
	}
	if profiles == nil || len(profiles) != 0 {
		t.Fatalf("empty profiles = %#v, want non-nil empty slice", profiles)
	}
}

func TestUserSubscribeProfileAccumulatorUsesCapacityHintLazily(t *testing.T) {
	acc := newUserSubscribeProfileAccumulator(subscribeProfileLinearLimit + 1)
	if acc.rows != nil || acc.index != nil {
		t.Fatalf("hinted profile accumulator should not allocate before first profile: %#v", acc)
	}
	acc.Add("grpc", "gRPC", "vless", "grpc", "reality", 5)
	rows := acc.Rows()
	if cap(rows) != subscribeProfileLinearLimit+1 {
		t.Fatalf("profile rows cap = %d, want %d", cap(rows), subscribeProfileLinearLimit+1)
	}
	if acc.index != nil {
		t.Fatal("large hinted profile accumulator should not allocate map for one unique profile")
	}
	acc.Add("grpc", "gRPC", "vless", "grpc", "reality", 3)
	if len(rows) != 1 || acc.Rows()[0].NodeCount != 2 || acc.Rows()[0].Sort != 3 {
		t.Fatalf("duplicate profile should update existing row: %#v", acc.Rows())
	}
}

func TestUserSubscribeProfileAccumulatorBoundsSparseUpgradeCapacity(t *testing.T) {
	acc := newUserSubscribeProfileAccumulator(subscribeProfileLinearLimit + 20)
	for i := 0; i <= subscribeProfileLinearLimit; i++ {
		acc.Add(string(rune('a'+i)), "profile", "vless", "tcp", "none", i)
	}
	if acc.index == nil {
		t.Fatal("profile accumulator should allocate map after actual unique profiles exceed threshold")
	}
	if maxCap := len(acc.Rows()) * 2; cap(acc.Rows()) > maxCap {
		t.Fatalf("profile rows cap = %d, want <= %d", cap(acc.Rows()), maxCap)
	}
}

func TestUserSubscribeProfileAccumulatorUsesLinearPathForSmallInput(t *testing.T) {
	var acc userSubscribeProfileAccumulator
	acc.Add("grpc", "gRPC", "vless", "grpc", "reality", 5)
	acc.Add("tcp", "TCP", "trojan", "tcp", "tls", 3)
	acc.Add("grpc", "ignored", "ignored", "ignored", "ignored", 2)

	if acc.index != nil {
		t.Fatal("small profile accumulator should not allocate map index")
	}
	rows := acc.Rows()
	if len(rows) != 2 {
		t.Fatalf("profiles len = %d, want 2: %#v", len(rows), rows)
	}
	if rows[0].Key != "grpc" || rows[0].NodeCount != 2 || rows[0].Sort != 2 || rows[0].Protocol != "vless" {
		t.Fatalf("merged grpc profile = %#v", rows[0])
	}
	if rows[1].Key != "tcp" || rows[1].NodeCount != 1 || rows[1].Sort != 3 {
		t.Fatalf("tcp profile = %#v", rows[1])
	}
}

func TestUserSubscribeProfileAccumulatorUpgradesLargeInputToMap(t *testing.T) {
	var acc userSubscribeProfileAccumulator
	for i := 0; i <= subscribeProfileLinearLimit; i++ {
		acc.Add(string(rune('a'+i)), "profile", "vless", "tcp", "none", i)
	}
	acc.Add("a", "profile", "vless", "tcp", "none", -1)

	if acc.index == nil {
		t.Fatal("large profile accumulator should upgrade to map index")
	}
	if len(acc.rows) != subscribeProfileLinearLimit+1 {
		t.Fatalf("profile rows len = %d, want %d", len(acc.rows), subscribeProfileLinearLimit+1)
	}
	if acc.rows[0].NodeCount != 2 || acc.rows[0].Sort != -1 {
		t.Fatalf("duplicate profile should update first row: %#v", acc.rows[0])
	}
}

func TestProfileMetaCacheReusesBindingMeta(t *testing.T) {
	binding := &node.ServerXrayTemplate{
		Alias: "fallback-alias",
		Template: &node.XrayTemplate{
			Name: "VLESS Reality",
		},
	}
	cfg := xraytemplate.ConfigObject{
		"protocol": "vless",
		"streamSettings": xraytemplate.ConfigObject{
			"network":  "grpc",
			"security": "reality",
		},
	}
	cache := newProfileMetaCache(2)

	first := cache.Meta(binding, cfg, "inbound-a")
	second := cache.Meta(binding, cfg, "inbound-a")

	if first.name != "VLESS Reality" || first.protocol != "vless" || first.transport != "grpc" || first.security != "reality" {
		t.Fatalf("profile meta = %#v", first)
	}
	if second != first {
		t.Fatalf("cached profile meta = %#v, want %#v", second, first)
	}
	if len(cache.rows) != 1 {
		t.Fatalf("duplicate profile meta should use one cache row, got %d", len(cache.rows))
	}
	if cache.index != nil {
		t.Fatal("small profile meta cache should not allocate map")
	}
}

func TestProfileMetaCacheKeepsFallbackInKey(t *testing.T) {
	binding := &node.ServerXrayTemplate{}
	cache := newProfileMetaCache(2)

	first := cache.Meta(binding, nil, "inbound-a")
	second := cache.Meta(binding, nil, "inbound-b")

	if first.name != "inbound-a" || second.name != "inbound-b" {
		t.Fatalf("fallback names should stay distinct: first=%#v second=%#v", first, second)
	}
	if len(cache.rows) != 2 {
		t.Fatalf("distinct fallback profile meta should use two cache rows, got %d", len(cache.rows))
	}
}

func TestProfileMetaCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := newProfileMetaCache(subscribeProfileLinearLimit + 1)
	if cache == nil || cache.rows != nil || cache.index != nil {
		t.Fatalf("large profile meta cache should not preallocate storage from hint alone: %#v", cache)
	}
	binding := &node.ServerXrayTemplate{Alias: "alias"}
	first := cache.Meta(binding, nil, "fallback")
	second := cache.Meta(binding, nil, "fallback")
	if first != second || len(cache.rows) != 1 || cache.index != nil {
		t.Fatalf("duplicate profile meta should stay on linear path: rows=%#v index=%#v", cache.rows, cache.index)
	}

	cache = newProfileMetaCache(subscribeProfileLinearLimit + 1)
	for i := 1; i <= subscribeProfileLinearLimit+1; i++ {
		binding := &node.ServerXrayTemplate{Alias: "alias-" + strconv.Itoa(i)}
		meta := cache.Meta(binding, nil, "fallback")
		if meta.name != "alias-"+strconv.Itoa(i) {
			t.Fatalf("profile meta %d = %#v", i, meta)
		}
	}

	if len(cache.rows) != subscribeProfileLinearLimit+1 || len(cache.index) != subscribeProfileLinearLimit+1 {
		t.Fatalf("profile meta cache rows/index = %d/%d, want %d", len(cache.rows), len(cache.index), subscribeProfileLinearLimit+1)
	}
}

func TestProfileMetaCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := newProfileMetaCache(subscribeProfileLinearLimit + 20)
	for i := 1; i <= subscribeProfileLinearLimit+1; i++ {
		binding := &node.ServerXrayTemplate{Alias: "alias-" + strconv.Itoa(i)}
		meta := cache.Meta(binding, nil, "fallback")
		if meta.name != "alias-"+strconv.Itoa(i) {
			t.Fatalf("profile meta %d = %#v", i, meta)
		}
	}
	if cache.index == nil {
		t.Fatal("profile meta cache should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("profile meta cache row cap = %d, want <= %d", cap(cache.rows), maxCap)
	}
}

func TestNodeSubscriptionBindingIndexUsesLinearPathForSmallInput(t *testing.T) {
	first := &node.ServerXrayTemplate{Id: 1, Alias: "vless"}
	updated := &node.ServerXrayTemplate{Id: 2, Alias: "vless-updated"}
	var index nodeSubscriptionBindingIndex
	index.Set(" VLESS ", nodeSubscriptionBindingEntry{binding: first})
	index.Set("Trojan", nodeSubscriptionBindingEntry{binding: &node.ServerXrayTemplate{Id: 3}})
	index.Set("VLESS", nodeSubscriptionBindingEntry{binding: updated})

	if index.byInbound != nil {
		t.Fatal("small node subscription binding index should not allocate map")
	}
	if index.Len() != 2 {
		t.Fatalf("binding index len = %d, want 2", index.Len())
	}
	if got, ok := index.Find("VLESS"); !ok || got.binding != updated {
		t.Fatalf("Find(VLESS) = %#v, %v", got.binding, ok)
	}
	if _, ok := index.Find("missing"); ok {
		t.Fatal("missing binding alias should not be found")
	}
}

func TestNodeSubscriptionBindingIndexUpgradesLargeInputToMap(t *testing.T) {
	index := nodeSubscriptionBindingIndex{capHint: subscribeProfileLinearLimit + 20}
	for i := 1; i <= subscribeProfileLinearLimit+1; i++ {
		index.Set("alias-"+strconv.Itoa(i), nodeSubscriptionBindingEntry{binding: &node.ServerXrayTemplate{Id: int64(i)}})
	}
	if index.byInbound == nil {
		t.Fatal("large node subscription binding index should allocate map")
	}
	if index.Len() != subscribeProfileLinearLimit+1 {
		t.Fatalf("binding index len = %d, want %d", index.Len(), subscribeProfileLinearLimit+1)
	}
	lastAlias := "alias-" + strconv.Itoa(subscribeProfileLinearLimit+1)
	if got, ok := index.Find(lastAlias); !ok || got.binding.Id != int64(subscribeProfileLinearLimit+1) {
		t.Fatalf("Find(%s) = %#v, %v", lastAlias, got.binding, ok)
	}
	if maxCap := index.Len() * 2; cap(index.rows) > maxCap {
		t.Fatalf("binding index rows cap = %d, want <= %d", cap(index.rows), maxCap)
	}
}

func TestNodeSubscriptionBindingIndexAvoidsMapForLargeDuplicateInput(t *testing.T) {
	index := nodeSubscriptionBindingIndex{capHint: subscribeProfileLinearLimit + 20}
	updated := &node.ServerXrayTemplate{Id: 99, Alias: "updated"}
	for i := 0; i < subscribeProfileLinearLimit+20; i++ {
		index.Set("vless", nodeSubscriptionBindingEntry{binding: updated})
	}

	if index.byInbound != nil {
		t.Fatal("duplicate-heavy node subscription binding index should not allocate map")
	}
	if index.Len() != 1 {
		t.Fatalf("binding index len = %d, want 1", index.Len())
	}
	if cap(index.rows) != subscribeProfileLinearLimit+1 {
		t.Fatalf("binding index rows cap = %d, want %d", cap(index.rows), subscribeProfileLinearLimit+1)
	}
	if got, ok := index.Find("vless"); !ok || got.binding != updated {
		t.Fatalf("Find(vless) = %#v, %v; want updated", got.binding, ok)
	}
}

func TestBuildNodeSubscriptionBindingIndexUsesRenderedConfigAliases(t *testing.T) {
	enabled := true
	disabled := false
	binding := &node.ServerXrayTemplate{
		Id:                  1,
		Alias:               "main",
		SubscriptionEnabled: &enabled,
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Name: "VLESS",
			Config: `{
				"tag": "{{ .tag }}",
				"port": 443,
				"protocol": "vless",
				"settings": {"clients": [], "decryption": "none"}
			}`,
			DefaultVariables: `{"tag":"default-tag"}`,
		},
		Variables: `{"tag":"rendered-tag"}`,
	}
	disabledBinding := &node.ServerXrayTemplate{
		Id:                  2,
		Alias:               "disabled",
		SubscriptionEnabled: &disabled,
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Name: "Disabled",
			Config: `{
				"tag": "disabled-tag",
				"port": 443,
				"protocol": "vless",
				"settings": {"clients": [], "decryption": "none"}
			}`,
		},
	}

	index := buildNodeSubscriptionBindingIndex(&node.Server{Id: 9, Name: "edge", Address: "node.example.com"}, []*node.ServerXrayTemplate{binding, disabledBinding})
	got, ok := index.Find("rendered-tag")
	if !ok || got.binding != binding {
		t.Fatalf("Find(rendered-tag) = %#v/%v, want rendered binding", got.binding, ok)
	}
	if got.config["tag"] != "rendered-tag" {
		t.Fatalf("rendered config tag = %v, want rendered-tag", got.config["tag"])
	}
	if _, ok := index.Find("disabled-tag"); ok {
		t.Fatal("disabled subscription binding should not be indexed")
	}
}

func TestNodeSubscriptionBindingIndexCacheUsesLinearPathForSmallInput(t *testing.T) {
	cache := newNodeSubscriptionBindingIndexCache(subscribeProfileLinearLimit + 20)
	first := &nodeSubscriptionBindingIndex{}
	updated := &nodeSubscriptionBindingIndex{}
	cache.Set(1, first)
	cache.Set(2, &nodeSubscriptionBindingIndex{})
	cache.Set(1, updated)

	if cache.index != nil {
		t.Fatal("small binding index cache should not allocate map index")
	}
	got, ok := cache.Get(1)
	if !ok || got != updated {
		t.Fatalf("binding index cache Get(1) = %#v/%v, want updated/true", got, ok)
	}
	if _, ok := cache.Get(3); ok {
		t.Fatal("binding index cache should miss unknown server id")
	}
}

func TestNodeSubscriptionBindingIndexCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := newNodeSubscriptionBindingIndexCache(subscribeProfileLinearLimit + 20)
	for i := int64(1); i <= subscribeProfileLinearLimit+1; i++ {
		cache.Set(i, &nodeSubscriptionBindingIndex{})
	}

	if cache.index == nil {
		t.Fatal("large binding index cache should upgrade to map index")
	}
	if len(cache.rows) != subscribeProfileLinearLimit+1 {
		t.Fatalf("binding index cache rows len = %d, want %d", len(cache.rows), subscribeProfileLinearLimit+1)
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("binding index cache rows cap = %d, want <= %d", cap(cache.rows), maxCap)
	}
}

func TestNodeSubscriptionBindingIndexCacheAvoidsMapForLargeDuplicateInput(t *testing.T) {
	cache := newNodeSubscriptionBindingIndexCache(subscribeProfileLinearLimit + 20)
	updated := &nodeSubscriptionBindingIndex{}
	for i := 0; i < subscribeProfileLinearLimit+20; i++ {
		cache.Set(1, updated)
	}

	if cache.index != nil {
		t.Fatal("duplicate-heavy binding index cache should not allocate map index")
	}
	if len(cache.rows) != 1 {
		t.Fatalf("binding index cache rows len = %d, want 1", len(cache.rows))
	}
	if cap(cache.rows) != subscribeProfileLinearLimit+1 {
		t.Fatalf("binding index cache rows cap = %d, want %d", cap(cache.rows), subscribeProfileLinearLimit+1)
	}
	if got, ok := cache.Get(1); !ok || got != updated {
		t.Fatalf("binding index cache Get(1) = %#v/%v, want updated/true", got, ok)
	}
}

func TestQueryPresetsBySubscribeIDsGroupsAndFilters(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE user_subscribe_presets (
	id integer primary key,
	user_id integer not null,
	user_subscribe_id integer not null,
	name text not null,
	preset_key text not null,
	mode text not null,
	profile_key text not null,
	items text not null,
	sort integer not null,
	enabled boolean not null,
	is_default boolean not null,
	created_at datetime not null,
	updated_at datetime not null
)`).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0).UTC()
	rows := []user.SubscribePreset{
		{Id: 2, UserId: 7, UserSubscribeId: 11, Name: "second", PresetKey: "p2", Mode: user.SubscribePresetModeCustom, Items: "[]", Sort: 2, Enabled: boolPtr(true), CreatedAt: now, UpdatedAt: now},
		{Id: 1, UserId: 7, UserSubscribeId: 11, Name: "first", PresetKey: "p1", Mode: user.SubscribePresetModeCustom, Items: `[{"node_id":1,"inbound_alias":"vless","display_name":"Home Exit"}]`, Sort: 1, Enabled: boolPtr(true), IsDefault: true, CreatedAt: now, UpdatedAt: now},
		{Id: 3, UserId: 7, UserSubscribeId: 12, Name: "profile", PresetKey: "p3", Mode: user.SubscribePresetModeProfile, ProfileKey: "hysteria2", Items: "[]", Sort: 1, Enabled: boolPtr(false), CreatedAt: now, UpdatedAt: now},
		{Id: 4, UserId: 8, UserSubscribeId: 11, Name: "other-user", PresetKey: "p4", Mode: user.SubscribePresetModeCustom, Items: "[]", Sort: 1, Enabled: boolPtr(true), CreatedAt: now, UpdatedAt: now},
		{Id: 5, UserId: 7, UserSubscribeId: 13, Name: "not-requested", PresetKey: "p5", Mode: user.SubscribePresetModeCustom, Items: "[]", Sort: 1, Enabled: boolPtr(true), CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	grouped, err := QueryPresetsBySubscribeIDs(ctx, &svc.ServiceContext{DB: db}, 7, []int64{11, 12, 14, 11, 0})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(grouped); got != 3 {
		t.Fatalf("group count = %d, want 3: %#v", got, grouped)
	}
	if got := presetNames(grouped[11]); !reflect.DeepEqual(got, []string{"first", "second"}) {
		t.Fatalf("subscribe 11 preset names = %#v", got)
	}
	if got := grouped[11][0].Items; len(got) != 1 || got[0].NodeId != 1 || got[0].InboundAlias != "vless" || got[0].DisplayName != "Home Exit" {
		t.Fatalf("subscribe 11 first preset items = %#v", got)
	}
	if !grouped[11][0].IsDefault {
		t.Fatal("default preset should be preserved")
	}
	if got := presetNames(grouped[12]); !reflect.DeepEqual(got, []string{"profile"}) {
		t.Fatalf("subscribe 12 preset names = %#v", got)
	}
	if grouped[12][0].Enabled {
		t.Fatal("disabled preset should be preserved as Enabled=false")
	}
	if got, ok := grouped[14]; !ok || got == nil || len(got) != 0 {
		t.Fatalf("subscribe 14 empty group = %#v, exists=%v; want non-nil empty slice", got, ok)
	}
}

func TestGroupPresetRowsBySubscribeIDPreallocatesLargeGroups(t *testing.T) {
	presets := []user.SubscribePreset{
		{UserSubscribeId: 11, Name: "a1", Items: "[]"},
		{UserSubscribeId: 11, Name: "a2", Items: "[]"},
		{UserSubscribeId: 12, Name: "b1", Items: "[]"},
		{UserSubscribeId: 11, Name: "a3", Items: "[]"},
		{UserSubscribeId: 12, Name: "b2", Items: "[]"},
		{UserSubscribeId: 11, Name: "a4", Items: "[]"},
		{UserSubscribeId: 12, Name: "b3", Items: "[]"},
		{UserSubscribeId: 11, Name: "a5", Items: "[]"},
		{UserSubscribeId: 12, Name: "b4", Items: "[]"},
	}

	grouped := groupPresetRowsBySubscribeID(presets, []int64{11, 12, 13})
	if got := presetNames(grouped[11]); !reflect.DeepEqual(got, []string{"a1", "a2", "a3", "a4", "a5"}) {
		t.Fatalf("subscribe 11 preset names = %#v", got)
	}
	if got := presetNames(grouped[12]); !reflect.DeepEqual(got, []string{"b1", "b2", "b3", "b4"}) {
		t.Fatalf("subscribe 12 preset names = %#v", got)
	}
	if cap(grouped[11]) != 5 || cap(grouped[12]) != 4 {
		t.Fatalf("group caps = 11:%d 12:%d, want exact counts 5 and 4", cap(grouped[11]), cap(grouped[12]))
	}
	if grouped[13] == nil || len(grouped[13]) != 0 {
		t.Fatalf("empty requested group 13 = %#v, want non-nil empty slice", grouped[13])
	}
}

func TestPresetItemsCacheReusesTrimmedItems(t *testing.T) {
	cache := newPresetItemsCache(2)
	raw := `[{"node_id":2,"inbound_alias":" trojan ","display_name":" Backup "}]`

	first, err := cache.Parse(" " + raw + " ")
	if err != nil {
		t.Fatal(err)
	}
	second, err := cache.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	want := []types.UserSubscribePresetItem{{NodeId: 2, InboundAlias: "trojan", DisplayName: "Backup"}}
	if !reflect.DeepEqual(first, want) || !reflect.DeepEqual(second, first) {
		t.Fatalf("cached preset items mismatch: first=%#v second=%#v", first, second)
	}
	if len(cache.rows) != 1 {
		t.Fatalf("duplicate trimmed preset items should use one cache row, got %d", len(cache.rows))
	}
	if cache.index != nil {
		t.Fatal("small preset items cache should not allocate map")
	}
}

func TestPresetItemsCacheCachesInvalidItems(t *testing.T) {
	cache := newPresetItemsCache(2)

	if items, err := cache.Parse(`[{"node_id":`); err == nil || items != nil {
		t.Fatalf("invalid items should return nil/error, got items=%#v err=%v", items, err)
	}
	if items, err := cache.Parse(` [{"node_id": `); err == nil || items != nil {
		t.Fatalf("trim-equivalent invalid items should return cached nil/error, got items=%#v err=%v", items, err)
	}
	if len(cache.rows) != 1 {
		t.Fatalf("duplicate invalid preset items should use one cache row, got %d", len(cache.rows))
	}
}

func TestPresetItemsCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := newPresetItemsCache(selectionLinearLimit + 1)
	if cache == nil || cache.rows != nil || cache.index != nil {
		t.Fatalf("large preset items cache should not preallocate storage from hint alone: %#v", cache)
	}
	raw := `[{"node_id":2,"inbound_alias":" trojan ","display_name":" Backup "}]`
	first, err := cache.Parse(" " + raw + " ")
	if err != nil {
		t.Fatal(err)
	}
	second, err := cache.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != len(second) || len(cache.rows) != 1 || cache.index != nil {
		t.Fatalf("duplicate preset items should stay on linear path: rows=%#v index=%#v", cache.rows, cache.index)
	}

	cache = newPresetItemsCache(selectionLinearLimit + 1)
	for i := 1; i <= selectionLinearLimit+1; i++ {
		items, err := cache.Parse(`[{"node_id":` + strconv.Itoa(i) + `,"inbound_alias":"vless"}]`)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].NodeId != int64(i) {
			t.Fatalf("preset items %d = %#v", i, items)
		}
	}
	_, err = cache.Parse(`[{"node_id":1,"inbound_alias":"vless"}]`)
	if err != nil {
		t.Fatal(err)
	}

	if len(cache.rows) != selectionLinearLimit+1 || len(cache.index) != selectionLinearLimit+1 {
		t.Fatalf("preset items cache rows/index = %d/%d, want %d", len(cache.rows), len(cache.index), selectionLinearLimit+1)
	}
}

func TestPresetItemsCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := newPresetItemsCache(selectionLinearLimit + 20)
	for i := 1; i <= selectionLinearLimit+1; i++ {
		items, err := cache.Parse(`[{"node_id":` + strconv.Itoa(i) + `,"inbound_alias":"vless"}]`)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].NodeId != int64(i) {
			t.Fatalf("preset items %d = %#v", i, items)
		}
	}
	if cache.index == nil {
		t.Fatal("preset items cache should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("preset items cache row cap = %d, want <= %d", cap(cache.rows), maxCap)
	}
}

func TestQueryPresetsBySubscribeIDsKeepsEmptyMapShape(t *testing.T) {
	grouped, err := QueryPresetsBySubscribeIDs(context.Background(), nil, 7, nil)
	if err != nil {
		t.Fatal(err)
	}
	if grouped == nil || len(grouped) != 0 {
		t.Fatalf("empty subscribe ids map = %#v, want non-nil empty map", grouped)
	}
	if got := grouped[11]; got != nil {
		t.Fatalf("missing key lookup = %#v, want nil missing key", got)
	}
	grouped, err = QueryPresetsBySubscribeIDs(context.Background(), nil, 0, []int64{11})
	if err != nil {
		t.Fatal(err)
	}
	if grouped == nil || len(grouped) != 0 {
		t.Fatalf("invalid user id map = %#v, want non-nil empty map", grouped)
	}
}

func TestQueryPresetsKeepsEmptyListShape(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&user.SubscribePreset{}); err != nil {
		t.Fatal(err)
	}

	presets, err := QueryPresets(ctx, &svc.ServiceContext{DB: db}, 7, 11)
	if err != nil {
		t.Fatal(err)
	}
	if presets == nil || len(presets) != 0 {
		t.Fatalf("empty presets = %#v, want non-nil empty slice", presets)
	}
}

func TestResolveSelectionIncludesPresetDisplayNames(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&user.SubscribePreset{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	preset := user.SubscribePreset{
		UserId:          7,
		UserSubscribeId: 11,
		Name:            "custom",
		PresetKey:       "ps_custom",
		Mode:            user.SubscribePresetModeCustom,
		Items:           `[{"node_id":1,"inbound_alias":"vless","display_name":"Home Exit"},{"node_id":2,"inbound_alias":"trojan"}]`,
		Enabled:         &enabled,
		IsDefault:       true,
	}
	if err := db.Create(&preset).Error; err != nil {
		t.Fatal(err)
	}

	selection, err := ResolveSelection(ctx, &svc.ServiceContext{DB: db}, &user.Subscribe{Id: 11, UserId: 7}, "", "ps_custom")
	if err != nil {
		t.Fatal(err)
	}
	vlessKey, ok := newPresetItemKey(1, "vless")
	if !ok {
		t.Fatal("expected valid vless selection key")
	}
	if !selection.Items.Contains(vlessKey) {
		t.Fatalf("selection items missing vless: %#v", selection.Items)
	}
	if got := selection.DisplayName(1, "vless"); got != "Home Exit" {
		t.Fatalf("display name = %q, want Home Exit", got)
	}
	if got := selection.DisplayName(2, "trojan"); got != "" {
		t.Fatalf("empty display name = %q, want empty", got)
	}

	defaultSelection, err := ResolveSelection(ctx, &svc.ServiceContext{DB: db}, &user.Subscribe{Id: 11, UserId: 7}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := defaultSelection.DisplayName(1, "vless"); got != "Home Exit" {
		t.Fatalf("default preset display name = %q, want Home Exit", got)
	}
}

func TestSelectionItemSetUsesLinearPathForSmallInput(t *testing.T) {
	set := newSelectionItemSet(2)
	if set.items != nil || set.index != nil {
		t.Fatalf("small selection item set should not allocate before first item: %#v", set)
	}
	vless, ok := newPresetItemKey(1, "vless")
	if !ok {
		t.Fatal("expected valid vless selection key")
	}
	trojan, ok := newPresetItemKey(2, "trojan")
	if !ok {
		t.Fatal("expected valid trojan selection key")
	}
	ss, ok := newPresetItemKey(3, "ss")
	if !ok {
		t.Fatal("expected valid ss selection key")
	}
	if !set.Add(vless) || !set.Add(trojan) {
		t.Fatalf("expected new selection keys to be added: %#v", set)
	}
	if cap(set.items) != 2 {
		t.Fatalf("small selection item set cap = %d, want 2", cap(set.items))
	}
	if set.Add(vless) {
		t.Fatalf("duplicate selection key should not be added: %#v", set)
	}
	if set.index != nil {
		t.Fatal("small selection item set should not allocate map")
	}
	if set.Len() != 2 || !set.Contains(vless) || !set.Contains(trojan) || set.Contains(ss) {
		t.Fatalf("selection set mismatch: %#v", set)
	}
}

func TestSelectionItemSetUpgradesLargeInputToMap(t *testing.T) {
	set := newSelectionItemSet(selectionLinearLimit + 1)
	if set.items != nil || set.index != nil {
		t.Fatalf("large selection item set should not allocate storage from hint alone: %#v", set)
	}
	firstKey, ok := newPresetItemKey(1, "vless")
	if !ok {
		t.Fatal("expected valid first selection key")
	}
	set.Add(firstKey)
	set.Add(firstKey)
	if set.index != nil || set.Len() != 1 {
		t.Fatalf("duplicate first selection key should stay on linear path: %#v", set)
	}
	for i := 2; i <= selectionLinearLimit+1; i++ {
		key, ok := newPresetItemKey(int64(i), "vless")
		if !ok {
			t.Fatalf("expected valid selection key for id %d", i)
		}
		set.Add(key)
	}
	if set.index == nil {
		t.Fatal("large selection item set should allocate map")
	}
	if set.Len() != selectionLinearLimit+1 {
		t.Fatalf("selection set length = %d, want %d", set.Len(), selectionLinearLimit+1)
	}
	lastKey, ok := newPresetItemKey(int64(selectionLinearLimit+1), "vless")
	if !ok {
		t.Fatal("expected valid last selection key")
	}
	if !set.Contains(lastKey) {
		t.Fatal("selection set missing last key")
	}
}

func TestSelectionItemSetBoundsSparseUpgradeCapacity(t *testing.T) {
	set := newSelectionItemSet(selectionLinearLimit + 20)
	for i := 1; i <= selectionLinearLimit+1; i++ {
		key, ok := newPresetItemKey(int64(i), "vless")
		if !ok {
			t.Fatalf("expected valid selection key for id %d", i)
		}
		set.Add(key)
	}
	if set.index == nil {
		t.Fatal("selection item set should allocate map after actual unique values exceed threshold")
	}
	if maxCap := set.Len() * 2; cap(set.items) > maxCap {
		t.Fatalf("selection item set cap = %d, want <= %d", cap(set.items), maxCap)
	}
}

func TestSelectionItemNameIndexUsesLinearPathForSmallInput(t *testing.T) {
	index := newSelectionItemNameIndex(2)
	if index.rows != nil || index.index != nil {
		t.Fatalf("small selection item name index should not allocate before first item: %#v", index)
	}
	vless, ok := newPresetItemKey(1, "vless")
	if !ok {
		t.Fatal("expected valid vless selection key")
	}
	trojan, ok := newPresetItemKey(2, "trojan")
	if !ok {
		t.Fatal("expected valid trojan selection key")
	}
	ss, ok := newPresetItemKey(3, "ss")
	if !ok {
		t.Fatal("expected valid ss selection key")
	}
	index.Set(vless, "Home")
	index.Set(trojan, "Backup")
	index.Set(vless, "Updated")
	if cap(index.rows) != 2 {
		t.Fatalf("small selection item name index cap = %d, want 2", cap(index.rows))
	}

	if index.index != nil {
		t.Fatal("small selection item name index should not allocate map")
	}
	if got, ok := index.Find(vless); !ok || got != "Updated" {
		t.Fatalf("Find(1:vless) = %q, %v", got, ok)
	}
	if _, ok := index.Find(ss); ok {
		t.Fatal("missing selection item name found")
	}
}

func TestSelectionItemNameIndexUpgradesLargeInputToMap(t *testing.T) {
	index := newSelectionItemNameIndex(selectionLinearLimit + 1)
	if index.rows != nil || index.index != nil {
		t.Fatalf("large selection item name index should not allocate storage from hint alone: %#v", index)
	}
	firstKey, ok := newPresetItemKey(1, "vless")
	if !ok {
		t.Fatal("expected valid first selection key")
	}
	index.Set(firstKey, "Node")
	index.Set(firstKey, "Updated")
	if index.index != nil || len(index.rows) != 1 {
		t.Fatalf("duplicate first selection item name should stay on linear path: %#v", index)
	}
	for i := 2; i <= selectionLinearLimit+1; i++ {
		key, ok := newPresetItemKey(int64(i), "vless")
		if !ok {
			t.Fatalf("expected valid selection key for id %d", i)
		}
		index.Set(key, "Node")
	}
	if index.index == nil {
		t.Fatal("large selection item name index should allocate map")
	}
	lastKey, ok := newPresetItemKey(int64(selectionLinearLimit+1), "vless")
	if !ok {
		t.Fatal("expected valid last selection key")
	}
	if got, ok := index.Find(lastKey); !ok || got != "Node" {
		t.Fatalf("Find(%v) = %q, %v", lastKey, got, ok)
	}
}

func TestSelectionItemNameIndexBoundsSparseUpgradeCapacity(t *testing.T) {
	index := newSelectionItemNameIndex(selectionLinearLimit + 20)
	for i := 1; i <= selectionLinearLimit+1; i++ {
		key, ok := newPresetItemKey(int64(i), "vless")
		if !ok {
			t.Fatalf("expected valid selection key for id %d", i)
		}
		index.Set(key, "Node")
	}
	if index.index == nil {
		t.Fatal("selection item name index should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(index.rows) * 2; cap(index.rows) > maxCap {
		t.Fatalf("selection item name index cap = %d, want <= %d", cap(index.rows), maxCap)
	}
}

func TestValidateSelectedNodeInfoItemsPreservesDisplayName(t *testing.T) {
	selected, err := ValidateSelectedNodeInfoItems([]types.UserSubscribeNodeInfo{
		{Id: 1, InboundAlias: "vless"},
	}, []types.UserSubscribePresetItem{
		{NodeId: 1, InboundAlias: " vless ", DisplayName: "  Home Exit  "},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 {
		t.Fatalf("selected len = %d, want 1", len(selected))
	}
	if selected[0].InboundAlias != "vless" || selected[0].DisplayName != "Home Exit" {
		t.Fatalf("selected item = %#v", selected[0])
	}
}

func TestValidateSelectedItemsPreservesDisplayName(t *testing.T) {
	selected, err := ValidateSelectedItems([]*node.Node{
		{Id: 1, InboundAlias: "vless"},
	}, []types.UserSubscribePresetItem{
		{NodeId: 1, InboundAlias: " vless ", DisplayName: "  Home Exit  "},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 {
		t.Fatalf("selected len = %d, want 1", len(selected))
	}
	if selected[0].InboundAlias != "vless" || selected[0].DisplayName != "Home Exit" {
		t.Fatalf("selected item = %#v", selected[0])
	}
}

func TestValidateSelectedItemsDeduplicatesAcrossAvailableTypes(t *testing.T) {
	selected := []types.UserSubscribePresetItem{
		{NodeId: 1, InboundAlias: " vless ", DisplayName: "Home Exit"},
		{NodeId: 1, InboundAlias: "vless", DisplayName: "Duplicate"},
		{NodeId: 2, InboundAlias: "trojan"},
	}

	fromNodes, err := ValidateSelectedItems([]*node.Node{
		{Id: 1, InboundAlias: "vless"},
		{Id: 2, InboundAlias: "trojan"},
	}, selected)
	if err != nil {
		t.Fatal(err)
	}
	fromNodeInfos, err := ValidateSelectedNodeInfoItems([]types.UserSubscribeNodeInfo{
		{Id: 1, InboundAlias: "vless"},
		{Id: 2, InboundAlias: "trojan"},
	}, selected)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(fromNodes, fromNodeInfos) {
		t.Fatalf("selection mismatch: nodes=%#v nodeInfos=%#v", fromNodes, fromNodeInfos)
	}
	if len(fromNodes) != 2 {
		t.Fatalf("deduplicated selection len = %d, want 2: %#v", len(fromNodes), fromNodes)
	}
	if fromNodes[0].DisplayName != "Home Exit" || fromNodes[1].InboundAlias != "trojan" {
		t.Fatalf("deduplicated selection = %#v", fromNodes)
	}
}

func TestValidateSelectedItemsRejectsUnavailableAcrossAvailableTypes(t *testing.T) {
	selected := []types.UserSubscribePresetItem{{NodeId: 2, InboundAlias: "trojan"}}
	if _, err := ValidateSelectedItems([]*node.Node{{Id: 1, InboundAlias: "vless"}}, selected); err == nil {
		t.Fatal("ValidateSelectedItems should reject unavailable items")
	}
	if _, err := ValidateSelectedNodeInfoItems([]types.UserSubscribeNodeInfo{{Id: 1, InboundAlias: "vless"}}, selected); err == nil {
		t.Fatal("ValidateSelectedNodeInfoItems should reject unavailable items")
	}
}

func TestValidateSelectedPresetItemsSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	key, ok := newPresetItemKey(1, "vless")
	if !ok {
		t.Fatal("expected valid key")
	}
	allowed := newPresetItemKeySet(1)
	allowed.Add(key)
	selected := make([]types.UserSubscribePresetItem, viewutil.SmallRowsCapacityThreshold+20)
	selected[0] = types.UserSubscribePresetItem{NodeId: 1, InboundAlias: " vless "}

	got, err := validateSelectedPresetItems(allowed, selected)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("validated selected len = %d, want 1", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("validated selected cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestValidateSelectedPresetItemsUsesCapacityHintAfterActualThreshold(t *testing.T) {
	allowed := newPresetItemKeySet(viewutil.SmallRowsCapacityThreshold + 1)
	selected := make([]types.UserSubscribePresetItem, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		nodeID := int64(index + 1)
		key, ok := newPresetItemKey(nodeID, "vless")
		if !ok {
			t.Fatalf("expected valid key for id %d", nodeID)
		}
		allowed.Add(key)
		selected[index] = types.UserSubscribePresetItem{NodeId: nodeID, InboundAlias: "vless"}
	}

	got, err := validateSelectedPresetItems(allowed, selected)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("validated selected len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("validated selected cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestNormalizeItemsSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	items := make([]types.UserSubscribePresetItem, viewutil.SmallRowsCapacityThreshold+20)
	items[0] = types.UserSubscribePresetItem{NodeId: 1, InboundAlias: " vless ", DisplayName: "  Home  "}

	got := normalizeItems(items)
	if len(got) != 1 {
		t.Fatalf("normalized items len = %d, want 1", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("normalized items cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].InboundAlias != "vless" || got[0].DisplayName != "Home" {
		t.Fatalf("normalized item = %#v", got[0])
	}
}

func TestNormalizeItemsUsesCapacityHintAfterActualThreshold(t *testing.T) {
	items := make([]types.UserSubscribePresetItem, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		items[index] = types.UserSubscribePresetItem{NodeId: int64(viewutil.SmallRowsCapacityThreshold + 1 - index), InboundAlias: "vless"}
	}

	got := normalizeItems(items)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("normalized items len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("normalized items cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].NodeId != 1 || got[len(got)-1].NodeId != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("normalized items sort order = %#v", got)
	}
}

func TestPresetItemKeySetUsesLinearDedupForSmallInput(t *testing.T) {
	set := newPresetItemKeySet(2)
	if set.items != nil || set.index != nil {
		t.Fatalf("small preset item key set should not allocate before first item: %#v", set)
	}
	vless, ok := newPresetItemKey(1, "vless")
	if !ok {
		t.Fatal("expected valid vless key")
	}
	trojan, ok := newPresetItemKey(2, "trojan")
	if !ok {
		t.Fatal("expected valid trojan key")
	}

	if !set.Add(vless) || !set.Add(trojan) {
		t.Fatalf("expected new keys to be added: %#v", set)
	}
	if cap(set.items) != 2 {
		t.Fatalf("small preset item key set cap = %d, want 2", cap(set.items))
	}
	if set.Add(vless) {
		t.Fatalf("duplicate key should not be added: %#v", set)
	}
	if set.index != nil {
		t.Fatal("small preset item key set should not allocate map")
	}
	if set.Len() != 2 {
		t.Fatalf("set length = %d, want 2", set.Len())
	}
	if !set.Contains(vless) || !set.Contains(trojan) {
		t.Fatalf("set should contain both keys: %#v", set)
	}
}

func TestPresetItemKeySetUpgradesLargeInputToMap(t *testing.T) {
	set := newPresetItemKeySet(presetItemKeySetLinearLimit + 1)
	if set.items != nil || set.index != nil {
		t.Fatalf("large preset item key set should not allocate storage from hint alone: %#v", set)
	}
	first, ok := newPresetItemKey(1, "vless")
	if !ok {
		t.Fatal("expected valid first key")
	}
	set.Add(first)
	if set.Add(first) || set.index != nil || set.Len() != 1 {
		t.Fatalf("duplicate first key should stay on linear path: %#v", set)
	}
	for i := 2; i <= presetItemKeySetLinearLimit+1; i++ {
		key, ok := newPresetItemKey(int64(i), "vless")
		if !ok {
			t.Fatalf("expected valid key for id %d", i)
		}
		set.Add(key)
	}
	if set.Add(first) {
		t.Fatal("duplicate key should not be added after map upgrade")
	}
	if set.index == nil {
		t.Fatal("large preset item key set should upgrade to map")
	}
	if set.Len() != presetItemKeySetLinearLimit+1 {
		t.Fatalf("set length = %d, want %d", set.Len(), presetItemKeySetLinearLimit+1)
	}
}

func TestPresetItemKeySetBoundsSparseUpgradeCapacity(t *testing.T) {
	set := newPresetItemKeySet(presetItemKeySetLinearLimit + 20)
	for i := 1; i <= presetItemKeySetLinearLimit+1; i++ {
		key, ok := newPresetItemKey(int64(i), "vless")
		if !ok {
			t.Fatalf("expected valid key for id %d", i)
		}
		set.Add(key)
	}
	if set.index == nil {
		t.Fatal("preset item key set should allocate map after actual unique values exceed threshold")
	}
	if maxCap := set.Len() * 2; cap(set.items) > maxCap {
		t.Fatalf("preset item key set cap = %d, want <= %d", cap(set.items), maxCap)
	}
}

func TestMarshalItemsUsesPresetKeySetDedup(t *testing.T) {
	raw := MarshalItems([]types.UserSubscribePresetItem{
		{NodeId: 2, InboundAlias: " trojan ", DisplayName: "  Backup  "},
		{NodeId: 1, InboundAlias: "vless", DisplayName: "Home"},
		{NodeId: 1, InboundAlias: " vless ", DisplayName: "Duplicate"},
		{NodeId: 0, InboundAlias: "ignored"},
	})
	items, err := ParseItems(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := []types.UserSubscribePresetItem{
		{NodeId: 1, InboundAlias: "vless", DisplayName: "Home"},
		{NodeId: 2, InboundAlias: "trojan", DisplayName: "Backup"},
	}
	if !reflect.DeepEqual(items, want) {
		t.Fatalf("preset items = %#v, want %#v; raw=%s", items, want, raw)
	}
}

func TestNodeTagListNormalizesResponseTags(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "empty", raw: "", want: []string{}},
		{name: "blank", raw: " ,  ,", want: []string{}},
		{name: "trim and dedupe", raw: "edge, premium,edge,, premium ", want: []string{"edge", "premium"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nodeTagList(tt.raw); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("nodeTagList(%q) = %#v, want %#v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestNodeTagListSmallInputAvoidsMapAllocation(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		_ = nodeTagList("edge, premium,edge")
	})
	if allocs > 1 {
		t.Fatalf("nodeTagList small input allocations = %f, want at most 1", allocs)
	}
}

func TestNodeTagListCacheReusesRawTags(t *testing.T) {
	cache := newNodeTagListCache(2)

	first := cache.Tags("edge, premium,edge")
	second := cache.Tags("edge, premium,edge")

	want := []string{"edge", "premium"}
	if !reflect.DeepEqual(first, want) || !reflect.DeepEqual(second, first) {
		t.Fatalf("cached node tags mismatch: first=%#v second=%#v", first, second)
	}
	if len(cache.rows) != 1 {
		t.Fatalf("duplicate node tags should use one cache row, got %d", len(cache.rows))
	}
	if cache.index != nil {
		t.Fatal("small node tag list cache should not allocate map")
	}
}

func TestNodeTagListCacheCachesEmptyTags(t *testing.T) {
	cache := newNodeTagListCache(2)

	if tags := cache.Tags(" , "); tags == nil || len(tags) != 0 {
		t.Fatalf("empty tags = %#v, want non-nil empty slice", tags)
	}
	if tags := cache.Tags(" , "); tags == nil || len(tags) != 0 {
		t.Fatalf("cached empty tags = %#v, want non-nil empty slice", tags)
	}
	if len(cache.rows) != 1 {
		t.Fatalf("duplicate empty tags should use one cache row, got %d", len(cache.rows))
	}
}

func TestNodeTagListCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := newNodeTagListCache(subscribeProfileLinearLimit + 1)
	if cache == nil || cache.rows != nil || cache.index != nil {
		t.Fatalf("large node tag list cache should not preallocate storage from hint alone: %#v", cache)
	}
	first := cache.Tags("edge, premium,edge")
	second := cache.Tags("edge, premium,edge")
	if !reflect.DeepEqual(first, second) || len(cache.rows) != 1 || cache.index != nil {
		t.Fatalf("duplicate node tags should stay on linear path: rows=%#v index=%#v", cache.rows, cache.index)
	}

	cache = newNodeTagListCache(subscribeProfileLinearLimit + 1)
	for i := 1; i <= subscribeProfileLinearLimit+1; i++ {
		tags := cache.Tags("tag-" + strconv.Itoa(i))
		if len(tags) != 1 || tags[0] != "tag-"+strconv.Itoa(i) {
			t.Fatalf("tags %d = %#v", i, tags)
		}
	}
	cache.Tags("tag-1")

	if len(cache.rows) != subscribeProfileLinearLimit+1 || len(cache.index) != subscribeProfileLinearLimit+1 {
		t.Fatalf("node tag list cache rows/index = %d/%d, want %d", len(cache.rows), len(cache.index), subscribeProfileLinearLimit+1)
	}
}

func TestNodeTagListCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := newNodeTagListCache(subscribeProfileLinearLimit + 20)
	for i := 1; i <= subscribeProfileLinearLimit+1; i++ {
		tags := cache.Tags("tag-" + strconv.Itoa(i))
		if len(tags) != 1 || tags[0] != "tag-"+strconv.Itoa(i) {
			t.Fatalf("tags %d = %#v", i, tags)
		}
	}
	if cache.index == nil {
		t.Fatal("node tag list cache should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("node tag list cache row cap = %d, want <= %d", cap(cache.rows), maxCap)
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func profileTestNode(id int64, server *node.Server, inboundAlias string) *node.Node {
	return &node.Node{
		Id:           id,
		Name:         inboundAlias,
		ServerId:     server.Id,
		Server:       server,
		Protocol:     "vless",
		InboundAlias: inboundAlias,
		Port:         443,
		Address:      server.Address,
		CreatedAt:    time.Unix(1700000000, 0).UTC(),
	}
}

func profileTestInboundBinding(serverID int64, alias string, enabled *bool) *node.ServerXrayTemplate {
	return &node.ServerXrayTemplate{
		ServerId: serverID,
		Alias:    alias,
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Name: alias,
			Config: `{
				"tag": "` + alias + `",
				"port": 443,
				"protocol": "vless",
				"settings": {"decryption": "none"},
				"streamSettings": {"network":"tcp","security":"none"}
			}`,
			DefaultVariables: "{}",
			Enabled:          enabled,
		},
		Variables:           "{}",
		SubscriptionEnabled: enabled,
		Enabled:             enabled,
		Sort:                5,
	}
}

func presetNames(presets []types.UserSubscribePreset) []string {
	names := make([]string, 0, len(presets))
	for _, preset := range presets {
		names = append(names, preset.Name)
	}
	return names
}
