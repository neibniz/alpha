package subscribe

import (
	"context"
	"reflect"
	"testing"
	"time"

	subscribeModel "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSubscribeSortIDsDeduplicatesAndSkipsInvalidIDs(t *testing.T) {
	got := subscribeSortIDs([]types.SortItem{
		{Id: 0, Sort: 1},
		{Id: 3, Sort: 1},
		{Id: -1, Sort: 2},
		{Id: 4, Sort: 0},
		{Id: 2, Sort: 3},
		{Id: 3, Sort: 4},
	})
	want := []int64{3, 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("subscribeSortIDs() = %#v, want %#v", got, want)
	}
}

func TestBuildSubscribeSortUpdatesUsesRequestedSortValues(t *testing.T) {
	got := buildSubscribeSortUpdates([]types.SortItem{
		{Id: 0, Sort: 1},
		{Id: 99, Sort: 7},
		{Id: 2, Sort: 2},
		{Id: 2, Sort: 5},
		{Id: 4, Sort: 0},
		{Id: 3, Sort: 4},
	})
	want := []types.SortItem{
		{Id: 99, Sort: 7},
		{Id: 2, Sort: 5},
		{Id: 3, Sort: 4},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSubscribeSortUpdates() = %#v, want %#v", got, want)
	}
}

func TestSubscribeSortAccumulatorUsesLinearDedupForSmallInput(t *testing.T) {
	var acc subscribeSortAccumulator
	for _, item := range []types.SortItem{
		{Id: 3, Sort: 1},
		{Id: 2, Sort: 2},
		{Id: 3, Sort: 4},
		{Id: 0, Sort: 9},
	} {
		acc.Add(item)
	}

	if acc.index != nil {
		t.Fatal("small subscribe sort accumulator should not allocate map")
	}
	if want := []int64{3, 2}; !reflect.DeepEqual(acc.IDs(), want) {
		t.Fatalf("ids = %#v, want %#v", acc.IDs(), want)
	}
	if want := []types.SortItem{{Id: 3, Sort: 4}, {Id: 2, Sort: 2}}; !reflect.DeepEqual(acc.Updates(), want) {
		t.Fatalf("updates = %#v, want %#v", acc.Updates(), want)
	}
}

func TestSubscribeSortAccumulatorUpgradesLargeInputToMap(t *testing.T) {
	var acc subscribeSortAccumulator
	for i := int64(1); i <= smallSubscribeSortSetThreshold+1; i++ {
		acc.Add(types.SortItem{Id: i, Sort: i})
	}
	acc.Add(types.SortItem{Id: 1, Sort: 99})

	if acc.index == nil {
		t.Fatal("large subscribe sort accumulator should upgrade to map")
	}
	if len(acc.ids) != smallSubscribeSortSetThreshold+1 {
		t.Fatalf("ids = %#v, want %d unique ids", acc.ids, smallSubscribeSortSetThreshold+1)
	}
	if acc.Updates()[0].Sort != 99 {
		t.Fatalf("duplicate id should update final sort value: %#v", acc.Updates()[0])
	}
}

func TestSubscribeSortAccumulatorBoundsSparseCapacity(t *testing.T) {
	acc := newSubscribeSortAccumulator(smallSubscribeSortSetThreshold + 20)
	if cap(acc.ids) != smallSubscribeSortSetThreshold+1 {
		t.Fatalf("subscribe sort accumulator ids cap = %d, want %d", cap(acc.ids), smallSubscribeSortSetThreshold+1)
	}
	if cap(acc.sorts) != smallSubscribeSortSetThreshold+1 {
		t.Fatalf("subscribe sort accumulator sorts cap = %d, want %d", cap(acc.sorts), smallSubscribeSortSetThreshold+1)
	}
	acc.Add(types.SortItem{Id: 7, Sort: 1})
	if acc.index != nil {
		t.Fatal("large hinted subscribe sort accumulator should not allocate map for one actual item")
	}
	acc.Add(types.SortItem{Id: 7, Sort: 9})
	if len(acc.ids) != 1 || acc.Updates()[0].Sort != 9 || acc.index != nil {
		t.Fatalf("subscribe sort accumulator duplicate handling mismatch: %#v/%#v", acc.ids, acc.sorts)
	}
	for id := int64(8); id <= int64(smallSubscribeSortSetThreshold+7); id++ {
		acc.Add(types.SortItem{Id: id, Sort: id})
	}
	if acc.index == nil {
		t.Fatal("subscribe sort accumulator should allocate map after actual unique ids exceed threshold")
	}
	if maxCap := len(acc.ids) * 2; cap(acc.ids) > maxCap || cap(acc.sorts) > maxCap {
		t.Fatalf("subscribe sort accumulator caps = %d/%d, want <= %d", cap(acc.ids), cap(acc.sorts), maxCap)
	}
}

