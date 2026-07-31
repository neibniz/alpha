package subscribe

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSubscribeCacheColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(subscribeCacheColumns))
	for _, column := range subscribeCacheColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "nodes", "node_tags", "sort"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("subscribe cache columns must include %q", column)
		}
	}

	forbidden := []string{
		"description",
		"discount",
		"unit_price",
		"traffic",
		"speed_limit",
		"device_limit",
		"created_at",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("subscribe cache columns must not include %q", column)
		}
	}
}

func TestQuerySubscribeCacheRowsSkipsDuplicateIDsWithoutInputSizedCapacity(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Subscribe{Id: 7, Name: "basic"}).Error; err != nil {
		t.Fatal(err)
	}
	ids := make([]int64, subscribeSmallSetLinearDedupLimit+20)
	for index := range ids {
		ids[index] = 7
	}

	rows, err := (&customSubscribeModel{defaultSubscribeModel: newSubscribeModel(db, nil)}).querySubscribeCacheRows(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Id != 7 {
		t.Fatalf("cache rows = %#v, want subscribe 7 only", rows)
	}
	if maxCap := subscribeSmallSetLinearDedupLimit + 1; cap(rows) > maxCap {
		t.Fatalf("cache rows cap = %d, want <= %d", cap(rows), maxCap)
	}
}

func TestSubscribeRuntimeColumnsStayLightweight(t *testing.T) {
	want := []string{"id", "name", "nodes", "node_tags"}
	if !reflect.DeepEqual(subscribeRuntimeColumns, want) {
		t.Fatalf("subscribe runtime columns = %#v, want %#v", subscribeRuntimeColumns, want)
	}
}

func TestSubscribeNodeSelectorColumnsStayLightweight(t *testing.T) {
	want := []string{"id", "nodes", "node_tags"}
	if !reflect.DeepEqual(subscribeNodeSelectorColumns, want) {
		t.Fatalf("subscribe node selector columns = %#v, want %#v", subscribeNodeSelectorColumns, want)
	}
}

func TestSubscribeOrderActivationColumnsStayFocused(t *testing.T) {
	want := []string{"id", "name", "unit_time", "traffic", "nodes", "node_tags", "renewal_reset"}
	if !reflect.DeepEqual(subscribeOrderActivationColumns, want) {
		t.Fatalf("subscribe order activation columns = %#v, want %#v", subscribeOrderActivationColumns, want)
	}
}

func TestSubscribePublicPurchaseColumnsStayFocused(t *testing.T) {
	columns := map[string]struct{}{}
	for _, column := range subscribePublicPurchaseColumns {
		columns[column] = struct{}{}
	}
	required := []string{
		"id",
		"name",
		"description",
		"unit_price",
		"unit_time",
		"discount",
		"replacement",
		"inventory",
		"traffic",
		"speed_limit",
		"device_limit",
		"quota",
		"nodes",
		"node_tags",
		"user_visible",
		"sell",
		"allow_renewal",
		"allow_reset_traffic",
		"reset_cycle",
		"renewal_reset",
		"show_original_price",
		"display_template",
		"badge",
		"min_quantity",
		"max_quantity",
		"sale_starts_at",
		"sale_ends_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("subscribe public purchase columns must include %q", column)
		}
	}
	forbidden := []string{
		"language",
		"show",
		"sort",
		"deduction_ratio",
		"allow_deduction",
		"created_at",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("subscribe public purchase columns must not include %q", column)
		}
	}
}

func TestSubscribePortalPrePurchaseColumnsStayFocused(t *testing.T) {
	want := []string{
		"id",
		"unit_price",
		"discount",
		"inventory",
		"show",
		"sell",
		"min_quantity",
		"max_quantity",
		"sale_starts_at",
		"sale_ends_at",
	}
	if !reflect.DeepEqual(subscribePortalPrePurchaseColumns, want) {
		t.Fatalf("subscribe portal pre-purchase columns = %#v, want %#v", subscribePortalPrePurchaseColumns, want)
	}
}

func TestSubscribeOrderSnapshotColumnsStayFocused(t *testing.T) {
	want := []string{
		"id",
		"name",
		"description",
		"unit_price",
		"unit_time",
		"discount",
		"replacement",
		"traffic",
		"speed_limit",
		"device_limit",
		"quota",
		"nodes",
		"node_tags",
		"show_original_price",
		"reset_cycle",
		"renewal_reset",
		"allow_renewal",
		"allow_reset_traffic",
	}
	if !reflect.DeepEqual(subscribeOrderSnapshotColumns, want) {
		t.Fatalf("subscribe order snapshot columns = %#v, want %#v", subscribeOrderSnapshotColumns, want)
	}
}

func TestSubscribeSubscriptionExpiryColumnsStayMinimal(t *testing.T) {
	want := []string{"id", "unit_time"}
	if !reflect.DeepEqual(subscribeSubscriptionExpiryColumns, want) {
		t.Fatalf("subscribe subscription expiry columns = %#v, want %#v", subscribeSubscriptionExpiryColumns, want)
	}
}

func TestSubscribeUpdateDefaultsColumnsStayMinimal(t *testing.T) {
	want := []string{
		"id",
		"show",
		"user_visible",
		"sell",
		"allow_deduction",
		"allow_renewal",
		"allow_reset_traffic",
		"renewal_reset",
	}
	if !reflect.DeepEqual(subscribeUpdateDefaultsColumns, want) {
		t.Fatalf("subscribe update default columns = %#v, want %#v", subscribeUpdateDefaultsColumns, want)
	}
}

func TestSubscribePublicListColumnsStayPublicProductScoped(t *testing.T) {
	columns := map[string]struct{}{}
	for _, column := range subscribePublicListColumns {
		columns[column] = struct{}{}
	}
	required := []string{
		"id",
		"name",
		"description",
		"unit_price",
		"unit_time",
		"discount",
		"replacement",
		"inventory",
		"traffic",
		"speed_limit",
		"device_limit",
		"user_visible",
		"sell",
		"show_original_price",
		"display_template",
		"badge",
		"min_quantity",
		"max_quantity",
		"sale_starts_at",
		"sale_ends_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("subscribe public list columns must include %q", column)
		}
	}
	forbidden := []string{
		"language",
		"quota",
		"show",
		"sort",
		"deduction_ratio",
		"allow_deduction",
		"allow_renewal",
		"allow_reset_traffic",
		"reset_cycle",
		"renewal_reset",
		"created_at",
		"updated_at",
		"nodes",
		"node_tags",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("subscribe public list columns must not include %q", column)
		}
	}
	if !reflect.DeepEqual(subscribeFilterListColumns(FilterListProjectionPublic), subscribePublicListColumns) {
		t.Fatalf("public filter projection should use subscribePublicListColumns")
	}
	if got := subscribeFilterListColumns(FilterListProjectionFull); got != nil {
		t.Fatalf("full filter projection should not force select columns: %#v", got)
	}
}

func TestSubscribeAdminListColumnsOmitDetailFields(t *testing.T) {
	columns := map[string]struct{}{}
	for _, column := range subscribeAdminListColumns {
		columns[column] = struct{}{}
	}
	required := []string{
		"id",
		"name",
		"language",
		"unit_price",
		"unit_time",
		"replacement",
		"inventory",
		"traffic",
		"speed_limit",
		"device_limit",
		"quota",
		"show",
		"user_visible",
		"sell",
		"allow_renewal",
		"reset_cycle",
		"display_template",
		"badge",
		"min_quantity",
		"max_quantity",
		"sale_starts_at",
		"sale_ends_at",
		"sort",
		"created_at",
		"updated_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("subscribe admin list columns must include %q", column)
		}
	}
	forbidden := []string{
		"description",
		"discount",
		"nodes",
		"node_tags",
		"deduction_ratio",
		"allow_deduction",
		"allow_reset_traffic",
		"renewal_reset",
		"show_original_price",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("subscribe admin list columns must not include %q", column)
		}
	}
	if !reflect.DeepEqual(subscribeFilterListColumns(FilterListProjectionAdmin), subscribeAdminListColumns) {
		t.Fatalf("admin filter projection should use subscribeAdminListColumns")
	}
}

