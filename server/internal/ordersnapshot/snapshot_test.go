package ordersnapshot

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/types"
)

func TestSubscribeTypeMapsSnapshotWithoutJSONRoundTrip(t *testing.T) {
	snapshot := Snapshot{
		Kind:              KindSubscribe,
		ID:                7,
		Name:              "Snapshot Plan",
		Description:       "captured product",
		UnitPrice:         1200,
		UnitTime:          "month",
		Discount:          []types.SubscribeDiscount{{Quantity: 12, Discount: 80}},
		Replacement:       100,
		Traffic:           2048,
		SpeedLimit:        300,
		DeviceLimit:       5,
		Quota:             2,
		Nodes:             []int64{10, 11},
		NodeTags:          []string{"edge", "vip"},
		ShowOriginalPrice: true,
		ResetCycle:        2,
		RenewalReset:      true,
		AllowRenewal:      true,
		AllowResetTraffic: true,
	}

	got := SubscribeType(snapshot)
	if got.Id != snapshot.ID || got.Name != snapshot.Name || got.Description != snapshot.Description || got.UnitPrice != snapshot.UnitPrice || got.UnitTime != snapshot.UnitTime {
		t.Fatalf("basic subscribe fields not mapped: %#v", got)
	}
	if len(got.Discount) != 1 || got.Discount[0].Quantity != 12 || got.Discount[0].Discount != 80 {
		t.Fatalf("discount not mapped: %#v", got.Discount)
	}
	if got.Replacement != snapshot.Replacement || got.Traffic != snapshot.Traffic || got.SpeedLimit != snapshot.SpeedLimit || got.DeviceLimit != snapshot.DeviceLimit || got.Quota != snapshot.Quota {
		t.Fatalf("resource fields not mapped: %#v", got)
	}
	if len(got.Nodes) != 2 || got.Nodes[0] != 10 || len(got.NodeTags) != 2 || got.NodeTags[0] != "edge" {
		t.Fatalf("topology fields not mapped: nodes=%#v tags=%#v", got.Nodes, got.NodeTags)
	}
	if !got.ShowOriginalPrice || got.ResetCycle != 2 || !got.RenewalReset || !got.AllowRenewal || !got.AllowResetTraffic {
		t.Fatalf("policy fields not mapped: %#v", got)
	}
	if got.Show || got.UserVisible || got.Sell || got.Sort != 0 || got.DeductionRatio != 0 || got.AllowDeduction || got.CreatedAt != 0 || got.UpdatedAt != 0 {
		t.Fatalf("snapshot mapping should not synthesize live list fields: %#v", got)
	}
}

func TestSubscribeTypeNormalizesSnapshotTopology(t *testing.T) {
	got := SubscribeType(Snapshot{
		Kind:     KindSubscribe,
		Name:     "Snapshot Plan",
		Nodes:    []int64{10, 0, 10, -1, 11},
		NodeTags: []string{" edge ", "", "vip", "edge"},
	})
	if want := []int64{10, 11}; !reflect.DeepEqual(got.Nodes, want) {
		t.Fatalf("nodes = %#v, want %#v", got.Nodes, want)
	}
	if want := []string{"edge", "vip"}; !reflect.DeepEqual(got.NodeTags, want) {
		t.Fatalf("node tags = %#v, want %#v", got.NodeTags, want)
	}
}

func TestSubscribeTypeKeepsArrayJSONShapeForEmptySnapshotFields(t *testing.T) {
	got := SubscribeType(Snapshot{
		Kind: KindSubscribe,
		Name: "Snapshot Plan",
	})
	if got.Discount == nil || got.Nodes == nil || got.NodeTags == nil {
		t.Fatalf("empty subscribe arrays should be non-nil: discount=%#v nodes=%#v node_tags=%#v", got.Discount, got.Nodes, got.NodeTags)
	}

	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal subscribe type: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal subscribe type: %v", err)
	}
	for _, key := range []string{"discount", "nodes", "node_tags"} {
		if decoded[key] == nil {
			t.Fatalf("%s must encode as [] instead of null: %s", key, payload)
		}
	}
}