func TestSubscribeSortRowIndexUsesLinearPathForSmallInput(t *testing.T) {
	rows := []subscribeSortRow{
		{Id: 1, Sort: 10},
		{Id: 2, Sort: 20},
		{Id: 1, Sort: 30},
	}
	index := newSubscribeSortRowIndex(rows)
	if index.index != nil {
		t.Fatal("small subscribe sort row index should not allocate map")
	}
	if got, ok := index.Sort(1); !ok || got != 30 {
		t.Fatalf("Sort(1) = %d, %v; want last duplicate value 30", got, ok)
	}
	if _, ok := index.Sort(9); ok {
		t.Fatal("Sort(9) found missing row")
	}
}

func TestSubscribeSortRowIndexUpgradesLargeInputToMap(t *testing.T) {
	rows := make([]subscribeSortRow, 0, smallSubscribeSortSetThreshold+1)
	for id := int64(1); id <= smallSubscribeSortSetThreshold+1; id++ {
		rows = append(rows, subscribeSortRow{Id: id, Sort: id * 10})
	}
	index := newSubscribeSortRowIndex(rows)
	if index.index == nil {
		t.Fatal("large subscribe sort row index should allocate map")
	}
	lastID := int64(smallSubscribeSortSetThreshold + 1)
	if got, ok := index.Sort(lastID); !ok || got != lastID*10 {
		t.Fatalf("Sort(%d) = %d, %v", lastID, got, ok)
	}
}

func TestSubscribeSortHelpersKeepEmptySliceShape(t *testing.T) {
	if got := subscribeSortIDs(nil); got == nil || len(got) != 0 {
		t.Fatalf("empty ids = %#v, want non-nil empty slice", got)
	}
	if got := buildSubscribeSortUpdates(nil); got == nil || len(got) != 0 {
		t.Fatalf("empty updates = %#v, want non-nil empty slice", got)
	}
}

