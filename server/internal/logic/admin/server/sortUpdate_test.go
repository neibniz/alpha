package server

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBuildSortUpdatesFiltersInvalidNoopsAndDuplicates(t *testing.T) {
	got := buildSortUpdates([]sortRow{
		{Id: 1, Sort: 1},
		{Id: 2, Sort: 2},
		{Id: 3, Sort: 3},
		{Id: 4, Sort: 4},
	}, []types.SortItem{
		{Id: 0, Sort: 1},
		{Id: 99, Sort: 1},
		{Id: 1, Sort: 1},
		{Id: 1, Sort: 4},
		{Id: 2, Sort: 3},
		{Id: 2, Sort: 1},
		{Id: 3, Sort: 2},
		{Id: 4, Sort: 0},
	})
	want := []types.SortItem{
		{Id: 1, Sort: 4},
		{Id: 2, Sort: 1},
		{Id: 3, Sort: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSortUpdates() = %#v, want %#v", got, want)
	}
}

func TestBuildSortUpdatesKeepsEmptySliceShape(t *testing.T) {
	got := buildSortUpdates([]sortRow{{Id: 1, Sort: 1}}, []types.SortItem{
		{Id: 1, Sort: 1},
		{Id: 2, Sort: 3},
		{Id: 0, Sort: 4},
	})
	if got == nil || len(got) != 0 {
		t.Fatalf("buildSortUpdates empty result = %#v, want non-nil empty slice", got)
	}
}

func TestSortRowIndexUsesLinearPathForSmallInput(t *testing.T) {
	idx := newSortRowIndex([]sortRow{
		{Id: 1, Sort: 1},
		{Id: 2, Sort: 2},
		{Id: 1, Sort: 3},
	})

	if idx.index != nil {
		t.Fatal("small sort row index should not allocate map")
	}
	got, ok := idx.Sort(1)
	if !ok || got != 3 {
		t.Fatalf("sort row index Sort(1) = %d/%v, want 3/true", got, ok)
	}
	if _, ok := idx.Sort(99); ok {
		t.Fatal("sort row index should miss unknown id")
	}
}

func TestSortRowIndexUpgradesLargeInputToMap(t *testing.T) {
	rows := make([]sortRow, 0, smallSortSetThreshold+1)
	for i := int64(1); i <= smallSortSetThreshold+1; i++ {
		rows = append(rows, sortRow{Id: i, Sort: i})
	}
	idx := newSortRowIndex(rows)

	if idx.index == nil {
		t.Fatal("large sort row index should allocate map")
	}
	got, ok := idx.Sort(smallSortSetThreshold + 1)
	if !ok || got != smallSortSetThreshold+1 {
		t.Fatalf("sort row index Sort(last) = %d/%v", got, ok)
	}
}

func TestSortUpdateAccumulatorUsesLinearPathForSmallInput(t *testing.T) {
	var acc sortUpdateAccumulator
	acc.Add(types.SortItem{Id: 1, Sort: 1})
	acc.Add(types.SortItem{Id: 2, Sort: 2})
	acc.Add(types.SortItem{Id: 1, Sort: 4})
	acc.Add(types.SortItem{Id: 0, Sort: 7})

	if acc.index != nil {
		t.Fatal("small sort update accumulator should not allocate map")
	}
	current := newSortRowIndex([]sortRow{{Id: 1, Sort: 1}, {Id: 2, Sort: 2}})
	got := acc.ChangedRows(current)
	want := []types.SortItem{{Id: 1, Sort: 4}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("changed rows = %#v, want %#v", got, want)
	}
}

func TestSortUpdateAccumulatorChangedRowsSkipNoopsWithoutInputSizedCapacity(t *testing.T) {
	var acc sortUpdateAccumulator
	for id := int64(1); id <= int64(viewutil.SmallRowsCapacityThreshold+20); id++ {
		acc.Add(types.SortItem{Id: id, Sort: id})
	}
	current := newSortRowIndex([]sortRow{
		{Id: 1, Sort: 0},
		{Id: int64(viewutil.SmallRowsCapacityThreshold + 20), Sort: 0},
	})

	got := acc.ChangedRows(current)
	if len(got) != 2 {
		t.Fatalf("changed rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy changed rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != int64(viewutil.SmallRowsCapacityThreshold+20) {
		t.Fatalf("changed row order changed: %#v", got)
	}
}

func TestSortUpdateAccumulatorChangedRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	var acc sortUpdateAccumulator
	rows := make([]sortRow, 0, viewutil.SmallRowsCapacityThreshold+1)
	for id := int64(1); id <= int64(viewutil.SmallRowsCapacityThreshold+1); id++ {
		acc.Add(types.SortItem{Id: id, Sort: id + 100})
		rows = append(rows, sortRow{Id: id, Sort: id})
	}

	got := acc.ChangedRows(newSortRowIndex(rows))
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("changed rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("changed rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("changed row order changed: %#v", got)
	}
}

func TestSortUpdateAccumulatorUpgradesLargeInputToMap(t *testing.T) {
	var acc sortUpdateAccumulator
	for i := int64(1); i <= smallSortSetThreshold+1; i++ {
		acc.Add(types.SortItem{Id: i, Sort: i})
	}
	acc.Add(types.SortItem{Id: 1, Sort: 99})

	if acc.index == nil {
		t.Fatal("large sort update accumulator should allocate map")
	}
	if len(acc.ids) != smallSortSetThreshold+1 {
		t.Fatalf("ids len = %d, want %d", len(acc.ids), smallSortSetThreshold+1)
	}
	if acc.sorts[0] != 99 {
		t.Fatalf("duplicate update should keep latest sort, got %#v", acc.sorts)
	}
}

func TestSortUpdateAccumulatorBoundsSparseCapacity(t *testing.T) {
	acc := newSortUpdateAccumulator(smallSortSetThreshold + 20)
	if cap(acc.ids) != smallSortSetThreshold+1 {
		t.Fatalf("sort update accumulator ids cap = %d, want %d", cap(acc.ids), smallSortSetThreshold+1)
	}
	if cap(acc.sorts) != smallSortSetThreshold+1 {
		t.Fatalf("sort update accumulator sorts cap = %d, want %d", cap(acc.sorts), smallSortSetThreshold+1)
	}
	acc.Add(types.SortItem{Id: 7, Sort: 1})
	if acc.index != nil {
		t.Fatal("large hinted sort update accumulator should not allocate map for one actual item")
	}
	acc.Add(types.SortItem{Id: 7, Sort: 9})
	if len(acc.ids) != 1 || acc.sorts[0] != 9 || acc.index != nil {
		t.Fatalf("sort update accumulator duplicate handling mismatch: %#v/%#v", acc.ids, acc.sorts)
	}
	for id := int64(8); id <= int64(smallSortSetThreshold+7); id++ {
		acc.Add(types.SortItem{Id: id, Sort: id})
	}
	if acc.index == nil {
		t.Fatal("sort update accumulator should allocate map after actual unique ids exceed threshold")
	}
	if maxCap := len(acc.ids) * 2; cap(acc.ids) > maxCap || cap(acc.sorts) > maxCap {
		t.Fatalf("sort update accumulator caps = %d/%d, want <= %d", cap(acc.ids), cap(acc.sorts), maxCap)
	}
}

func TestResetModelSortSwapsUniqueNodeSorts(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&node.Node{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*node.Node{
		{Id: 1, Name: "node-1", Protocol: "vless", InboundAlias: "vless", Sort: 1},
		{Id: 2, Name: "node-2", Protocol: "vless", InboundAlias: "vless", Sort: 2},
		{Id: 3, Name: "node-3", Protocol: "vless", InboundAlias: "vless", Sort: 3},
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return resetModelSort(ctx, tx, &node.Node{}, []types.SortItem{
			{Id: 1, Sort: 3},
			{Id: 3, Sort: 1},
		}, true)
	}); err != nil {
		t.Fatal(err)
	}

	var rows []sortRow
	if err := db.Model(&node.Node{}).Select("id", "sort").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Id < rows[j].Id
	})
	got := []sortRow(rows)
	want := []sortRow{
		{Id: 1, Sort: 3},
		{Id: 2, Sort: 2},
		{Id: 3, Sort: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("node sorts = %#v, want %#v", got, want)
	}
}

func TestSortCaseExpression(t *testing.T) {
	expr, args, ids := sortCaseExpression([]types.SortItem{
		{Id: 4, Sort: 1},
		{Id: 2, Sort: 3},
	})
	if expr != "CASE id WHEN ? THEN ? WHEN ? THEN ? ELSE sort END" {
		t.Fatalf("expr = %q", expr)
	}
	if !reflect.DeepEqual(args, []interface{}{int64(4), int64(1), int64(2), int64(3)}) {
		t.Fatalf("args = %#v", args)
	}
	if !reflect.DeepEqual(ids, []int64{4, 2}) {
		t.Fatalf("ids = %#v", ids)
	}
}

func TestSortCaseExpressionCapacity(t *testing.T) {
	if got, want := sortCaseExpressionCapacity(0), len("CASE id ELSE sort END"); got != want {
		t.Fatalf("sortCaseExpressionCapacity(0) = %d, want %d", got, want)
	}
	if got, want := sortCaseExpressionCapacity(2), len("CASE id WHEN ? THEN ? WHEN ? THEN ? ELSE sort END"); got != want {
		t.Fatalf("sortCaseExpressionCapacity(2) = %d, want %d", got, want)
	}
}