func TestFilterListPublicProjectionDoesNotLoadTopology(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	now := time.Now().UTC()
	if err := db.Create(&Subscribe{
		Id:                12,
		Name:              "public-list",
		Language:          "en",
		Description:       "visible public plan",
		UnitPrice:         900,
		UnitTime:          "month",
		Discount:          `[{"quantity":12,"discount":80}]`,
		Replacement:       2,
		Inventory:         99,
		Traffic:           2048,
		SpeedLimit:        20,
		DeviceLimit:       4,
		Quota:             8,
		Nodes:             "101,102",
		NodeTags:          "edge,premium",
		Show:              &enabled,
		UserVisible:       &enabled,
		Sell:              &enabled,
		Sort:              1,
		DeductionRatio:    50,
		AllowDeduction:    &enabled,
		AllowRenewal:      &enabled,
		AllowResetTraffic: &enabled,
		ResetCycle:        1,
		RenewalReset:      &enabled,
		ShowOriginalPrice: true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	total, list, err := NewModel(db, nil).FilterList(ctx, &FilterParams{
		Page:       1,
		Size:       10,
		Projection: FilterListProjectionPublic,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("public list total/list = %d/%d, want 1/1", total, len(list))
	}
	if cap(list) != 1 {
		t.Fatalf("public list cap = %d, want 1", cap(list))
	}
	got := list[0]
	if got.Id != 12 || got.Name != "public-list" || got.Description != "visible public plan" || got.UnitPrice != 900 || got.Traffic != 2048 {
		t.Fatalf("public projection dropped response fields: %#v", got)
	}
	if got.Nodes != "" || got.NodeTags != "" {
		t.Fatalf("public projection loaded topology fields: nodes=%q tags=%q", got.Nodes, got.NodeTags)
	}
	if got.Show != nil {
		t.Fatalf("public projection should not load portal-only show flag: %#v", got)
	}
	if got.UserVisible == nil || !*got.UserVisible || got.Sell == nil || !*got.Sell {
		t.Fatalf("public projection dropped visibility flags: %#v", got)
	}
}

func TestFilterListSkipCountUsesResultLengthAndKeepsLanguageFallback(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	rows := []*Subscribe{
		{Id: 21, Name: "fallback-a", Language: "", UnitPrice: 100, Show: &enabled, UserVisible: &enabled, Sell: &enabled, Sort: 2},
		{Id: 22, Name: "fallback-b", Language: "", UnitPrice: 200, Show: &enabled, UserVisible: &enabled, Sell: &enabled, Sort: 1},
		{Id: 23, Name: "hidden", Language: "fr", UnitPrice: 300, Show: &enabled, UserVisible: &enabled, Sell: &enabled, Sort: 3},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	total, list, err := NewModel(db, nil).FilterList(ctx, &FilterParams{
		Page:            1,
		Size:            10,
		Language:        "zh-CN",
		DefaultLanguage: true,
		Show:            true,
		UserVisible:     true,
		Sell:            true,
		Projection:      FilterListProjectionPublic,
		SkipCount:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != int64(len(list)) || total != 2 {
		t.Fatalf("skip count total/list = %d/%d, want 2/2", total, len(list))
	}
	if cap(list) != 10 {
		t.Fatalf("skip count list cap = %d, want 10", cap(list))
	}
	if list[0].Id != 22 || list[1].Id != 21 {
		t.Fatalf("fallback list order = %#v", []int64{list[0].Id, list[1].Id})
	}
}

func TestFilterListAdminProjectionDoesNotLoadDetailFields(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	now := time.Now().UTC()
	if err := db.Create(&Subscribe{
		Id:                13,
		Name:              "admin-list",
		Language:          "en",
		Description:       "heavy admin plan",
		UnitPrice:         1100,
		UnitTime:          "month",
		Discount:          `[{"quantity":12,"discount":80}]`,
		Replacement:       3,
		Inventory:         88,
		Traffic:           4096,
		SpeedLimit:        30,
		DeviceLimit:       5,
		Quota:             9,
		Nodes:             "201,202",
		NodeTags:          "edge,premium",
		Show:              &enabled,
		UserVisible:       &enabled,
		Sell:              &enabled,
		Sort:              2,
		DeductionRatio:    60,
		AllowDeduction:    &enabled,
		AllowRenewal:      &enabled,
		AllowResetTraffic: &enabled,
		ResetCycle:        1,
		RenewalReset:      &enabled,
		ShowOriginalPrice: true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	total, list, err := NewModel(db, nil).FilterList(ctx, &FilterParams{
		Page:       1,
		Size:       10,
		Projection: FilterListProjectionAdmin,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("admin list total/list = %d/%d, want 1/1", total, len(list))
	}
	got := list[0]
	if got.Id != 13 || got.Name != "admin-list" || got.Language != "en" || got.UnitPrice != 1100 || got.UnitTime != "month" || got.Replacement != 3 || got.Inventory != 88 || got.Traffic != 4096 || got.SpeedLimit != 30 || got.DeviceLimit != 5 || got.Quota != 9 || got.ResetCycle != 1 {
		t.Fatalf("admin projection dropped list fields: %#v", got)
	}
	if got.Show == nil || !*got.Show || got.UserVisible == nil || !*got.UserVisible || got.Sell == nil || !*got.Sell || got.AllowRenewal == nil || !*got.AllowRenewal || got.Sort != 2 || got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Fatalf("admin projection dropped state fields: %#v", got)
	}
	if got.Description != "" || got.Discount != "" || got.Nodes != "" || got.NodeTags != "" || got.DeductionRatio != 0 || got.AllowDeduction != nil || got.AllowResetTraffic != nil || got.RenewalReset != nil || got.ShowOriginalPrice {
		t.Fatalf("admin projection loaded detail fields: %#v", got)
	}
}

func TestFilterParamsNormalizeDeduplicatesSelectors(t *testing.T) {
	params := &FilterParams{
		Page:    -1,
		Size:    0,
		MaxSize: 50,
		Ids:     []int64{3, 0, 2, 3, -1},
		Node:    []int64{10, 10, 0, 11, -2},
		Tags:    []string{" edge ", "", "premium", "edge", "  "},
	}

	params.Normalize()

	if params.Page != 1 || params.Size != 10 {
		t.Fatalf("pagination = page %d size %d, want page 1 size 10", params.Page, params.Size)
	}
	if want := []int64{3, 2}; !reflect.DeepEqual(params.Ids, want) {
		t.Fatalf("ids = %#v, want %#v", params.Ids, want)
	}
	if want := []int64{10, 11}; !reflect.DeepEqual(params.Node, want) {
		t.Fatalf("node ids = %#v, want %#v", params.Node, want)
	}
	if want := []string{"edge", "premium"}; !reflect.DeepEqual(params.Tags, want) {
		t.Fatalf("tags = %#v, want %#v", params.Tags, want)
	}
}

func TestFilterParamsNormalizeKeepsSparseNodeIDsCompact(t *testing.T) {
	nodeIDs := make([]int64, subscribeSmallSetLinearDedupLimit+20)
	nodeIDs[1] = 10
	nodeIDs[subscribeSmallSetLinearDedupLimit+10] = 10
	params := &FilterParams{Node: nodeIDs}

	params.Normalize()

	if want := []int64{10}; !reflect.DeepEqual(params.Node, want) {
		t.Fatalf("node ids = %#v, want %#v", params.Node, want)
	}
	if maxCap := subscribeSmallSetLinearDedupLimit + 1; cap(params.Node) > maxCap {
		t.Fatalf("node ids cap = %d, want <= %d", cap(params.Node), maxCap)
	}
}

func TestSubscribeCacheNodeServerColumnsStayLightweight(t *testing.T) {
	if !reflect.DeepEqual(subscribeCacheNodeServerColumns, []string{"DISTINCT server_id"}) {
		t.Fatalf("subscribe cache node server columns = %#v, want DISTINCT server_id only", subscribeCacheNodeServerColumns)
	}
}

func TestSubscribeGroupListColumnsMatchAPIShape(t *testing.T) {
	want := []string{"id", "name", "description", "created_at", "updated_at"}
	if !reflect.DeepEqual(subscribeGroupListColumns, want) {
		t.Fatalf("subscribe group list columns = %#v, want %#v", subscribeGroupListColumns, want)
	}
}

func TestQueryGroupListReturnsOrderedRows(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Group{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*Group{
		{Id: 3, Name: "team", Description: "team users"},
		{Id: 1, Name: "basic", Description: "basic users"},
	}).Error; err != nil {
		t.Fatal(err)
	}

	list, err := NewModel(db, nil).QueryGroupList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]int64, 0, len(list))
	for _, item := range list {
		got = append(got, item.Id)
	}
	if want := []int64{1, 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("group ids = %#v, want %#v", got, want)
	}
	if list[0].Name != "basic" || list[0].Description != "basic users" {
		t.Fatalf("first group = %#v, want basic group data", list[0])
	}
}

func TestSubscribeListQueriesKeepStableEmptySlices(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}, &Group{}); err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil).(*customSubscribeModel)
	groups, err := model.QueryGroupList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if groups == nil || len(groups) != 0 {
		t.Fatalf("subscribe groups = %#v, want non-nil empty slice", groups)
	}

	total, list, err := model.FilterList(ctx, &FilterParams{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || list == nil || len(list) != 0 {
		t.Fatalf("subscribe list = total %d list %#v, want total 0 and non-nil empty slice", total, list)
	}

	cacheRows, err := model.querySubscribeCacheRows(ctx, []int64{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	if cacheRows == nil || len(cacheRows) != 0 {
		t.Fatalf("subscribe cache rows = %#v, want non-nil empty slice", cacheRows)
	}
}

func TestDefaultSubscribeCacheKeysUseDistinctServerKeys(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}, &node.Node{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*node.Node{
		{Id: 10, Name: "node-a", ServerId: 101, Tags: "edge,shared", Protocol: "vless", InboundAlias: "vless", Sort: 10},
		{Id: 11, Name: "node-b", ServerId: 101, Tags: "shared", Protocol: "vless", InboundAlias: "vless", Sort: 11},
		{Id: 12, Name: "node-c", ServerId: 102, Tags: "edge", Protocol: "vless", InboundAlias: "vless", Sort: 12},
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil).(*customSubscribeModel)
	keys := model.getCacheKeys(&Subscribe{
		Id:    7,
		Nodes: "10,11,12",
	})

	want := map[string]struct{}{
		cacheSubscribeIdPrefix + "7":                          {},
		cacheSubscribeRuntimePrefix + "7":                     {},
		cacheSubscribeNodeSelectorPrefix + "7":                {},
		cacheSubscribeOrderSnapshotPrefix + "7":               {},
		fmt.Sprintf("%s%d", node.ServerUserListCacheKey, 101): {},
		fmt.Sprintf("%s%d", node.ServerUserListCacheKey, 102): {},
	}
	if len(keys) != len(want) {
		t.Fatalf("cache keys = %#v, want %d unique keys", keys, len(want))
	}
	for _, key := range keys {
		if _, ok := want[key]; !ok {
			t.Fatalf("unexpected cache key %q in %#v", key, keys)
		}
	}
}

func TestBatchSubscribeCacheKeysDeduplicatesBaseKeys(t *testing.T) {
	keys := (&defaultSubscribeModel{}).batchGetCacheKeys(
		&Subscribe{Id: 7},
		&Subscribe{Id: 7},
		nil,
	)

	want := map[string]struct{}{
		cacheSubscribeIdPrefix + "7":            {},
		cacheSubscribeRuntimePrefix + "7":       {},
		cacheSubscribeNodeSelectorPrefix + "7":  {},
		cacheSubscribeOrderSnapshotPrefix + "7": {},
	}
	if len(keys) != len(want) {
		t.Fatalf("cache keys = %#v, want %d unique keys", keys, len(want))
	}
	for _, key := range keys {
		if _, ok := want[key]; !ok {
			t.Fatalf("unexpected cache key %q in %#v", key, keys)
		}
	}
}

func TestSubscribeGetCacheKeysSkipsInvalidID(t *testing.T) {
	keys := (&defaultSubscribeModel{}).getCacheKeys(&Subscribe{Id: 0})
	if keys == nil || len(keys) != 0 {
		t.Fatalf("invalid subscribe id cache keys = %#v, want non-nil empty slice", keys)
	}
}

func TestCacheKeysForSubscribesDeduplicatesFetchedAndMissingIDs(t *testing.T) {
	keys, err := (&customSubscribeModel{}).cacheKeysForSubscribes(
		context.Background(),
		[]int64{7, 8, 8},
		[]*Subscribe{{Id: 7}, {Id: 7}, nil},
	)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]struct{}{
		cacheSubscribeIdPrefix + "7":            {},
		cacheSubscribeRuntimePrefix + "7":       {},
		cacheSubscribeNodeSelectorPrefix + "7":  {},
		cacheSubscribeOrderSnapshotPrefix + "7": {},
		cacheSubscribeIdPrefix + "8":            {},
		cacheSubscribeRuntimePrefix + "8":       {},
		cacheSubscribeNodeSelectorPrefix + "8":  {},
		cacheSubscribeOrderSnapshotPrefix + "8": {},
	}
	if len(keys) != len(want) {
		t.Fatalf("cache keys = %#v, want %d unique keys", keys, len(want))
	}
	for _, key := range keys {
		if _, ok := want[key]; !ok {
			t.Fatalf("unexpected cache key %q in %#v", key, keys)
		}
	}
}

func TestCacheKeysForSubscribesSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	ids := make([]int64, subscribeSmallSetLinearDedupLimit+20)
	ids[1] = 7
	ids[subscribeSmallSetLinearDedupLimit+10] = 7
	subscribes := make([]*Subscribe, subscribeSmallSetLinearDedupLimit+20)
	subscribes[2] = &Subscribe{Id: 0}
	subscribes[3] = &Subscribe{Id: 7}

	keys, err := (&customSubscribeModel{}).cacheKeysForSubscribes(context.Background(), ids, subscribes)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		cacheSubscribeIdPrefix + "7",
		cacheSubscribeRuntimePrefix + "7",
		cacheSubscribeNodeSelectorPrefix + "7",
		cacheSubscribeOrderSnapshotPrefix + "7",
	}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("cache keys = %#v, want %#v", keys, want)
	}
	if cap(keys) != subscribeSmallSetLinearDedupLimit+1 {
		t.Fatalf("cache key cap = %d, want small initial cap", cap(keys))
	}
}

func TestUniqueSubscribeCacheKeysDropsEmptyAndDuplicates(t *testing.T) {
	got := uniqueSubscribeCacheKeys([]string{"a", "", "b", "a", "b", "c"})
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueSubscribeCacheKeys() = %#v, want %#v", got, want)
	}
}

func TestUniqueSubscribeCacheKeysSmallInputAvoidsMapAllocation(t *testing.T) {
	keys := []string{"subscribe:id:1", "subscribe:list", "subscribe:id:1"}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = uniqueSubscribeCacheKeys(keys)
	})
	if allocs > 1 {
		t.Fatalf("uniqueSubscribeCacheKeys small input allocations = %f, want at most 1", allocs)
	}
}

