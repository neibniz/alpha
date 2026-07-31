package subscription

import (
	"reflect"
	"strings"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/user"
)

func TestCheckSubscriptionCacheColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(checkSubscriptionCacheColumns))
	for _, column := range checkSubscriptionCacheColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"user_id",
		"subscribe_id",
		"product_snapshot",
		"token",
		"expire_time",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("check subscription cache columns must include %q", column)
		}
	}

	for _, column := range checkSubscriptionCacheColumns {
		normalized := strings.Trim(column, `"`)
		switch normalized {
		case "traffic", "download", "upload", "uuid", "note", "created_at", "updated_at":
			t.Fatalf("check subscription cache columns must not include full-row-only field %q", normalized)
		}
	}
}

func TestCheckSubscriptionBatchesAreBounded(t *testing.T) {
	if checkSubscriptionBatchSize <= 0 || checkSubscriptionBatchSize > 1000 {
		t.Fatalf("batch size = %d, want 1..1000", checkSubscriptionBatchSize)
	}
	if checkSubscriptionMaxBatches <= 0 {
		t.Fatalf("max batches = %d, want positive", checkSubscriptionMaxBatches)
	}
	if subscriptionDueExpired >= subscriptionDueTraffic {
		t.Fatal("expiry must retain priority over traffic exhaustion")
	}
}

func TestCheckSubscriptionEmailAuthColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(checkSubscriptionEmailAuthColumns))
	for _, column := range checkSubscriptionEmailAuthColumns {
		columns[column] = struct{}{}
	}

	required := []string{"user_id", "auth_identifier"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("check subscription email auth columns must include %q", column)
		}
	}

	for _, column := range checkSubscriptionEmailAuthColumns {
		normalized := strings.Trim(column, `"`)
		switch normalized {
		case "id", "auth_type", "verified", "created_at", "updated_at":
			t.Fatalf("check subscription email auth columns must not include full-row-only field %q", normalized)
		}
	}
}

func TestCheckSubscriptionUserIDsDeduplicates(t *testing.T) {
	got := checkSubscriptionUserIDs([]*user.Subscribe{
		nil,
		{UserId: 0},
		{UserId: 2},
		{UserId: 1},
		{UserId: 2},
	})
	want := []int64{2, 1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("checkSubscriptionUserIDs() = %#v, want %#v", got, want)
	}
}

func TestCheckSubscriptionUserIDsSmallInputAvoidsMapAllocation(t *testing.T) {
	subs := []*user.Subscribe{{UserId: 2}, {UserId: 1}, {UserId: 2}}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = checkSubscriptionUserIDs(subs)
	})
	if allocs > 1 {
		t.Fatalf("checkSubscriptionUserIDs small input allocations = %f, want at most 1", allocs)
	}
}

func TestCheckSubscriptionUserIDsSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	subs := make([]*user.Subscribe, smallCheckSubscriptionIndexThreshold+20)
	subs[3] = &user.Subscribe{UserId: 7}
	subs[smallCheckSubscriptionIndexThreshold+10] = &user.Subscribe{UserId: 7}

	got := checkSubscriptionUserIDs(subs)
	want := []int64{7}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("checkSubscriptionUserIDs() = %#v, want %#v", got, want)
	}
	if cap(got) != smallCheckSubscriptionIndexThreshold+1 {
		t.Fatalf("checkSubscriptionUserIDs cap = %d, want small initial cap", cap(got))
	}
}

