package traffic

import (
	"context"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/subscribecache"
	"github.com/neibniz/alpha/server/internal/subscriptionstate"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/redislock"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ResetTrafficLogic struct {
	svc *svc.ServiceContext
}

const (
	resetTrafficBatchSize  = 500
	resetTrafficMaxBatches = 20
	resetTrafficLockTTL    = 5 * time.Minute
	resetTrafficLockKey    = "reset_traffic_lock"
)

var resetTrafficDueColumns = []string{
	"id",
	"user_id",
	"subscribe_id",
	"product_snapshot",
	"token",
	"start_time",
	"expire_time",
	"reset_cycle",
	"next_traffic_reset_at",
	"status",
}

func NewResetTrafficLogic(svc *svc.ServiceContext) *ResetTrafficLogic {
	return &ResetTrafficLogic{svc: svc}
}

func (l *ResetTrafficLogic) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	lock, acquired, err := redislock.Acquire(ctx, l.svc.Redis, resetTrafficLockKey, resetTrafficLockTTL)
	if err != nil {
		return err
	}
	if !acquired {
		return nil
	}
	defer func() {
		if releaseErr := lock.Release(ctx); releaseErr != nil {
			logger.Errorw("[ResetTraffic] release lock failed", logger.Field("error", releaseErr.Error()))
		}
	}()

	now := time.Now()
	total := 0
	for batch := 0; batch < resetTrafficMaxBatches; batch++ {
		rows, batchErr := l.resetNextBatch(ctx, now)
		if batchErr != nil {
			return batchErr
		}
		if len(rows) == 0 {
			break
		}
		total += len(rows)
		l.clearCachesAfterCommit(ctx, rows)
		if len(rows) < resetTrafficBatchSize {
			break
		}
	}
	logger.Infow("[ResetTraffic] processed due subscriptions", logger.Field("count", total))
	return nil
}

func (l *ResetTrafficLogic) resetNextBatch(ctx context.Context, now time.Time) ([]*user.Subscribe, error) {
	rows := make([]*user.Subscribe, 0, resetTrafficBatchSize)
	err := l.svc.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Model(&user.Subscribe{}).
			Select(resetTrafficDueColumns).
			Where("status IN ?", []int64{int64(subscriptionstate.StatusActive), int64(subscriptionstate.StatusExhausted)}).
			Where("next_traffic_reset_at IS NOT NULL AND next_traffic_reset_at <= ?", now).
			Where("expire_time = to_timestamp(0) OR expire_time > ?", now).
			Order("next_traffic_reset_at ASC, id ASC").
			Limit(resetTrafficBatchSize).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
		if err := query.Find(&rows).Error; err != nil || len(rows) == 0 {
			return err
		}

		nextByID := make(map[int64]*time.Time, len(rows))
		ids := make([]int64, 0, len(rows))
		for _, row := range rows {
			if row == nil {
				continue
			}
			ids = append(ids, row.Id)
			nextByID[row.Id] = subscriptionstate.NextTrafficResetAt(
				row.ResetCycle,
				row.StartTime,
				now,
				row.ExpireTime,
				subscriptionstate.TrafficResetLocation(),
			)
		}
		if len(ids) == 0 {
			return nil
		}

		updated := tx.Model(&user.Subscribe{}).
			Where("id IN ? AND next_traffic_reset_at IS NOT NULL AND next_traffic_reset_at <= ?", ids, now).
			Updates(map[string]any{
				"download":              int64(0),
				"upload":                int64(0),
				"traffic_reset_at":      now,
				"next_traffic_reset_at": nextTrafficResetAtExpr(ids, nextByID),
				"status":                subscriptionstate.StatusActive,
				"finished_at":           nil,
			})
		if updated.Error != nil {
			return updated.Error
		}
		logRows := buildResetTrafficLogRows(rows, now)
		if len(logRows) > 0 {
			return tx.Model(&log.SystemLog{}).CreateInBatches(&logRows, resetTrafficBatchSize).Error
		}
		return nil
	})
	return rows, err
}

func nextTrafficResetAtExpr(ids []int64, nextByID map[int64]*time.Time) clause.Expr {
	var sql strings.Builder
	sql.WriteString("CASE id")
	args := make([]any, 0, len(ids)*2)
	for _, id := range ids {
		sql.WriteString(" WHEN ? THEN ?")
		args = append(args, id)
		if next := nextByID[id]; next != nil {
			args = append(args, *next)
		} else {
			args = append(args, nil)
		}
	}
	sql.WriteString(" ELSE next_traffic_reset_at END")
	return gorm.Expr(sql.String(), args...)
}

func (l *ResetTrafficLogic) clearCachesAfterCommit(ctx context.Context, rows []*user.Subscribe) {
	productIDs := viewutil.NewRowsCollector[int64](len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		if err := l.svc.UserModel.ClearSubscribeCache(ctx, row); err != nil {
			logger.Errorw("[ResetTraffic] clear user subscription cache failed",
				logger.Field("user_subscribe_id", row.Id),
				logger.Field("error", err.Error()),
			)
		}
		if err := subscribecache.ClearBySnapshot(ctx, l.svc.DB, l.svc.Redis, row.ProductSnapshot); err != nil {
			logger.Errorw("[ResetTraffic] clear snapshot server cache failed",
				logger.Field("user_subscribe_id", row.Id),
				logger.Field("error", err.Error()),
			)
		}
		productIDs.Add(row.SubscribeId)
	}
	if ids := productIDs.Rows(); len(ids) > 0 {
		if err := l.svc.SubscribeModel.ClearCache(ctx, ids...); err != nil {
			logger.Errorw("[ResetTraffic] clear product cache failed", logger.Field("error", err.Error()))
		}
	}
}

func buildResetTrafficLogRows(rows []*user.Subscribe, now time.Time) []log.SystemLog {
	if len(rows) == 0 {
		return nil
	}
	result := viewutil.NewRowsCollector[log.SystemLog](len(rows))
	date := now.In(subscriptionstate.TrafficResetLocation()).Format(time.DateOnly)
	timestamp := now.UnixMilli()
	for _, row := range rows {
		if row == nil {
			continue
		}
		content, _ := (&log.ResetSubscribe{
			Type:      log.ResetSubscribeTypeAuto,
			UserId:    row.UserId,
			Timestamp: timestamp,
		}).Marshal()
		result.Add(log.SystemLog{
			Type:     log.TypeResetSubscribe.Uint8(),
			ObjectID: row.Id,
			Date:     date,
			Success:  true,
			EventAt:  timestamp,
			Content:  string(content),
		})
	}
	return result.Rows()
}