func TestSubscribeCacheKeyCollectorSkipsEmptyAndDuplicatesWithoutInputSizedCapacity(t *testing.T) {
	collector := newSubscribeCacheKeyCollector(subscribeSmallSetLinearDedupLimit + 20)
	for i := 0; i < subscribeSmallSetLinearDedupLimit+20; i++ {
		collector.Add("")
		collector.Add("subscribe:id:7")
	}

	keys := collector.Values()
	if want := []string{"subscribe:id:7"}; !reflect.DeepEqual(keys, want) {
		t.Fatalf("cache keys = %#v, want %#v", keys, want)
	}
	if cap(keys) != subscribeSmallSetLinearDedupLimit+1 {
		t.Fatalf("cache key cap = %d, want small initial cap", cap(keys))
	}
}

func TestSubscribeCacheKeyCollectorBoundsSparseUpgradeCapacity(t *testing.T) {
	collector := newSubscribeCacheKeyCollector(subscribeSmallSetLinearDedupLimit + 40)
	for i := 0; i <= subscribeSmallSetLinearDedupLimit; i++ {
		collector.Add(fmt.Sprintf("subscribe:id:%d", i+1))
	}
	for i := 0; i < subscribeSmallSetLinearDedupLimit+40; i++ {
		collector.Add("subscribe:id:1")
	}

	keys := collector.Values()
	if collector.seen == nil {
		t.Fatal("cache key collector should allocate map after actual unique keys exceed threshold")
	}
	if maxCap := len(keys) * 2; cap(keys) > maxCap {
		t.Fatalf("cache key cap = %d, want <= %d", cap(keys), maxCap)
	}
}