func TestCheckSubscriptionEmailIndexUsesLinearPathForSmallInput(t *testing.T) {
	index := newCheckSubscriptionEmailIndex([]*user.AuthMethods{
		nil,
		{UserId: 1, AuthIdentifier: "first@example.com"},
		{UserId: 2, AuthIdentifier: "second@example.com"},
		{UserId: 1, AuthIdentifier: "later@example.com"},
		{UserId: 3},
	})
	if index.byID != nil {
		t.Fatal("small email index should not allocate map")
	}
	if got, ok := index.Find(1); !ok || got != "first@example.com" {
		t.Fatalf("Find(1) = %q, %v; want first@example.com", got, ok)
	}
	if got, ok := index.Find(2); !ok || got != "second@example.com" {
		t.Fatalf("Find(2) = %q, %v; want second@example.com", got, ok)
	}
	if _, ok := index.Find(9); ok {
		t.Fatal("Find(9) found missing email")
	}
}

func TestCheckSubscriptionEmailIndexUpgradesLargeInputToMap(t *testing.T) {
	methods := make([]*user.AuthMethods, 0, smallCheckSubscriptionEmailIndexThreshold+2)
	for id := int64(1); id <= smallCheckSubscriptionEmailIndexThreshold+1; id++ {
		methods = append(methods, &user.AuthMethods{
			UserId:         id,
			AuthIdentifier: "user@example.com",
		})
	}
	methods = append(methods, &user.AuthMethods{
		UserId:         1,
		AuthIdentifier: "later@example.com",
	})
	index := newCheckSubscriptionEmailIndex(methods)
	if index.byID == nil {
		t.Fatal("large email index should allocate map")
	}
	lastID := int64(smallCheckSubscriptionEmailIndexThreshold + 1)
	if got, ok := index.Find(lastID); !ok || got != "user@example.com" {
		t.Fatalf("Find(%d) = %q, %v", lastID, got, ok)
	}
	if got, ok := index.Find(1); !ok || got != "user@example.com" {
		t.Fatalf("duplicate user should keep first email, got %q, %v", got, ok)
	}
}

func TestCheckSubscriptionEmailIndexAvoidsMapForLargeDuplicateInput(t *testing.T) {
	methods := make([]*user.AuthMethods, 0, smallCheckSubscriptionEmailIndexThreshold+20)
	for i := 0; i < cap(methods); i++ {
		methods = append(methods, &user.AuthMethods{
			UserId:         1,
			AuthIdentifier: "first@example.com",
		})
	}
	index := newCheckSubscriptionEmailIndex(methods)
	if index.byID != nil {
		t.Fatal("large duplicate email index should not allocate map")
	}
	if len(index.rows) != 1 {
		t.Fatalf("email index rows = %d, want 1", len(index.rows))
	}
	if cap(index.rows) != smallCheckSubscriptionEmailIndexThreshold+1 {
		t.Fatalf("email index cap = %d, want small initial cap", cap(index.rows))
	}
	if got, ok := index.Find(1); !ok || got != "first@example.com" {
		t.Fatalf("Find(1) = %q, %v", got, ok)
	}
}

func TestCheckSubscriptionEmailIndexBoundsSparseUpgradeCapacity(t *testing.T) {
	methods := make([]*user.AuthMethods, 0, smallCheckSubscriptionEmailIndexThreshold+20)
	for id := int64(1); id <= smallCheckSubscriptionEmailIndexThreshold+1; id++ {
		methods = append(methods, &user.AuthMethods{
			UserId:         id,
			AuthIdentifier: "user@example.com",
		})
	}
	for len(methods) < smallCheckSubscriptionEmailIndexThreshold+20 {
		methods = append(methods, &user.AuthMethods{
			UserId:         1,
			AuthIdentifier: "later@example.com",
		})
	}
	index := newCheckSubscriptionEmailIndex(methods)
	if index.byID == nil {
		t.Fatal("email index should allocate map after actual unique users exceed threshold")
	}
	if maxCap := len(index.rows) * 2; cap(index.rows) > maxCap {
		t.Fatalf("email index cap = %d, want at most %d", cap(index.rows), maxCap)
	}
}

