package subscribe

import (
	"context"
	"strings"
	"time"

	subscribeModel "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type SubscribeSortLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type subscribeSortColumnUpdate struct {
	CaseSQL   string
	Args      []any
	UpdatedAt time.Time
}

type subscribeSortRow struct {
	Id   int64
	Sort int64
}

const smallSubscribeSortSetThreshold = 8

func (u subscribeSortColumnUpdate) updates() map[string]any {
	return map[string]any{
		"sort":       gorm.Expr(u.CaseSQL, u.Args...),
		"updated_at": u.UpdatedAt,
	}
}

// NewSubscribeSortLogic Subscribe sort
func NewSubscribeSortLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubscribeSortLogic {
	return &SubscribeSortLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SubscribeSortLogic) SubscribeSort(req *types.SubscribeSortRequest) error {
	updates := buildSubscribeSortUpdates(req.Sort)
	if len(updates) == 0 {
		l.Logger.Info("[UpdateSubscribeSort] Empty subscribe sort updates")
		return nil
	}

	err := l.svcCtx.SubscribeModel.Transaction(l.ctx, func(db *gorm.DB) error {
		updates, err := filterExistingSubscribeSortUpdates(l.ctx, db, updates)
		if err != nil {
			return err
		}
		if len(updates) == 0 {
			return nil
		}
		return applySubscribeSortUpdates(l.ctx, db, updates, time.Now())
	})
	if err != nil {
		l.Logger.Error("[SubscribeSortLogic] update subscribe sort error: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update subscribe sort error: %v", err.Error())
	}
	l.Logger.Info("[UpdateSubscribeSort] Successfully updated subscribe sort")
	return nil
}

func subscribeSortIDs(items []types.SortItem) []int64 {
	acc := newSubscribeSortAccumulator(len(items))
	for _, item := range items {
		acc.Add(item)
	}
	return acc.IDs()
}

func buildSubscribeSortUpdates(items []types.SortItem) []types.SortItem {
	acc := newSubscribeSortAccumulator(len(items))
	for _, item := range items {
		acc.Add(item)
	}
	return acc.Updates()
}

type subscribeSortAccumulator struct {
	ids     []int64
	sorts   []int64
	index   map[int64]int
	capHint int
}

func newSubscribeSortAccumulator(capHint int) subscribeSortAccumulator {
	if capHint <= 0 {
		return subscribeSortAccumulator{}
	}
	return subscribeSortAccumulator{
		ids:     make([]int64, 0, subscribeSortAccumulatorInitialCap(capHint)),
		sorts:   make([]int64, 0, subscribeSortAccumulatorInitialCap(capHint)),
		capHint: capHint,
	}
}

func (a *subscribeSortAccumulator) Add(item types.SortItem) {
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
	if len(a.ids) > smallSubscribeSortSetThreshold {
		a.upgradeLookup()
	}
}

func (a *subscribeSortAccumulator) upgradeLookup() {
	if a.index != nil {
		return
	}
	nextCap := subscribeSortAccumulatorLookupCap(len(a.ids), a.capHint)
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

func subscribeSortAccumulatorInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallSubscribeSortSetThreshold+1 {
		return smallSubscribeSortSetThreshold + 1
	}
	return capHint
}

func subscribeSortAccumulatorLookupCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func (a *subscribeSortAccumulator) IDs() []int64 {
	if len(a.ids) == 0 {
		return []int64{}
	}
	return a.ids
}

func (a *subscribeSortAccumulator) Updates() []types.SortItem {
	if len(a.ids) == 0 {
		return []types.SortItem{}
	}
	updates := make([]types.SortItem, 0, len(a.ids))
	for i, id := range a.ids {
		updates = append(updates, types.SortItem{
			Id:   id,
			Sort: a.sorts[i],
		})
	}
	return updates
}

type subscribeSortRowIndex struct {
	rows  []subscribeSortRow
	index map[int64]int
}

func newSubscribeSortRowIndex(rows []subscribeSortRow) subscribeSortRowIndex {
	idx := subscribeSortRowIndex{rows: rows}
	if len(rows) > smallSubscribeSortSetThreshold {
		idx.index = make(map[int64]int, len(rows))
		for pos, row := range rows {
			idx.index[row.Id] = pos
		}
	}
	return idx
}

func (i subscribeSortRowIndex) Sort(id int64) (int64, bool) {
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

func filterExistingSubscribeSortUpdates(ctx context.Context, db *gorm.DB, updates []types.SortItem) ([]types.SortItem, error) {
	if len(updates) == 0 {
		return nil, nil
	}

	ids := viewutil.NewRowsCollector[int64](len(updates))
	for _, item := range updates {
		if item.Id > 0 {
			ids.Add(item.Id)
		}
	}
	idRows := ids.Rows()
	if len(idRows) == 0 {
		return nil, nil
	}

	rows := make([]subscribeSortRow, 0, len(idRows))
	if err := db.WithContext(ctx).
		Model(&subscribeModel.Subscribe{}).
		Select("id", "sort").
		Where("id IN ?", idRows).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	currentSorts := newSubscribeSortRowIndex(rows)

	filtered := viewutil.NewRowsCollector[types.SortItem](len(updates))
	for _, item := range updates {
		if currentSort, ok := currentSorts.Sort(item.Id); ok && currentSort != item.Sort {
			filtered.Add(item)
		}
	}
	return filtered.Rows(), nil
}

func applySubscribeSortUpdates(ctx context.Context, db *gorm.DB, updates []types.SortItem, updatedAt time.Time) error {
	caseSQL, args, ids := subscribeSortCaseExpression(updates)
	if len(ids) == 0 {
		return nil
	}
	return db.WithContext(ctx).
		Model(&subscribeModel.Subscribe{}).
		Where("id IN ?", ids).
		UpdateColumns(subscribeSortColumnUpdate{
			CaseSQL:   caseSQL,
			Args:      args,
			UpdatedAt: updatedAt,
		}.updates()).Error
}

func subscribeSortCaseExpression(updates []types.SortItem) (string, []any, []int64) {
	var builder strings.Builder
	builder.Grow(subscribeSortCaseExpressionCapacity(len(updates)))
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

func subscribeSortCaseExpressionCapacity(updateCount int) int {
	if updateCount <= 0 {
		return len("CASE id") + len(" ELSE sort END")
	}
	return len("CASE id") + updateCount*len(" WHEN ? THEN ?") + len(" ELSE sort END")
}
