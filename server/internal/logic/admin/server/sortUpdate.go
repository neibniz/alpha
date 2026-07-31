package server

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"gorm.io/gorm"
)

type sortRow struct {
	Id   int64
	Sort int64
}

type sortColumnUpdate struct {
	CaseSQL   string
	Args      []any
	UpdatedAt time.Time
}

const smallSortSetThreshold = 8

func (u sortColumnUpdate) updates() map[string]any {
	return map[string]any{
		"sort":       gorm.Expr(u.CaseSQL, u.Args...),
		"updated_at": u.UpdatedAt,
	}
}

func resetModelSort(ctx context.Context, db *gorm.DB, model any, requested []types.SortItem, uniqueSort bool) error {
	var rows []sortRow
	if err := db.WithContext(ctx).
		Model(model).
		Select("id", "sort").
		Find(&rows).Error; err != nil {
		return err
	}

	updates := buildSortUpdates(rows, requested)
	if len(updates) == 0 {
		return nil
	}

	now := time.Now()
	if uniqueSort {
		tempUpdates := temporarySortUpdates(rows, updates)
		if err := applySortUpdates(ctx, db, model, tempUpdates, now); err != nil {
			return err
		}
	}
	return applySortUpdates(ctx, db, model, updates, now)
}

func buildSortUpdates(rows []sortRow, requested []types.SortItem) []types.SortItem {
	current := newSortRowIndex(rows)
	updates := newSortUpdateAccumulator(len(requested))
	for _, item := range requested {
		if item.Id <= 0 || item.Sort <= 0 {
			continue
		}
		if _, ok := current.Sort(item.Id); !ok {
			continue
		}
		updates.Add(item)
	}
	return updates.ChangedRows(current)
}

type sortRowIndex struct {
	rows  []sortRow
	index map[int64]int
}

func newSortRowIndex(rows []sortRow) sortRowIndex {
	idx := sortRowIndex{rows: rows}
	if len(rows) > smallSortSetThreshold {
		idx.index = make(map[int64]int, len(rows))
		for i, row := range rows {
			idx.index[row.Id] = i
		}
	}
	return idx
}

func (i sortRowIndex) Sort(id int64) (int64, bool) {
	if i.index != nil {
		pos, ok := i.index[id]
		if !ok {
			return 0, false
		}
		return i.rows[pos].Sort, true
	}
	for pos := len(i.rows) - 1; pos >= 0; pos-- {
		if i.rows[pos].Id == id {
			return i.rows[pos].Sort, true
		}
	}
	return 0, false
}

type sortUpdateAccumulator struct {
	ids     []int64
	sorts   []int64
	index   map[int64]int
	capHint int
}

func newSortUpdateAccumulator(capHint int) sortUpdateAccumulator {
	if capHint <= 0 {
		return sortUpdateAccumulator{}
	}
	return sortUpdateAccumulator{
		ids:     make([]int64, 0, sortUpdateAccumulatorInitialCap(capHint)),
		sorts:   make([]int64, 0, sortUpdateAccumulatorInitialCap(capHint)),
		capHint: capHint,
	}
}

func (a *sortUpdateAccumulator) Add(item types.SortItem) {
	if item.Id <= 0 || item.Sort <= 0 {
		return
	}
	if a.index != nil {
		if pos, ok := a.index[item.Id]; ok {
			a.sorts[pos] = item.Sort
			return
		}
		a.index[item.Id] = len(a.ids)
		a.ids = append(a.ids, item.Id)
		a.sorts = append(a.sorts, item.Sort)
		return
	}
	for i, id := range a.ids {
		if id == item.Id {
			a.sorts[i] = item.Sort
			return
		}
	}
	a.ids = append(a.ids, item.Id)
	a.sorts = append(a.sorts, item.Sort)
	if len(a.ids) > smallSortSetThreshold {
		a.upgradeLookup()
	}
}

func (a *sortUpdateAccumulator) upgradeLookup() {
	if a.index != nil {
		return
	}
	nextCap := sortUpdateAccumulatorLookupCap(len(a.ids), a.capHint)
	if nextCap > cap(a.ids) {
		ids := make([]int64, len(a.ids), nextCap)
		copy(ids, a.ids)
		a.ids = ids

		sorts := make([]int64, len(a.sorts), nextCap)
		copy(sorts, a.sorts)
		a.sorts = sorts
	}
	a.index = make(map[int64]int, nextCap)
	for i, id := range a.ids {
		a.index[id] = i
	}
}

func sortUpdateAccumulatorInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallSortSetThreshold+1 {
		return smallSortSetThreshold + 1
	}
	return capHint
}

func sortUpdateAccumulatorLookupCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func (a *sortUpdateAccumulator) ChangedRows(current sortRowIndex) []types.SortItem {
	if len(a.ids) == 0 {
		return []types.SortItem{}
	}
	updates := viewutil.NewRowsCollector[types.SortItem](len(a.ids))
	for i, id := range a.ids {
		sortValue := a.sorts[i]
		if currentSort, ok := current.Sort(id); ok && currentSort != sortValue {
			updates.Add(types.SortItem{
				Id:   id,
				Sort: sortValue,
			})
		}
	}
	return updates.Rows()
}

func temporarySortUpdates(rows []sortRow, updates []types.SortItem) []types.SortItem {
	maxAbs := int64(0)
	for _, row := range rows {
		if abs := absInt64(row.Sort); abs > maxAbs {
			maxAbs = abs
		}
		if abs := absInt64(row.Id); abs > maxAbs {
			maxAbs = abs
		}
	}
	if maxAbs > math.MaxInt64-int64(len(updates))-1 {
		maxAbs = int64(len(updates)) + 1
	}

	temp := make([]types.SortItem, 0, len(updates))
	for i, item := range updates {
		temp = append(temp, types.SortItem{
			Id:   item.Id,
			Sort: -(maxAbs + int64(i) + 1),
		})
	}
	return temp
}

func absInt64(value int64) int64 {
	if value >= 0 {
		return value
	}
	if value == math.MinInt64 {
		return math.MaxInt64
	}
	return -value
}

func applySortUpdates(ctx context.Context, db *gorm.DB, model any, updates []types.SortItem, updatedAt time.Time) error {
	caseSQL, args, ids := sortCaseExpression(updates)
	if len(ids) == 0 {
		return nil
	}
	return db.WithContext(ctx).
		Model(model).
		Where("id IN ?", ids).
		UpdateColumns(sortColumnUpdate{
			CaseSQL:   caseSQL,
			Args:      args,
			UpdatedAt: updatedAt,
		}.updates()).Error
}

func sortCaseExpression(updates []types.SortItem) (string, []any, []int64) {
	var builder strings.Builder
	builder.Grow(sortCaseExpressionCapacity(len(updates)))
	builder.WriteString("CASE id")
	args := make([]any, 0, len(updates)*2)
	ids := make([]int64, 0, len(updates))
	for _, item := range updates {
		if item.Id <= 0 {
			continue
		}
		builder.WriteString(" WHEN ? THEN ?")
		args = append(args, item.Id, item.Sort)
		ids = append(ids, item.Id)
	}
	builder.WriteString(" ELSE sort END")
	return builder.String(), args, ids
}

func sortCaseExpressionCapacity(updateCount int) int {
	if updateCount <= 0 {
		return len("CASE id") + len(" ELSE sort END")
	}
	return len("CASE id") + updateCount*len(" WHEN ? THEN ?") + len(" ELSE sort END")
}