func TestCheckSubscriptionDeduperBoundsSparseUpgradeCapacity(t *testing.T) {
	deduper := newCheckSubscriptionDeduper[int64](smallCheckSubscriptionIndexThreshold + 40)
	for id := int64(1); id <= smallCheckSubscriptionIndexThreshold+1; id++ {
		deduper.Add(id)
	}
	for id := 0; id < smallCheckSubscriptionIndexThreshold+40; id++ {
		deduper.Add(1)
	}

	got := deduper.Values()
	if len(got) != smallCheckSubscriptionIndexThreshold+1 {
		t.Fatalf("deduper values len = %d, want %d", len(got), smallCheckSubscriptionIndexThreshold+1)
	}
	if maxCap := len(got) * 2; cap(got) > maxCap {
		t.Fatalf("deduper values cap = %d, want at most %d", cap(got), maxCap)
	}
}

func TestCheckSubscriptionCacheTargetsDeduplicates(t *testing.T) {
	snapshotA := `{"version":1,"kind":"subscribe","name":"a"}`
	snapshotB := `{"version":1,"kind":"subscribe","name":"b"}`
	snapshots, subscribeIDs := checkSubscriptionCacheTargets([]*user.Subscribe{
		nil,
		{SubscribeId: 0},
		{SubscribeId: 10, ProductSnapshot: snapshotA},
		{SubscribeId: 10, ProductSnapshot: snapshotA},
		{SubscribeId: 11, ProductSnapshot: ""},
		{SubscribeId: 12, ProductSnapshot: snapshotB},
		{SubscribeId: 11, ProductSnapshot: snapshotB},
	})

	if want := []string{snapshotA, snapshotB}; !reflect.DeepEqual(snapshots, want) {
		t.Fatalf("snapshots = %#v, want %#v", snapshots, want)
	}
	if want := []int64{10, 11, 12}; !reflect.DeepEqual(subscribeIDs, want) {
		t.Fatalf("subscribe ids = %#v, want %#v", subscribeIDs, want)
	}
}

func TestCheckSubscriptionCacheTargetsSmallInputAvoidsMapAllocation(t *testing.T) {
	snapshot := `{"version":1,"kind":"subscribe","name":"a"}`
	subs := []*user.Subscribe{
		{SubscribeId: 10, ProductSnapshot: snapshot},
		{SubscribeId: 10, ProductSnapshot: snapshot},
		{SubscribeId: 11},
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_, _ = checkSubscriptionCacheTargets(subs)
	})
	if allocs > 4 {
		t.Fatalf("checkSubscriptionCacheTargets small input allocations = %f, want at most 4", allocs)
	}
}

func TestCheckSubscriptionCacheTargetsSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	snapshot := `{"version":1,"kind":"subscribe","name":"a"}`
	subs := make([]*user.Subscribe, smallCheckSubscriptionIndexThreshold+20)
	subs[0] = &user.Subscribe{SubscribeId: 0, ProductSnapshot: ""}
	subs[2] = &user.Subscribe{SubscribeId: 10, ProductSnapshot: snapshot}
	subs[smallCheckSubscriptionIndexThreshold+10] = &user.Subscribe{SubscribeId: 10, ProductSnapshot: snapshot}

	snapshots, subscribeIDs := checkSubscriptionCacheTargets(subs)
	if want := []string{snapshot}; !reflect.DeepEqual(snapshots, want) {
		t.Fatalf("snapshots = %#v, want %#v", snapshots, want)
	}
	if want := []int64{10}; !reflect.DeepEqual(subscribeIDs, want) {
		t.Fatalf("subscribe ids = %#v, want %#v", subscribeIDs, want)
	}
	if cap(snapshots) != smallCheckSubscriptionIndexThreshold+1 {
		t.Fatalf("snapshot cap = %d, want small initial cap", cap(snapshots))
	}
	if cap(subscribeIDs) != smallCheckSubscriptionIndexThreshold+1 {
		t.Fatalf("subscribe id cap = %d, want small initial cap", cap(subscribeIDs))
	}
}