func TestSubscribeNodeSelectorSetNormalizesRawSelectors(t *testing.T) {
	selectors := newSubscribeNodeSelectorSet(2)
	selectors.AddRaw("10, 0, bad, 11,10, -1", " edge, ,shared,edge ")

	if want := []int64{10, 11}; !reflect.DeepEqual(selectors.NodeIDs(), want) {
		t.Fatalf("node ids = %#v, want %#v", selectors.NodeIDs(), want)
	}
	if want := []string{"edge", "shared"}; !reflect.DeepEqual(selectors.Tags(), want) {
		t.Fatalf("tags = %#v, want %#v", selectors.Tags(), want)
	}
	if selectors.Len() != 4 {
		t.Fatalf("selector length = %d, want 4", selectors.Len())
	}
}

func TestSubscribeNodeSelectorSetSmallInputAvoidsMapDedup(t *testing.T) {
	selectors := newSubscribeNodeSelectorSet(2)
	if selectors.nodeIDs != nil || selectors.tags != nil {
		t.Fatalf("small hinted node selector should not allocate before first value: %#v", selectors)
	}
	selectors.AddRaw("10, 11, 10", "edge,edge,shared")

	if selectors.seenNodeIDs != nil {
		t.Fatal("small node selector set should not allocate node id map")
	}
	if selectors.seenTags != nil {
		t.Fatal("small node selector set should not allocate tag map")
	}
	if want := []int64{10, 11}; !reflect.DeepEqual(selectors.NodeIDs(), want) {
		t.Fatalf("node ids = %#v, want %#v", selectors.NodeIDs(), want)
	}
	if want := []string{"edge", "shared"}; !reflect.DeepEqual(selectors.Tags(), want) {
		t.Fatalf("tags = %#v, want %#v", selectors.Tags(), want)
	}
	if cap(selectors.NodeIDs()) != 2 {
		t.Fatalf("small node selector id cap = %d, want 2", cap(selectors.NodeIDs()))
	}
	if cap(selectors.Tags()) != 2 {
		t.Fatalf("small node selector tag cap = %d, want 2", cap(selectors.Tags()))
	}
}

func TestSubscribeNodeSelectorSetUpgradesLargeInputToMap(t *testing.T) {
	selectors := newSubscribeNodeSelectorSet(subscribeSmallSetLinearDedupLimit + 1)
	if selectors.nodeIDs != nil || selectors.tags != nil || selectors.seenNodeIDs != nil || selectors.seenTags != nil {
		t.Fatalf("large node selector set should not allocate storage before first use: %#v", selectors)
	}
	selectors.AddNodeID(1)
	selectors.AddNodeID(1)
	if selectors.seenNodeIDs != nil || len(selectors.NodeIDs()) != 1 {
		t.Fatalf("duplicate first node id should stay on linear path: %#v", selectors)
	}
	if selectors.seenTags != nil {
		t.Fatal("large node selector set should not allocate tag map before first tag")
	}
	for i := 2; i <= subscribeSmallSetLinearDedupLimit+1; i++ {
		selectors.AddNodeID(int64(i))
		selectors.AddTag(fmt.Sprintf("tag-%d", i))
	}
	selectors.AddTag("tag-1")
	selectors.AddNodeID(1)
	selectors.AddTag("tag-1")

	if selectors.seenNodeIDs == nil {
		t.Fatal("large node selector set should upgrade to node id map after actual unique ids exceed threshold")
	}
	if selectors.seenTags == nil {
		t.Fatal("large node selector set should upgrade to tag map after actual unique tags exceed threshold")
	}
	if len(selectors.NodeIDs()) != subscribeSmallSetLinearDedupLimit+1 {
		t.Fatalf("node ids = %#v, want %d unique ids", selectors.NodeIDs(), subscribeSmallSetLinearDedupLimit+1)
	}
	if len(selectors.Tags()) != subscribeSmallSetLinearDedupLimit+1 {
		t.Fatalf("tags = %#v, want %d unique tags", selectors.Tags(), subscribeSmallSetLinearDedupLimit+1)
	}
	if cap(selectors.NodeIDs()) != subscribeSmallSetLinearDedupLimit+1 {
		t.Fatalf("large node selector id cap = %d, want compact upgrade cap", cap(selectors.NodeIDs()))
	}
	if cap(selectors.Tags()) != subscribeSmallSetLinearDedupLimit+1 {
		t.Fatalf("large node selector tag cap = %d, want compact upgrade cap", cap(selectors.Tags()))
	}
}

func TestSubscribeNodeSelectorSetUsesSeparateCapacityHints(t *testing.T) {
	selectors := newSubscribeNodeSelectorSetWithHints(subscribeSmallSetLinearDedupLimit+1, 1)
	if selectors.nodeIDs != nil || selectors.tags != nil {
		t.Fatalf("separately hinted selector should not allocate before first value: %#v", selectors)
	}
	selectors.AddNodeID(10)
	selectors.AddNodeID(10)
	if selectors.seenNodeIDs != nil || len(selectors.NodeIDs()) != 1 {
		t.Fatalf("large hinted duplicate node selector should stay linear: %#v", selectors)
	}
	selectors.AddTag("edge")
	if selectors.seenTags != nil {
		t.Fatal("small hinted tag selector should not allocate tag map")
	}
	if cap(selectors.Tags()) != 1 {
		t.Fatalf("node selector tag cap = %d, want 1", cap(selectors.Tags()))
	}
	for i := 11; i <= subscribeSmallSetLinearDedupLimit+10; i++ {
		selectors.AddNodeID(int64(i))
	}
	if selectors.seenNodeIDs == nil {
		t.Fatal("large hinted node selector should allocate node id map after actual unique ids exceed threshold")
	}
	if cap(selectors.NodeIDs()) != subscribeSmallSetLinearDedupLimit+1 {
		t.Fatalf("node selector id cap = %d, want compact upgrade cap", cap(selectors.NodeIDs()))
	}
}