func TestSubscribeTypeRejectsNonSubscribeSnapshot(t *testing.T) {
	got := SubscribeType(Snapshot{Kind: KindMembership, Name: "Member"})
	if got.Name != "" || got.Id != 0 {
		t.Fatalf("non-subscribe snapshot mapped to subscribe response: %#v", got)
	}
}

func TestProductNameParsesOnlySnapshotIdentity(t *testing.T) {
	raw := Marshal(Snapshot{
		Kind:        KindSubscribe,
		Name:        "Snapshot Plan",
		Description: "large content",
		Traffic:     4096,
		Nodes:       []int64{1, 2, 3},
		NodeTags:    []string{"edge", "vip"},
	})
	if got := ProductName(raw); got != "Snapshot Plan" {
		t.Fatalf("ProductName() = %q, want Snapshot Plan", got)
	}

	for _, raw := range []string{
		"",
		"{bad",
		`{"kind":"subscribe"}`,
		`{"name":"missing kind"}`,
	} {
		if got := ProductName(raw); got != "" {
			t.Fatalf("ProductName(%q) = %q, want empty", raw, got)
		}
	}
}

func TestIsSubscribeParsesOnlySnapshotIdentity(t *testing.T) {
	raw := Marshal(Snapshot{
		Kind:        KindSubscribe,
		Name:        "Snapshot Plan",
		Description: "large content",
		Traffic:     4096,
		Nodes:       []int64{1, 2, 3},
		NodeTags:    []string{"edge", "vip"},
	})
	if !IsSubscribe(raw) {
		t.Fatal("IsSubscribe rejected subscribe snapshot")
	}

	for _, raw := range []string{
		"",
		"{bad",
		`{"kind":"subscribe"}`,
		`{"name":"missing kind"}`,
		Marshal(Snapshot{Kind: KindMembership, Name: "Member"}),
	} {
		if IsSubscribe(raw) {
			t.Fatalf("IsSubscribe(%q) = true, want false", raw)
		}
	}
}

func TestParseCacheUsesNilFallbackForSingleCallers(t *testing.T) {
	raw := Marshal(Snapshot{Kind: KindSubscribe, Name: "Snapshot Plan"})
	if cache := NewParseCache(1); cache != nil {
		t.Fatalf("single item cache = %#v, want nil", cache)
	}

	var cache *ParseCache
	snapshot, ok := cache.Parse(raw)
	if !ok || snapshot.Name != "Snapshot Plan" {
		t.Fatalf("nil cache Parse() = %#v/%v", snapshot, ok)
	}
}

func TestParseCacheCachesValidAndInvalidSnapshots(t *testing.T) {
	cache := NewParseCache(2)
	raw := Marshal(Snapshot{Kind: KindSubscribe, Name: "Snapshot Plan"})

	first, ok := cache.Parse(raw)
	if !ok || first.Name != "Snapshot Plan" {
		t.Fatalf("first Parse() = %#v/%v", first, ok)
	}
	second, ok := cache.Parse(raw)
	if !ok || second.Name != "Snapshot Plan" {
		t.Fatalf("second Parse() = %#v/%v", second, ok)
	}
	if len(cache.rows) != 1 || cache.index != nil {
		t.Fatalf("small cache rows/index = %d/%#v, want one row and no map", len(cache.rows), cache.index)
	}

	if snapshot, ok := cache.Parse(`{"kind":"subscribe"}`); ok || snapshot.Name != "" {
		t.Fatalf("invalid snapshot Parse() = %#v/%v, want zero false", snapshot, ok)
	}
	if _, ok := cache.Parse(`{"kind":"subscribe"}`); ok {
		t.Fatal("invalid snapshot should stay invalid on cache hit")
	}
	if len(cache.rows) != 2 {
		t.Fatalf("invalid cache rows = %d, want 2", len(cache.rows))
	}
}

func TestParseCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := NewParseCache(parseCacheLinearLimit + 1)
	if cache == nil || cache.rows != nil || cache.index != nil {
		t.Fatalf("large cache should not preallocate storage from hint alone: %#v", cache)
	}
	raw := Marshal(Snapshot{Kind: KindSubscribe, Name: "Plan A"})
	first, ok := cache.Parse(" " + raw + " ")
	if !ok || first.Name != "Plan A" {
		t.Fatalf("space-padded snapshot Parse() = %#v/%v", first, ok)
	}
	second, ok := cache.Parse(" " + raw + " ")
	if !ok || second.Name != "Plan A" || len(cache.rows) != 1 || cache.index != nil {
		t.Fatalf("duplicate raw snapshot should stay on linear path: rows=%#v index=%#v", cache.rows, cache.index)
	}

	cache = NewParseCache(parseCacheLinearLimit + 1)
	for i := 0; i < parseCacheLinearLimit+1; i++ {
		raw := Marshal(Snapshot{Kind: KindSubscribe, Name: "Plan " + string(rune('a'+i))})
		snapshot, ok := cache.Parse(raw)
		if !ok || snapshot.Name == "" {
			t.Fatalf("Parse(%d) = %#v/%v", i, snapshot, ok)
		}
	}
	if len(cache.rows) != parseCacheLinearLimit+1 || len(cache.index) != parseCacheLinearLimit+1 {
		t.Fatalf("large cache rows/index = %d/%d", len(cache.rows), len(cache.index))
	}
}

func TestParseCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := NewParseCache(parseCacheLinearLimit + 20)
	for i := 0; i < parseCacheLinearLimit+1; i++ {
		raw := Marshal(Snapshot{Kind: KindSubscribe, Name: "Plan " + string(rune('a'+i))})
		snapshot, ok := cache.Parse(raw)
		if !ok || snapshot.Name == "" {
			t.Fatalf("Parse(%d) = %#v/%v", i, snapshot, ok)
		}
	}
	if cache.index == nil {
		t.Fatal("parse cache should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("parse cache row cap = %d, want at most %d", cap(cache.rows), maxCap)
	}
	raw := Marshal(Snapshot{Kind: KindSubscribe, Name: "Plan a"})
	for i := 0; i < parseCacheLinearLimit+20; i++ {
		cache.Parse(raw)
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("parse cache row cap after duplicates = %d, want at most %d", cap(cache.rows), maxCap)
	}
}

func TestMembershipPlanTypeMapsSnapshotWithoutJSONRoundTrip(t *testing.T) {
	snapshot := Snapshot{
		Kind:          KindMembership,
		ID:            9,
		Name:          "Member Pro",
		Description:   "member snapshot",
		UnitPrice:     1500,
		DurationUnit:  "month",
		DurationValue: 3,
		MembershipPolicy: types.MembershipPolicy{
			MonthlyOrderLimitEnabled: true,
			MonthlyOrderLimit:        5,
		},
	}

	got := MembershipPlanType(snapshot)
	if got == nil {
		t.Fatal("MembershipPlanType returned nil for membership snapshot")
	}
	if got.Id != snapshot.ID || got.Name != snapshot.Name || got.Description != snapshot.Description || got.UnitPrice != snapshot.UnitPrice {
		t.Fatalf("membership fields not mapped: %#v", got)
	}
	if got.DurationUnit != "month" || got.DurationValue != 3 || !got.Enabled {
		t.Fatalf("membership duration/enabled mismatch: %#v", got)
	}
	if !got.Policy.MonthlyOrderLimitEnabled || got.Policy.MonthlyOrderLimit != 5 {
		t.Fatalf("membership policy not mapped: %#v", got.Policy)
	}
	if nonMembership := MembershipPlanType(Snapshot{Kind: KindSubscribe, Name: "Plan"}); nonMembership != nil {
		t.Fatalf("non-membership snapshot mapped to membership plan: %#v", nonMembership)
	}
}

func TestFromSubscribeNormalizesTopology(t *testing.T) {
	got := FromSubscribe(&subscribe.Subscribe{
		Id:       9,
		Name:     "Plan",
		Nodes:    "1, 2,2, bad,3",
		NodeTags: " edge, premium,edge,, ",
	})
	if len(got.Nodes) != 3 || got.Nodes[0] != 1 || got.Nodes[1] != 2 || got.Nodes[2] != 3 {
		t.Fatalf("snapshot nodes = %#v, want [1 2 3]", got.Nodes)
	}
	if len(got.NodeTags) != 2 || got.NodeTags[0] != "edge" || got.NodeTags[1] != "premium" {
		t.Fatalf("snapshot node tags = %#v, want [edge premium]", got.NodeTags)
	}
}

