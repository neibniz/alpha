package subscribe

import (
	"testing"
	"time"

	userModel "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestUserSubscribeInfoFromRuntimeDetails(t *testing.T) {
	nodes := []types.UserSubscribeNodeInfo{{Id: 9, Name: "node"}}
	profiles := []types.UserSubscribeProfile{{Key: "vless", NodeCount: 1}}
	presets := []types.UserSubscribePreset{{Id: 7, Name: "preset"}}

	got := userSubscribeInfoFromRuntimeDetails(&userModel.SubscribeDetails{
		Id: 1,
	}, nodes, profiles, presets)

	if got.Id != 1 {
		t.Fatalf("id = %d, want 1", got.Id)
	}
	if len(got.Nodes) != 1 || len(got.Profiles) != 1 || len(got.Presets) != 1 {
		t.Fatalf("nested fields = %#v", got)
	}
}

func TestUserSubscribeInfoFromRuntimeDetailsKeepsZeroTimesAndEmptyLists(t *testing.T) {
	got := userSubscribeInfoFromRuntimeDetails(&userModel.SubscribeDetails{}, nil, nil, nil)
	if got.Nodes == nil || len(got.Nodes) != 0 {
		t.Fatalf("empty nodes should be non-nil empty slice: %#v", got.Nodes)
	}
	if got.Profiles == nil || len(got.Profiles) != 0 {
		t.Fatalf("empty profiles should be non-nil empty slice: %#v", got.Profiles)
	}
	if got.Presets == nil || len(got.Presets) != 0 {
		t.Fatalf("empty presets should be non-nil empty slice: %#v", got.Presets)
	}
}

func TestNodeInfoRowsFiltersEmptyOnlyWhenNeeded(t *testing.T) {
	first := types.UserSubscribeNodeInfo{Id: 1, Name: "a"}
	second := types.UserSubscribeNodeInfo{Id: 2, Name: "b"}
	withoutEmpty := []types.UserSubscribeNodeInfo{first, second}
	reused := nodeInfoRows(withoutEmpty)
	if len(reused) != 2 || &reused[0] != &withoutEmpty[0] {
		t.Fatalf("node rows without empty entries should reuse the original slice: %#v", reused)
	}

	withEmpty := nodeInfoRows([]types.UserSubscribeNodeInfo{{}, first, {}, second})
	if len(withEmpty) != 2 || withEmpty[0].Id != first.Id || withEmpty[1].Id != second.Id {
		t.Fatalf("node rows should filter empty entries: %#v", withEmpty)
	}

	empty := nodeInfoRows(nil)
	if empty == nil || len(empty) != 0 {
		t.Fatalf("empty node rows should be a non-nil empty slice: %#v", empty)
	}
}

func TestNodeInfoRowsSkipEmptyWithoutInputSizedCapacity(t *testing.T) {
	first := types.UserSubscribeNodeInfo{Id: 1, Name: "a"}
	second := types.UserSubscribeNodeInfo{Id: 2, Name: "b"}
	rows := make([]types.UserSubscribeNodeInfo, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = first
	rows[len(rows)-1] = second

	got := nodeInfoRows(rows)
	if len(got) != 2 {
		t.Fatalf("node info rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("empty-heavy node info rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != first.Id || got[1].Id != second.Id {
		t.Fatalf("node info row order changed: %#v", got)
	}
}

func TestNodeInfoRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]types.UserSubscribeNodeInfo, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = types.UserSubscribeNodeInfo{Id: int64(index + 1), Name: "node"}
	}

	got := nodeInfoRows(rows)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("node info rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("node info rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("node info row order changed: %#v", got)
	}
}

func TestSubscriptionExpiredAtUsesSingleNow(t *testing.T) {
	now := time.Unix(200, 0)
	if !subscriptionExpiredAt(time.Unix(199, 0), now) {
		t.Fatalf("subscription should be expired")
	}
	if subscriptionExpiredAt(time.Unix(200, 0), now) {
		t.Fatalf("subscription expiring at now should stay active")
	}
	if subscriptionExpiredAt(time.Unix(0, 0), now) {
		t.Fatalf("zero unix expiration should stay active")
	}
}