func TestSubscribeNodeSelectorSetBoundsSparseUpgradeCapacity(t *testing.T) {
	selectors := newSubscribeNodeSelectorSetWithHints(subscribeSmallSetLinearDedupLimit+40, subscribeSmallSetLinearDedupLimit+40)
	for i := 1; i <= subscribeSmallSetLinearDedupLimit+1; i++ {
		selectors.AddNodeID(int64(i))
		selectors.AddTag(fmt.Sprintf("tag-%d", i))
	}
	for i := 0; i < subscribeSmallSetLinearDedupLimit+40; i++ {
		selectors.AddNodeID(1)
		selectors.AddTag("tag-1")
	}

	if selectors.seenNodeIDs == nil || selectors.seenTags == nil {
		t.Fatalf("selectors should upgrade after actual unique values exceed threshold: %#v", selectors)
	}
	if maxCap := (subscribeSmallSetLinearDedupLimit + 1) * 2; cap(selectors.NodeIDs()) > maxCap {
		t.Fatalf("node selector id cap = %d, want <= %d", cap(selectors.NodeIDs()), maxCap)
	}
	if maxCap := (subscribeSmallSetLinearDedupLimit + 1) * 2; cap(selectors.Tags()) > maxCap {
		t.Fatalf("node selector tag cap = %d, want <= %d", cap(selectors.Tags()), maxCap)
	}
}

func TestSubscribeSeenIDSetUsesLinearDedupForSmallInput(t *testing.T) {
	var ids []int64
	var index map[int64]struct{}
	for _, id := range []int64{0, 7, -1, 8, 7} {
		ids, index = appendSubscribeSeenID(ids, index, id)
	}

	if index != nil {
		t.Fatal("small subscribe id set should not allocate map")
	}
	if want := []int64{7, 8}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("seen ids = %#v, want %#v", ids, want)
	}
	if !containsSubscribeSeenID(ids, index, 8) {
		t.Fatal("expected seen id lookup to find 8")
	}
	if containsSubscribeSeenID(ids, index, 9) {
		t.Fatal("expected seen id lookup to reject missing id 9")
	}
}

func TestSubscribeSeenIDSetUpgradesLargeInputToMap(t *testing.T) {
	var ids []int64
	var index map[int64]struct{}
	for i := 1; i <= subscribeSmallSetLinearDedupLimit+1; i++ {
		ids, index = appendSubscribeSeenID(ids, index, int64(i))
	}
	ids, index = appendSubscribeSeenID(ids, index, 1)

	if index == nil {
		t.Fatal("large subscribe id set should upgrade to map")
	}
	if len(ids) != subscribeSmallSetLinearDedupLimit+1 {
		t.Fatalf("seen ids = %#v, want %d unique ids", ids, subscribeSmallSetLinearDedupLimit+1)
	}
}

func TestSubscribeSeenIDSetAvoidsMapForLargeDuplicateInput(t *testing.T) {
	set := newSubscribeSeenIDSet(subscribeSmallSetLinearDedupLimit + 20)
	for i := 0; i < subscribeSmallSetLinearDedupLimit+20; i++ {
		set.Add(7)
	}

	if set.index != nil {
		t.Fatal("duplicate-heavy subscribe id set should not allocate map")
	}
	if !reflect.DeepEqual(set.ids, []int64{7}) {
		t.Fatalf("seen ids = %#v, want [7]", set.ids)
	}
	if cap(set.ids) != subscribeSmallSetLinearDedupLimit+1 {
		t.Fatalf("seen ids cap = %d, want %d", cap(set.ids), subscribeSmallSetLinearDedupLimit+1)
	}
}

func TestSubscribeSeenIDSetBoundsSparseUpgradeCapacity(t *testing.T) {
	set := newSubscribeSeenIDSet(subscribeSmallSetLinearDedupLimit + 20)
	for i := 1; i <= subscribeSmallSetLinearDedupLimit+1; i++ {
		set.Add(int64(i))
	}
	for i := 0; i < subscribeSmallSetLinearDedupLimit+20; i++ {
		set.Add(1)
	}

	if set.index == nil {
		t.Fatal("subscribe id set should allocate map after actual unique ids exceed threshold")
	}
	if len(set.ids) != subscribeSmallSetLinearDedupLimit+1 {
		t.Fatalf("seen ids len = %d, want %d", len(set.ids), subscribeSmallSetLinearDedupLimit+1)
	}
	if maxCap := (subscribeSmallSetLinearDedupLimit + 1) * 2; cap(set.ids) > maxCap {
		t.Fatalf("seen ids cap = %d, want <= %d", cap(set.ids), maxCap)
	}
	if !set.Contains(int64(subscribeSmallSetLinearDedupLimit+1)) || set.Contains(99) {
		t.Fatalf("contains lookup mismatch: ids=%#v index=%#v", set.ids, set.index)
	}
}

func TestNormalizeNodeSelectors(t *testing.T) {
	nodeIDs, tags := NormalizeNodeSelectors("1, 2, bad, 1, 0", " premium, ,shared,premium ")
	if want := []int64{1, 2}; !reflect.DeepEqual(nodeIDs, want) {
		t.Fatalf("node ids = %#v, want %#v", nodeIDs, want)
	}
	if want := []string{"premium", "shared"}; !reflect.DeepEqual(tags, want) {
		t.Fatalf("tags = %#v, want %#v", tags, want)
	}
}

func TestNormalizeNodeSelectorsUsesCSVCapacityHints(t *testing.T) {
	nodeIDs, tags := NormalizeNodeSelectors("1,2,3,4,5,6,7,8,9", "premium")
	if cap(nodeIDs) != subscribeSmallSetLinearDedupLimit+1 {
		t.Fatalf("node ids cap = %d, want %d", cap(nodeIDs), subscribeSmallSetLinearDedupLimit+1)
	}
	if cap(tags) != 1 {
		t.Fatalf("tags cap = %d, want 1", cap(tags))
	}
	if len(nodeIDs) != subscribeSmallSetLinearDedupLimit+1 || len(tags) != 1 {
		t.Fatalf("selectors = %#v/%#v", nodeIDs, tags)
	}
}

func TestSubscribeListSelectorCapHints(t *testing.T) {
	nodeHint, tagHint := subscribeListSelectorCapHints([]*Subscribe{
		nil,
		{Nodes: "1,2,3", NodeTags: "edge"},
		{Nodes: "", NodeTags: "premium,shared"},
		{Nodes: "4", NodeTags: " "},
	})
	if nodeHint != 4 || tagHint != 3 {
		t.Fatalf("selector cap hints = %d/%d, want 4/3", nodeHint, tagHint)
	}
}

func TestSubscribeCacheNodeServerQueryCombinesNodeAndTagSelectors(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}

	query := buildSubscribeCacheNodeServerQuery(db, []int64{10, 11}, []string{"edge", "shared"}).Find(&[]subscribeCacheNodeServerRow{})
	sql := query.Statement.SQL.String()
	if !strings.Contains(sql, "DISTINCT server_id") {
		t.Fatalf("query SQL = %q, want distinct server_id projection", sql)
	}
	if !strings.Contains(sql, "id IN") || !strings.Contains(sql, " OR string_to_array") {
		t.Fatalf("query SQL = %q, want a single combined id/tag selector", sql)
	}
}