func TestFromSubscribeParsesDiscountAndOmitsEmptyDiscount(t *testing.T) {
	got := FromSubscribe(&subscribe.Subscribe{
		Name:     "Plan",
		Discount: `[{"quantity":3,"discount":0.8}]`,
	})
	if len(got.Discount) != 1 || got.Discount[0].Quantity != 3 || got.Discount[0].Discount != 0.8 {
		t.Fatalf("snapshot discount = %#v, want parsed discount", got.Discount)
	}

	for _, raw := range []string{"", "[]", " [] ", "null", "not-json"} {
		got = FromSubscribe(&subscribe.Subscribe{Name: "Plan", Discount: raw})
		if got.Discount != nil {
			t.Fatalf("empty/invalid discount %q should not be stored in snapshot: %#v", raw, got.Discount)
		}
		payload, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("marshal snapshot for %q: %v", raw, err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(payload, &decoded); err != nil {
			t.Fatalf("unmarshal snapshot for %q: %v", raw, err)
		}
		if _, ok := decoded["discount"]; ok {
			t.Fatalf("empty/invalid discount %q should be omitted from snapshot JSON: %s", raw, payload)
		}
	}
}

func TestApplyToSubscribeModelNormalizesSnapshotTopology(t *testing.T) {
	raw := Marshal(Snapshot{
		Kind:     KindSubscribe,
		Name:     "Snapshot Plan",
		Nodes:    []int64{10, 0, 10, -1, 11},
		NodeTags: []string{" edge ", "", "vip", "edge"},
	})
	var got subscribe.Subscribe
	if !ApplyToSubscribeModel(raw, &got) {
		t.Fatal("ApplyToSubscribeModel rejected subscribe snapshot")
	}
	if got.Nodes != "10,11" || got.NodeTags != "edge,vip" {
		t.Fatalf("topology = %q/%q, want 10,11/edge,vip", got.Nodes, got.NodeTags)
	}
}

func TestSubscribeLimitsParsesOnlyRuntimeLimitFields(t *testing.T) {
	raw := Marshal(Snapshot{
		Kind:        KindSubscribe,
		Name:        "Runtime Plan",
		Description: "large presentation field not needed by xray users",
		SpeedLimit:  320,
		DeviceLimit: 8,
		Traffic:     2048,
		Nodes:       []int64{1, 2, 3},
		NodeTags:    []string{"vip", "edge"},
	})
	speedLimit, deviceLimit, ok := SubscribeLimits(raw)
	if !ok {
		t.Fatal("SubscribeLimits rejected subscribe snapshot")
	}
	if speedLimit != 320 || deviceLimit != 8 {
		t.Fatalf("limits = (%d, %d), want (320, 8)", speedLimit, deviceLimit)
	}
}

func TestSubscribeLimitsRejectsNonSubscribeSnapshot(t *testing.T) {
	raw := Marshal(Snapshot{Kind: KindMembership, Name: "Member", SpeedLimit: 100, DeviceLimit: 2})
	speedLimit, deviceLimit, ok := SubscribeLimits(raw)
	if ok || speedLimit != 0 || deviceLimit != 0 {
		t.Fatalf("non-subscribe limits = (%d, %d, %v), want zero false", speedLimit, deviceLimit, ok)
	}
}

func TestSubscribeTopologyParsesOnlyNormalizedSelectors(t *testing.T) {
	raw := Marshal(Snapshot{
		Kind:     KindSubscribe,
		Name:     "Snapshot Plan",
		Nodes:    []int64{10, 0, 10, -1, 11},
		NodeTags: []string{" edge ", "", "vip", "edge"},
		Traffic:  4096,
	})

	nodes, tags, ok := SubscribeTopology(raw)
	if !ok {
		t.Fatal("SubscribeTopology rejected subscribe snapshot")
	}
	if want := []int64{10, 11}; !reflect.DeepEqual(nodes, want) {
		t.Fatalf("topology nodes = %#v, want %#v", nodes, want)
	}
	if want := []string{"edge", "vip"}; !reflect.DeepEqual(tags, want) {
		t.Fatalf("topology tags = %#v, want %#v", tags, want)
	}
}

func TestSubscribeTopologyRejectsNonSubscribeSnapshot(t *testing.T) {
	raw := Marshal(Snapshot{Kind: KindMembership, Name: "Member", Nodes: []int64{10}})
	nodes, tags, ok := SubscribeTopology(raw)
	if ok || nodes != nil || tags != nil {
		t.Fatalf("non-subscribe topology = (%#v, %#v, %v), want nil nil false", nodes, tags, ok)
	}
}

func TestMatchesNodesNormalizesSnapshotAndInputSelectors(t *testing.T) {
	raw := Marshal(Snapshot{
		Kind:     KindSubscribe,
		Name:     "Snapshot Plan",
		Nodes:    []int64{0, 10, 10, -1},
		NodeTags: []string{" edge ", "", "edge"},
	})
	if !MatchesNodes(raw, []int64{0, 10, 10}, nil) {
		t.Fatal("expected node id match after normalization")
	}
	if !MatchesNodes(raw, nil, []string{" edge ", "edge"}) {
		t.Fatal("expected node tag match after normalization")
	}
	if MatchesNodes(raw, []int64{0, -1}, []string{" "}) {
		t.Fatal("invalid selectors should not match")
	}
}

func TestSelectorMatchHelpersUseLinearPathForSmallInput(t *testing.T) {
	intAllocs := testing.AllocsPerRun(1000, func() {
		_ = hasAnyInt64Selector([]int64{10, 11}, []int64{1, 2, 10})
	})
	if intAllocs != 0 {
		t.Fatalf("hasAnyInt64Selector small input allocations = %f, want 0", intAllocs)
	}

	stringAllocs := testing.AllocsPerRun(1000, func() {
		_ = hasAnyStringSelector([]string{"edge", "vip"}, []string{"standard", "edge"})
	})
	if stringAllocs != 0 {
		t.Fatalf("hasAnyStringSelector small input allocations = %f, want 0", stringAllocs)
	}
}

func TestSelectorMatchHelpersAvoidMapForLargeDuplicateInput(t *testing.T) {
	leftIDs := make([]int64, smallSelectorMatchThreshold+20)
	for index := range leftIDs {
		leftIDs[index] = 7
	}
	rightIDs := make([]int64, smallSelectorMatchThreshold+21)
	for index := range rightIDs {
		rightIDs[index] = int64(index + 100)
	}
	rightIDs[len(rightIDs)-1] = 7
	intAllocs := testing.AllocsPerRun(1000, func() {
		_ = hasAnyInt64Selector(leftIDs, rightIDs)
	})
	if intAllocs > 1 {
		t.Fatalf("hasAnyInt64Selector large duplicate allocations = %f, want at most 1", intAllocs)
	}

	leftTags := make([]string, smallSelectorMatchThreshold+20)
	for index := range leftTags {
		leftTags[index] = "edge"
	}
	rightTags := make([]string, smallSelectorMatchThreshold+21)
	for index := range rightTags {
		rightTags[index] = "tag-" + string(rune('a'+index))
	}
	rightTags[len(rightTags)-1] = "edge"
	stringAllocs := testing.AllocsPerRun(1000, func() {
		_ = hasAnyStringSelector(leftTags, rightTags)
	})
	if stringAllocs > 1 {
		t.Fatalf("hasAnyStringSelector large duplicate allocations = %f, want at most 1", stringAllocs)
	}
}

func TestSelectorMatchSetUpgradesOnlyAfterActualUniqueThreshold(t *testing.T) {
	intSet := newSelectorMatchSet[int64](smallSelectorMatchThreshold + 20)
	for index := 0; index < smallSelectorMatchThreshold+20; index++ {
		intSet.Add(7)
	}
	if intSet.index != nil {
		t.Fatal("duplicate-heavy int64 selector set should not allocate map")
	}
	if len(intSet.values) != 1 || cap(intSet.values) != smallSelectorMatchThreshold+1 {
		t.Fatalf("int64 selector set len/cap = %d/%d, want 1/%d", len(intSet.values), cap(intSet.values), smallSelectorMatchThreshold+1)
	}
	if !intSet.Contains(7) || intSet.Contains(8) {
		t.Fatalf("int64 selector set contains mismatch: %#v", intSet.values)
	}

	stringSet := newSelectorMatchSet[string](smallSelectorMatchThreshold + 20)
	stringSet.Add("tag-a")
	for index := 1; index <= smallSelectorMatchThreshold; index++ {
		stringSet.Add("tag-" + string(rune('a'+index)))
	}
	if stringSet.index == nil {
		t.Fatal("string selector set should allocate map after actual unique values exceed threshold")
	}
	if len(stringSet.values) != smallSelectorMatchThreshold+1 {
		t.Fatalf("string selector set len = %d, want %d", len(stringSet.values), smallSelectorMatchThreshold+1)
	}
	if maxCap := len(stringSet.values) * 2; cap(stringSet.values) > maxCap {
		t.Fatalf("string selector set cap = %d, want at most %d", cap(stringSet.values), maxCap)
	}
	if !stringSet.Contains("tag-a") || stringSet.Contains("missing") {
		t.Fatalf("string selector set contains mismatch: %#v", stringSet.values)
	}
}

func TestSelectorMatchSetBoundsSparseUpgradeCapacity(t *testing.T) {
	stringSet := newSelectorMatchSet[string](smallSelectorMatchThreshold + 40)
	for index := 0; index <= smallSelectorMatchThreshold; index++ {
		stringSet.Add("tag-" + string(rune('a'+index)))
	}
	for index := 0; index < smallSelectorMatchThreshold+40; index++ {
		stringSet.Add("tag-a")
	}
	if stringSet.index == nil {
		t.Fatal("string selector set should allocate map after actual unique values exceed threshold")
	}
	if len(stringSet.values) != smallSelectorMatchThreshold+1 {
		t.Fatalf("string selector set len = %d, want %d", len(stringSet.values), smallSelectorMatchThreshold+1)
	}
	if maxCap := len(stringSet.values) * 2; cap(stringSet.values) > maxCap {
		t.Fatalf("string selector set cap = %d, want at most %d", cap(stringSet.values), maxCap)
	}
}

func TestSelectorMatchHelpersHandleLargeInputs(t *testing.T) {
	leftIDs := make([]int64, 0, smallSelectorMatchThreshold+1)
	rightIDs := make([]int64, 0, smallSelectorMatchThreshold+2)
	for i := int64(1); i <= smallSelectorMatchThreshold+1; i++ {
		leftIDs = append(leftIDs, i)
		rightIDs = append(rightIDs, i+100)
	}
	rightIDs = append(rightIDs, smallSelectorMatchThreshold+1)
	if !hasAnyInt64Selector(leftIDs, rightIDs) {
		t.Fatal("expected large int64 selector match")
	}
	if hasAnyInt64Selector(leftIDs, []int64{1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008}) {
		t.Fatal("unexpected large int64 selector match")
	}

	leftTags := make([]string, 0, smallSelectorMatchThreshold+1)
	rightTags := make([]string, 0, smallSelectorMatchThreshold+2)
	for i := 0; i <= smallSelectorMatchThreshold; i++ {
		leftTags = append(leftTags, "tag-left-"+string(rune('a'+i)))
		rightTags = append(rightTags, "tag-right-"+string(rune('a'+i)))
	}
	rightTags = append(rightTags, "tag-left-a")
	if !hasAnyStringSelector(leftTags, rightTags) {
		t.Fatal("expected large string selector match")
	}
	if hasAnyStringSelector(leftTags, []string{"missing-a", "missing-b", "missing-c", "missing-d", "missing-e", "missing-f", "missing-g", "missing-h", "missing-i"}) {
		t.Fatal("unexpected large string selector match")
	}
}