func TestSelectActiveSubscriptionsReturnsCompactIDsAndDetails(t *testing.T) {
	now := time.Unix(200, 0)
	rows := []*userModel.SubscribeDetails{
		{Id: 1, ExpireTime: time.Unix(199, 0)},
		nil,
		{Id: 2, ExpireTime: time.Unix(200, 0)},
		{Id: 3, ExpireTime: time.Unix(0, 0)},
	}
	selection := selectActiveSubscriptions(rows, now)

	if selection.Len() != 2 {
		t.Fatalf("active selection len = %d, want 2: %#v", selection.Len(), selection)
	}
	if selection.ids[0] != 2 || selection.ids[1] != 3 {
		t.Fatalf("active ids = %#v, want [2 3]", selection.ids)
	}
	if selection.details[0] != rows[2] || selection.details[1] != rows[3] {
		t.Fatalf("active details = %#v, want original row pointers", selection.details)
	}
	if cap(selection.ids) != 2 || cap(selection.details) != 2 {
		t.Fatalf("selection caps = ids:%d details:%d, want 2", cap(selection.ids), cap(selection.details))
	}
}

func TestSelectActiveSubscriptionsAvoidsAllocationsWhenNoneActive(t *testing.T) {
	selection := selectActiveSubscriptions([]*userModel.SubscribeDetails{
		{Id: 1, ExpireTime: time.Unix(199, 0)},
		nil,
	}, time.Unix(200, 0))
	if selection.Len() != 0 {
		t.Fatalf("active selection len = %d, want 0: %#v", selection.Len(), selection)
	}
	if selection.ids != nil || selection.details != nil {
		t.Fatalf("inactive selection should not allocate slices: %#v", selection)
	}
}

func TestActiveSubscriptionSelectionNextDetailIsAdvancesCursor(t *testing.T) {
	first := &userModel.SubscribeDetails{Id: 1}
	second := &userModel.SubscribeDetails{Id: 2}
	selection := activeSubscriptionSelection{details: []*userModel.SubscribeDetails{first, second}}
	cursor := 0
	if selection.NextDetailIs(&cursor, &userModel.SubscribeDetails{Id: 99}) {
		t.Fatal("unselected detail should not be active")
	}
	if cursor != 0 {
		t.Fatalf("cursor advanced after inactive detail: %d", cursor)
	}
	if !selection.NextDetailIs(&cursor, first) {
		t.Fatal("first detail should be active")
	}
	if cursor != 1 {
		t.Fatalf("cursor after first active index = %d, want 1", cursor)
	}
	if !selection.NextDetailIs(&cursor, second) {
		t.Fatal("second detail should be active")
	}
	if cursor != 2 {
		t.Fatalf("cursor after second active index = %d, want 2", cursor)
	}
}

func TestSubscribeNodeListRuntimeCachesAreLazyForSmallActiveSet(t *testing.T) {
	nodeCache, bindingCache := subscribeNodeListRuntimeCaches(0)
	if nodeCache != nil || bindingCache != nil {
		t.Fatalf("zero active subscriptions should not allocate caches: %#v %#v", nodeCache, bindingCache)
	}
	nodeCache, bindingCache = subscribeNodeListRuntimeCaches(1)
	if nodeCache != nil || bindingCache != nil {
		t.Fatalf("single active subscription should not allocate shared caches: %#v %#v", nodeCache, bindingCache)
	}
}

func TestSubscribeNodeListRuntimeCachesPreallocateForSharedUse(t *testing.T) {
	nodeCache, bindingCache := subscribeNodeListRuntimeCaches(3)
	if nodeCache == nil || bindingCache == nil {
		t.Fatalf("multiple active subscriptions should allocate shared caches: %#v %#v", nodeCache, bindingCache)
	}
	if len(nodeCache) != 0 || bindingCache.Len() != 0 {
		t.Fatalf("new caches should start empty: %#v %#v", nodeCache, bindingCache)
	}
}

func TestSubscribeNodeListRuntimeCacheCapIsBounded(t *testing.T) {
	if got := subscribeNodeListRuntimeCacheCap(0); got != 0 {
		t.Fatalf("zero active cache cap = %d, want 0", got)
	}
	if got := subscribeNodeListRuntimeCacheCap(1); got != 0 {
		t.Fatalf("single active cache cap = %d, want 0", got)
	}
	if got := subscribeNodeListRuntimeCacheCap(3); got != 3 {
		t.Fatalf("small active cache cap = %d, want 3", got)
	}
	if got := subscribeNodeListRuntimeCacheCap(viewutil.SmallRowsCapacityThreshold + 20); got != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("large active cache cap = %d, want %d", got, viewutil.SmallRowsCapacityThreshold+1)
	}
}