func TestServerUserCacheKeysForSubscribeUsesDistinctServerKeys(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}, &node.Node{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*node.Node{
		{Id: 20, Name: "node-a", ServerId: 201, Protocol: "vless", InboundAlias: "vless", Sort: 20},
		{Id: 21, Name: "node-b", ServerId: 201, Protocol: "vless", InboundAlias: "vless", Sort: 21},
		{Id: 22, Name: "node-c", ServerId: 202, Protocol: "vless", InboundAlias: "vless", Sort: 22},
	}).Error; err != nil {
		t.Fatal(err)
	}

	keys, err := ServerUserCacheKeysForSubscribe(ctx, db, &Subscribe{Nodes: "20, 21,20, 22"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct{}{
		fmt.Sprintf("%s%d", node.ServerUserListCacheKey, 201): {},
		fmt.Sprintf("%s%d", node.ServerUserListCacheKey, 202): {},
	}
	if len(keys) != len(want) {
		t.Fatalf("cache keys = %#v, want %d unique keys", keys, len(want))
	}
	for _, key := range keys {
		if _, ok := want[key]; !ok {
			t.Fatalf("unexpected cache key %q in %#v", key, keys)
		}
	}
}

func TestSubscribeServerUserCacheKeysDeduplicatesRows(t *testing.T) {
	keys := subscribeServerUserCacheKeys([]subscribeCacheNodeServerRow{
		{ServerId: 201},
		{ServerId: 0},
		{ServerId: 201},
		{ServerId: 202},
	})
	want := []string{
		fmt.Sprintf("%s%d", node.ServerUserListCacheKey, 201),
		fmt.Sprintf("%s%d", node.ServerUserListCacheKey, 202),
	}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("subscribeServerUserCacheKeys() = %#v, want %#v", keys, want)
	}

	empty := subscribeServerUserCacheKeys([]subscribeCacheNodeServerRow{{ServerId: 0}})
	if empty == nil || len(empty) != 0 {
		t.Fatalf("invalid-only server rows = %#v, want non-nil empty slice", empty)
	}
}

func TestAddSubscribeServerUserCacheKeysDeduplicatesBeforeCallback(t *testing.T) {
	var keys []string
	addSubscribeServerUserCacheKeys(func(key string) {
		keys = append(keys, key)
	}, []subscribeCacheNodeServerRow{
		{ServerId: 201},
		{ServerId: 0},
		{ServerId: 201},
		{ServerId: 202},
		{ServerId: 202},
	})

	want := []string{
		fmt.Sprintf("%s%d", node.ServerUserListCacheKey, 201),
		fmt.Sprintf("%s%d", node.ServerUserListCacheKey, 202),
	}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("callback keys = %#v, want %#v", keys, want)
	}
}