func TestFilterExistingSubscribeSortUpdatesSkipsMissingAndNoops(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&subscribeModel.Subscribe{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*subscribeModel.Subscribe{
		{Id: 1, Name: "basic", Sort: 1},
		{Id: 2, Name: "pro", Sort: 2},
		{Id: 3, Name: "team", Sort: 3},
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := filterExistingSubscribeSortUpdates(ctx, db, []types.SortItem{
		{Id: 1, Sort: 1},
		{Id: 2, Sort: 5},
		{Id: 99, Sort: 9},
		{Id: 3, Sort: 4},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := []types.SortItem{
		{Id: 2, Sort: 5},
		{Id: 3, Sort: 4},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterExistingSubscribeSortUpdates() = %#v, want %#v", got, want)
	}
}

func TestFilterExistingSubscribeSortUpdatesSkipNoopsWithoutInputSizedCapacity(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&subscribeModel.Subscribe{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*subscribeModel.Subscribe{
		{Id: 1, Name: "basic", Sort: 1},
		{Id: 2, Name: "pro", Sort: 2},
	}).Error; err != nil {
		t.Fatal(err)
	}
	updates := make([]types.SortItem, viewutil.SmallRowsCapacityThreshold+20)
	updates[0] = types.SortItem{Id: 1, Sort: 1}
	updates[len(updates)-1] = types.SortItem{Id: 2, Sort: 5}

	got, err := filterExistingSubscribeSortUpdates(ctx, db, updates)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Id != 2 || got[0].Sort != 5 {
		t.Fatalf("filtered updates = %#v, want changed id 2", got)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy filtered updates cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestFilterExistingSubscribeSortUpdatesUseCapacityHintAfterActualThreshold(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&subscribeModel.Subscribe{}); err != nil {
		t.Fatal(err)
	}
	rows := make([]*subscribeModel.Subscribe, 0, viewutil.SmallRowsCapacityThreshold+1)
	updates := make([]types.SortItem, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		id := int64(index + 1)
		rows = append(rows, &subscribeModel.Subscribe{Id: id, Name: "subscribe", Sort: id})
		updates[index] = types.SortItem{Id: id, Sort: id + 100}
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	got, err := filterExistingSubscribeSortUpdates(ctx, db, updates)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("filtered updates len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("filtered updates cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("filtered update order changed: %#v", got)
	}
}

func TestApplySubscribeSortUpdatesOnlyChangesSortAndUpdatedAt(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&subscribeModel.Subscribe{}); err != nil {
		t.Fatal(err)
	}

	createdAt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	originalUpdatedAt := time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)
	rows := []*subscribeModel.Subscribe{
		{
			Id:          1,
			Name:        "basic",
			Description: "keep basic description",
			UnitPrice:   100,
			Discount:    `[{"quantity":1,"discount":90}]`,
			Sort:        1,
			CreatedAt:   createdAt,
			UpdatedAt:   originalUpdatedAt,
		},
		{
			Id:          2,
			Name:        "pro",
			Description: "keep pro description",
			UnitPrice:   200,
			Discount:    `[{"quantity":3,"discount":80}]`,
			Sort:        2,
			CreatedAt:   createdAt,
			UpdatedAt:   originalUpdatedAt,
		},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	updatedAt := time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC)
	if err := applySubscribeSortUpdates(ctx, db, []types.SortItem{
		{Id: 1, Sort: 2},
		{Id: 2, Sort: 1},
		{Id: 99, Sort: 9},
	}, updatedAt); err != nil {
		t.Fatal(err)
	}

	var got []subscribeModel.Subscribe
	if err := db.Order("id ASC").Find(&got).Error; err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("rows len = %d, want 2", len(got))
	}
	if got[0].Sort != 2 || got[1].Sort != 1 {
		t.Fatalf("sorts = %#v, want [2 1]", []int64{got[0].Sort, got[1].Sort})
	}
	if got[0].Name != "basic" || got[0].Description != "keep basic description" || got[0].UnitPrice != 100 || got[0].Discount == "" {
		t.Fatalf("first row non-sort fields changed: %#v", got[0])
	}
	if got[1].Name != "pro" || got[1].Description != "keep pro description" || got[1].UnitPrice != 200 || got[1].Discount == "" {
		t.Fatalf("second row non-sort fields changed: %#v", got[1])
	}
	if !got[0].UpdatedAt.Equal(updatedAt) || !got[1].UpdatedAt.Equal(updatedAt) {
		t.Fatalf("updated_at = %#v/%#v, want %#v", got[0].UpdatedAt, got[1].UpdatedAt, updatedAt)
	}
}

func TestSubscribeSortCaseExpressionCapacity(t *testing.T) {
	if got, want := subscribeSortCaseExpressionCapacity(0), len("CASE id ELSE sort END"); got != want {
		t.Fatalf("subscribeSortCaseExpressionCapacity(0) = %d, want %d", got, want)
	}
	if got, want := subscribeSortCaseExpressionCapacity(2), len("CASE id WHEN ? THEN ? WHEN ? THEN ? ELSE sort END"); got != want {
		t.Fatalf("subscribeSortCaseExpressionCapacity(2) = %d, want %d", got, want)
	}
}