func TestFindOneForRuntimeSelectsOnlyRuntimeColumnsAndUsesDedicatedCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Subscribe{
		Id:          8,
		Name:        "runtime",
		Description: "heavy description",
		Discount:    `[{"months":12,"discount":80}]`,
		UnitPrice:   1000,
		Traffic:     1024,
		SpeedLimit:  50,
		DeviceLimit: 3,
		Nodes:       "10,11",
		NodeTags:    "edge",
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	got, err := NewModel(db, rds).FindOneForRuntime(ctx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 8 || got.Name != "runtime" || got.Nodes != "10,11" || got.NodeTags != "edge" {
		t.Fatalf("runtime subscribe = %#v, want id/name/nodes/node_tags", got)
	}
	if got.Description != "" || got.Discount != "" || got.UnitPrice != 0 || got.Traffic != 0 || got.SpeedLimit != 0 || got.DeviceLimit != 0 {
		t.Fatalf("runtime subscribe loaded non-runtime fields: %#v", got)
	}
	cacheKey := cacheSubscribeRuntimePrefix + "8"
	raw, err := rds.Get(ctx, cacheKey).Result()
	if err != nil {
		t.Fatalf("runtime cache key %q missing: %v", cacheKey, err)
	}
	if strings.Contains(raw, "heavy description") || strings.Contains(raw, "1024") {
		t.Fatalf("runtime cache payload contains non-runtime data: %s", raw)
	}
	if rds.Exists(ctx, cacheSubscribeIdPrefix+"8").Val() != 0 {
		t.Fatal("runtime lookup must not populate the full subscribe cache key")
	}
}

func TestFindOneForNodeSelectionSelectsOnlySelectorColumnsAndUsesDedicatedCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Subscribe{
		Id:          9,
		Name:        "selector",
		Description: "heavy selector description",
		Discount:    `[{"months":6,"discount":90}]`,
		UnitPrice:   2000,
		Traffic:     2048,
		SpeedLimit:  100,
		DeviceLimit: 5,
		Nodes:       "12,13",
		NodeTags:    "premium",
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	got, err := NewModel(db, rds).FindOneForNodeSelection(ctx, 9)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 9 || got.Nodes != "12,13" || got.NodeTags != "premium" {
		t.Fatalf("node selector subscribe = %#v, want id/nodes/node_tags", got)
	}
	if got.Name != "" || got.Description != "" || got.Discount != "" || got.UnitPrice != 0 || got.Traffic != 0 || got.SpeedLimit != 0 || got.DeviceLimit != 0 {
		t.Fatalf("node selector subscribe loaded non-selector fields: %#v", got)
	}
	cacheKey := cacheSubscribeNodeSelectorPrefix + "9"
	raw, err := rds.Get(ctx, cacheKey).Result()
	if err != nil {
		t.Fatalf("node selector cache key %q missing: %v", cacheKey, err)
	}
	if strings.Contains(raw, "heavy selector description") || strings.Contains(raw, "selector") {
		t.Fatalf("node selector cache payload contains non-selector data: %s", raw)
	}
	if rds.Exists(ctx, cacheSubscribeIdPrefix+"9").Val() != 0 {
		t.Fatal("node selector lookup must not populate the full subscribe cache key")
	}
}

func TestFindOneForOrderSnapshotSelectsOnlySnapshotColumnsAndUsesDedicatedCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	now := time.Now().UTC()
	if err := db.Create(&Subscribe{
		Id:                13,
		Name:              "snapshot",
		Language:          "en",
		Description:       "snapshot description",
		UnitPrice:         1800,
		UnitTime:          "month",
		Discount:          `[{"months":12,"discount":80}]`,
		Replacement:       10,
		Inventory:         99,
		Traffic:           4096,
		SpeedLimit:        100,
		DeviceLimit:       6,
		Quota:             3,
		Nodes:             "10,11",
		NodeTags:          "edge",
		Show:              &enabled,
		UserVisible:       &enabled,
		Sell:              &enabled,
		Sort:              9,
		DeductionRatio:    70,
		AllowDeduction:    &enabled,
		AllowRenewal:      &enabled,
		AllowResetTraffic: &enabled,
		ResetCycle:        2,
		RenewalReset:      &enabled,
		ShowOriginalPrice: true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	got, err := NewModel(db, rds).FindOneForOrderSnapshot(ctx, 13)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 13 || got.Name != "snapshot" || got.Description != "snapshot description" || got.UnitPrice != 1800 || got.UnitTime != "month" || got.Discount == "" || got.Replacement != 10 || got.Traffic != 4096 || got.SpeedLimit != 100 || got.DeviceLimit != 6 || got.Quota != 3 || got.Nodes != "10,11" || got.NodeTags != "edge" || !got.ShowOriginalPrice || got.ResetCycle != 2 || got.RenewalReset == nil || !*got.RenewalReset || got.AllowRenewal == nil || !*got.AllowRenewal || got.AllowResetTraffic == nil || !*got.AllowResetTraffic {
		t.Fatalf("order snapshot subscribe dropped required fields: %#v", got)
	}
	if got.Language != "" || got.Inventory != 0 || got.Show != nil || got.UserVisible != nil || got.Sell != nil || got.Sort != 0 || got.DeductionRatio != 0 || got.AllowDeduction != nil || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("order snapshot subscribe loaded unrelated fields: %#v", got)
	}
	cacheKey := cacheSubscribeOrderSnapshotPrefix + "13"
	raw, err := rds.Get(ctx, cacheKey).Result()
	if err != nil {
		t.Fatalf("order snapshot cache key %q missing: %v", cacheKey, err)
	}
	if strings.Contains(raw, `"language"`) || strings.Contains(raw, `"inventory"`) || strings.Contains(raw, `"sort"`) {
		t.Fatalf("order snapshot cache payload contains non-snapshot data: %s", raw)
	}
	if rds.Exists(ctx, cacheSubscribeIdPrefix+"13").Val() != 0 {
		t.Fatal("order snapshot lookup must not populate the full subscribe cache key")
	}
}

func TestFindOneForSubscriptionExpirySelectsOnlyUnitTime(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	now := time.Now().UTC()
	if err := db.Create(&Subscribe{
		Id:                14,
		Name:              "expiry",
		Language:          "en",
		Description:       "heavy description",
		UnitPrice:         1800,
		UnitTime:          "month",
		Discount:          `[{"months":12,"discount":80}]`,
		Replacement:       10,
		Inventory:         99,
		Traffic:           4096,
		SpeedLimit:        100,
		DeviceLimit:       6,
		Quota:             3,
		Nodes:             "10,11",
		NodeTags:          "edge",
		Show:              &enabled,
		UserVisible:       &enabled,
		Sell:              &enabled,
		Sort:              9,
		DeductionRatio:    70,
		AllowDeduction:    &enabled,
		AllowRenewal:      &enabled,
		AllowResetTraffic: &enabled,
		ResetCycle:        2,
		RenewalReset:      &enabled,
		ShowOriginalPrice: true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindOneForSubscriptionExpiry(ctx, 14)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 14 || got.UnitTime != "month" {
		t.Fatalf("subscription expiry lookup dropped required fields: %#v", got)
	}
	if got.Name != "" || got.Language != "" || got.Description != "" || got.UnitPrice != 0 || got.Discount != "" || got.Replacement != 0 || got.Inventory != 0 || got.Traffic != 0 || got.SpeedLimit != 0 || got.DeviceLimit != 0 || got.Quota != 0 || got.Nodes != "" || got.NodeTags != "" || got.Show != nil || got.UserVisible != nil || got.Sell != nil || got.Sort != 0 || got.DeductionRatio != 0 || got.AllowDeduction != nil || got.AllowRenewal != nil || got.AllowResetTraffic != nil || got.ResetCycle != 0 || got.RenewalReset != nil || got.ShowOriginalPrice || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("subscription expiry lookup loaded unrelated fields: %#v", got)
	}
}

func TestFindOneForUpdateDefaultsSelectsOnlyDefaultFlags(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	disabled := false
	now := time.Now().UTC()
	if err := db.Create(&Subscribe{
		Id:                15,
		Name:              "defaults",
		Language:          "en",
		Description:       "heavy description",
		UnitPrice:         1800,
		UnitTime:          "month",
		Discount:          `[{"months":12,"discount":80}]`,
		Replacement:       10,
		Inventory:         99,
		Traffic:           4096,
		SpeedLimit:        100,
		DeviceLimit:       6,
		Quota:             3,
		Nodes:             "10,11",
		NodeTags:          "edge",
		Show:              &enabled,
		UserVisible:       &enabled,
		Sell:              &disabled,
		Sort:              9,
		DeductionRatio:    70,
		AllowDeduction:    &enabled,
		AllowRenewal:      &disabled,
		AllowResetTraffic: &enabled,
		ResetCycle:        2,
		RenewalReset:      &disabled,
		ShowOriginalPrice: true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindOneForUpdateDefaults(ctx, 15)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 15 || got.Show == nil || !*got.Show || got.UserVisible == nil || !*got.UserVisible || got.Sell == nil || *got.Sell || got.AllowDeduction == nil || !*got.AllowDeduction || got.AllowRenewal == nil || *got.AllowRenewal || got.AllowResetTraffic == nil || !*got.AllowResetTraffic || got.RenewalReset == nil || *got.RenewalReset {
		t.Fatalf("update defaults lookup dropped required flags: %#v", got)
	}
	if got.Name != "" || got.Language != "" || got.Description != "" || got.UnitPrice != 0 || got.UnitTime != "" || got.Discount != "" || got.Replacement != 0 || got.Inventory != 0 || got.Traffic != 0 || got.SpeedLimit != 0 || got.DeviceLimit != 0 || got.Quota != 0 || got.Nodes != "" || got.NodeTags != "" || got.Sort != 0 || got.DeductionRatio != 0 || got.ResetCycle != 0 || got.ShowOriginalPrice || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("update defaults lookup loaded unrelated fields: %#v", got)
	}
}

func TestFindOneForOrderActivationSelectsOnlyActivationColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	now := time.Now().UTC()
	if err := db.Create(&Subscribe{
		Id:                10,
		Name:              "activation",
		Language:          "en",
		Description:       "heavy activation description",
		UnitPrice:         1200,
		UnitTime:          "month",
		Discount:          `[{"months":12,"discount":80}]`,
		Replacement:       10,
		Inventory:         100,
		Traffic:           1024,
		SpeedLimit:        50,
		DeviceLimit:       3,
		Quota:             2,
		Nodes:             "10,11",
		NodeTags:          "edge",
		Show:              &enabled,
		UserVisible:       &enabled,
		Sell:              &enabled,
		Sort:              9,
		DeductionRatio:    70,
		AllowDeduction:    &enabled,
		AllowRenewal:      &enabled,
		AllowResetTraffic: &enabled,
		ResetCycle:        2,
		RenewalReset:      &enabled,
		ShowOriginalPrice: true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindOneForOrderActivation(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 10 || got.Name != "activation" || got.UnitTime != "month" || got.Traffic != 1024 || got.Nodes != "10,11" || got.NodeTags != "edge" || got.RenewalReset == nil || !*got.RenewalReset {
		t.Fatalf("activation subscribe dropped required fields: %#v", got)
	}
	if got.Language != "" || got.Description != "" || got.UnitPrice != 0 || got.Discount != "" || got.Replacement != 0 || got.Inventory != 0 || got.SpeedLimit != 0 || got.DeviceLimit != 0 || got.Quota != 0 || got.Show != nil || got.UserVisible != nil || got.Sell != nil || got.Sort != 0 || got.DeductionRatio != 0 || got.AllowDeduction != nil || got.AllowRenewal != nil || got.AllowResetTraffic != nil || got.ResetCycle != 0 || got.ShowOriginalPrice || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("activation subscribe loaded unrelated/list fields: %#v", got)
	}
}

func TestFindOneForPublicPurchaseSelectsOnlyPurchaseColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	now := time.Now().UTC()
	if err := db.Create(&Subscribe{
		Id:                11,
		Name:              "purchase",
		Language:          "en",
		Description:       "purchase description",
		UnitPrice:         1200,
		UnitTime:          "month",
		Discount:          `[{"months":12,"discount":80}]`,
		Replacement:       10,
		Inventory:         100,
		Traffic:           1024,
		SpeedLimit:        50,
		DeviceLimit:       3,
		Quota:             2,
		Nodes:             "10,11",
		NodeTags:          "edge",
		Show:              &enabled,
		UserVisible:       &enabled,
		Sell:              &enabled,
		Sort:              9,
		DeductionRatio:    70,
		AllowDeduction:    &enabled,
		AllowRenewal:      &enabled,
		AllowResetTraffic: &enabled,
		ResetCycle:        2,
		RenewalReset:      &enabled,
		ShowOriginalPrice: true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindOneForPublicPurchase(ctx, 11)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 11 || got.Name != "purchase" || got.Description != "purchase description" || got.UnitPrice != 1200 || got.UnitTime != "month" || got.Discount == "" || got.Replacement != 10 || got.Inventory != 100 || got.Traffic != 1024 || got.SpeedLimit != 50 || got.DeviceLimit != 3 || got.Quota != 2 || got.Nodes != "10,11" || got.NodeTags != "edge" || got.UserVisible == nil || !*got.UserVisible || got.Sell == nil || !*got.Sell || got.AllowRenewal == nil || !*got.AllowRenewal || got.AllowResetTraffic == nil || !*got.AllowResetTraffic || got.ResetCycle != 2 || got.RenewalReset == nil || !*got.RenewalReset || !got.ShowOriginalPrice {
		t.Fatalf("public purchase subscribe dropped required fields: %#v", got)
	}
	if got.Language != "" || got.Show != nil || got.Sort != 0 || got.DeductionRatio != 0 || got.AllowDeduction != nil || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("public purchase subscribe loaded unrelated fields: %#v", got)
	}
}

func TestFindOneForPortalPrePurchaseSelectsOnlyPreviewColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	now := time.Now().UTC()
	if err := db.Create(&Subscribe{
		Id:                12,
		Name:              "portal purchase",
		Language:          "en",
		Description:       "large portal description",
		UnitPrice:         1500,
		UnitTime:          "month",
		Discount:          `[{"months":6,"discount":90}]`,
		Replacement:       20,
		Inventory:         5,
		Traffic:           2048,
		SpeedLimit:        100,
		DeviceLimit:       6,
		Quota:             1,
		Nodes:             "10,11",
		NodeTags:          "edge",
		Show:              &enabled,
		UserVisible:       &enabled,
		Sell:              &enabled,
		Sort:              9,
		DeductionRatio:    70,
		AllowDeduction:    &enabled,
		AllowRenewal:      &enabled,
		AllowResetTraffic: &enabled,
		ResetCycle:        2,
		RenewalReset:      &enabled,
		ShowOriginalPrice: true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindOneForPortalPrePurchase(ctx, 12)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 12 || got.UnitPrice != 1500 || got.Discount == "" || got.Inventory != 5 || got.Show == nil || !*got.Show || got.Sell == nil || !*got.Sell {
		t.Fatalf("portal pre-purchase subscribe dropped required fields: %#v", got)
	}
	if got.Name != "" || got.Language != "" || got.Description != "" || got.UnitTime != "" || got.Replacement != 0 || got.Traffic != 0 || got.SpeedLimit != 0 || got.DeviceLimit != 0 || got.Quota != 0 || got.Nodes != "" || got.NodeTags != "" || got.UserVisible != nil || got.Sort != 0 || got.DeductionRatio != 0 || got.AllowDeduction != nil || got.AllowRenewal != nil || got.AllowResetTraffic != nil || got.ResetCycle != 0 || got.RenewalReset != nil || got.ShowOriginalPrice || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("portal pre-purchase subscribe loaded unrelated fields: %#v", got)
	}
}

func TestNormalizeSubscribeIDs(t *testing.T) {
	got := normalizeSubscribeIDs([]int64{2, 0, 2, -1, 3, 1, 3})
	want := []int64{2, 3, 1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeSubscribeIDs() = %#v, want %#v", got, want)
	}
}

func TestNormalizeSubscribeIDsSmallInputAvoidsMapAllocation(t *testing.T) {
	ids := []int64{2, 0, 2, -1, 3, 1, 3}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = normalizeSubscribeIDs(ids)
	})
	if allocs > 1 {
		t.Fatalf("normalizeSubscribeIDs small input allocations = %f, want at most 1", allocs)
	}
}

func TestNormalizeSubscribeIDsSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	ids := make([]int64, subscribeSmallSetLinearDedupLimit+20)
	ids[2] = 7
	ids[subscribeSmallSetLinearDedupLimit+10] = 7

	got := normalizeSubscribeIDs(ids)
	if want := []int64{7}; !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeSubscribeIDs() = %#v, want %#v", got, want)
	}
	if cap(got) != subscribeSmallSetLinearDedupLimit+1 {
		t.Fatalf("normalized id cap = %d, want small initial cap", cap(got))
	}
}

func TestUpdateClearsOldAndNewTopologyCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}, &node.Node{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*node.Node{
		{Id: 10, Name: "old-node", ServerId: 101, Protocol: "vless", InboundAlias: "vless", Sort: 1},
		{Id: 20, Name: "new-node", ServerId: 202, Protocol: "vless", InboundAlias: "vless", Sort: 2},
	}).Error; err != nil {
		t.Fatal(err)
	}
	enabled := true
	if err := db.Create(&Subscribe{
		Id:                30,
		Name:              "pro",
		UnitPrice:         1000,
		UnitTime:          "month",
		Nodes:             "10",
		Show:              &enabled,
		UserVisible:       &enabled,
		Sell:              &enabled,
		AllowDeduction:    &enabled,
		AllowRenewal:      &enabled,
		AllowResetTraffic: &enabled,
		RenewalReset:      &enabled,
		Sort:              1,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	cacheKeys := []string{
		cacheSubscribeIdPrefix + "30",
		cacheSubscribeRuntimePrefix + "30",
		cacheSubscribeNodeSelectorPrefix + "30",
		cacheSubscribeOrderSnapshotPrefix + "30",
		fmt.Sprintf("%s%d", node.ServerUserListCacheKey, 101),
		fmt.Sprintf("%s%d", node.ServerUserListCacheKey, 202),
	}
	for _, key := range cacheKeys {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	if err := NewModel(db, rds).Update(ctx, &Subscribe{
		Id:                30,
		Name:              "pro-updated",
		UnitPrice:         1200,
		UnitTime:          "month",
		Nodes:             "20",
		Show:              &enabled,
		UserVisible:       &enabled,
		Sell:              &enabled,
		AllowDeduction:    &enabled,
		AllowRenewal:      &enabled,
		AllowResetTraffic: &enabled,
		RenewalReset:      &enabled,
		Sort:              1,
	}); err != nil {
		t.Fatal(err)
	}

	var got Subscribe
	if err := db.First(&got, 30).Error; err != nil {
		t.Fatal(err)
	}
	if got.Nodes != "20" || got.Name != "pro-updated" {
		t.Fatalf("updated subscribe = %#v, want new nodes/name", got)
	}
	for _, key := range cacheKeys {
		if rds.Exists(ctx, key).Val() != 0 {
			t.Fatalf("cache key %q was not cleared", key)
		}
	}
}

func TestDeleteByIdsClearsCacheAndKeepsSortDense(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}, &node.Node{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*node.Node{
		{Id: 10, Name: "node-a", ServerId: 101, Protocol: "vless", InboundAlias: "vless", Sort: 1},
		{Id: 11, Name: "node-b", ServerId: 102, Protocol: "vless", InboundAlias: "vless", Sort: 2},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*Subscribe{
		{Id: 1, Name: "basic", Sort: 1},
		{Id: 2, Name: "pro", Nodes: "10", Sort: 2},
		{Id: 3, Name: "team", Nodes: "11", Sort: 3},
		{Id: 4, Name: "enterprise", Sort: 4},
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	model := NewModel(db, rds)

	cacheKeys := []string{
		cacheSubscribeIdPrefix + "2",
		cacheSubscribeIdPrefix + "3",
		cacheSubscribeRuntimePrefix + "2",
		cacheSubscribeRuntimePrefix + "3",
		cacheSubscribeNodeSelectorPrefix + "2",
		cacheSubscribeNodeSelectorPrefix + "3",
		cacheSubscribeOrderSnapshotPrefix + "2",
		cacheSubscribeOrderSnapshotPrefix + "3",
		fmt.Sprintf("%s%d", node.ServerUserListCacheKey, 101),
		fmt.Sprintf("%s%d", node.ServerUserListCacheKey, 102),
	}
	for _, key := range cacheKeys {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	if err := model.DeleteByIds(ctx, []int64{3, 2, 2}); err != nil {
		t.Fatal(err)
	}

	var remaining []Subscribe
	if err := db.Order("sort ASC").Find(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	gotIDs := make([]int64, 0, len(remaining))
	gotSorts := make([]int64, 0, len(remaining))
	for _, item := range remaining {
		gotIDs = append(gotIDs, item.Id)
		gotSorts = append(gotSorts, item.Sort)
	}
	if want := []int64{1, 4}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("remaining ids = %#v, want %#v", gotIDs, want)
	}
	if want := []int64{1, 2}; !reflect.DeepEqual(gotSorts, want) {
		t.Fatalf("remaining sorts = %#v, want %#v", gotSorts, want)
	}
	for _, key := range cacheKeys {
		if rds.Exists(ctx, key).Val() != 0 {
			t.Fatalf("cache key %q was not cleared", key)
		}
	}
}
