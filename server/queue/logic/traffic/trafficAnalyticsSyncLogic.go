package traffic

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/analytics"
	trafficModel "github.com/neibniz/alpha/server/internal/model/traffic"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/redislock"
	"github.com/hibiken/asynq"
)

const trafficAnalyticsSyncBatchSize = 2000
const trafficAnalyticsSyncMaxBatches = 10
const trafficAnalyticsSyncLockKey = "traffic:analytics:sync:lock"
const trafficAnalyticsSyncLockTTL = 2 * time.Minute

var trafficAnalyticsOutboxColumns = []string{
	"id",
	"event_id",
	"server_id",
	"user_id",
	"user_subscribe_id",
	"download",
	"upload",
	"event_time",
}

type TrafficAnalyticsSyncLogic struct {
	svc *svc.ServiceContext
}

func NewTrafficAnalyticsSyncLogic(svcCtx *svc.ServiceContext) *TrafficAnalyticsSyncLogic {
	return &TrafficAnalyticsSyncLogic{svc: svcCtx}
}

func (l *TrafficAnalyticsSyncLogic) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	if !l.svc.Analytics.Available() {
		return analytics.ErrUnavailable
	}
	lock, acquired, err := redislock.Acquire(ctx, l.svc.Redis, trafficAnalyticsSyncLockKey, trafficAnalyticsSyncLockTTL)
	if err != nil {
		return err
	}
	if !acquired {
		return nil
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := lock.Release(releaseCtx); err != nil {
			logger.WithContext(ctx).Errorw("[TrafficAnalyticsSync] release lock failed", logger.Field("error", err.Error()))
		}
	}()
	for batch := 0; batch < trafficAnalyticsSyncMaxBatches; batch++ {
		count, err := l.syncBatch(ctx)
		if err != nil {
			return err
		}
		if count < trafficAnalyticsSyncBatchSize {
			return nil
		}
	}
	return nil
}

func (l *TrafficAnalyticsSyncLogic) syncBatch(ctx context.Context) (int, error) {
	var rows []trafficModel.EventOutbox
	if err := l.svc.DB.WithContext(ctx).
		Select(trafficAnalyticsOutboxColumns).
		Order("id ASC").
		Limit(trafficAnalyticsSyncBatchSize).
		Find(&rows).Error; err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}

	events := make([]analytics.TrafficEvent, len(rows))
	ids := make([]int64, len(rows))
	for i := range rows {
		row := &rows[i]
		events[i] = analytics.TrafficEvent{
			EventID:         row.EventId,
			EventTime:       row.EventTime,
			ServerID:        row.ServerId,
			UserID:          row.UserId,
			UserSubscribeID: row.UserSubscribeId,
			Download:        row.Download,
			Upload:          row.Upload,
		}
		ids[i] = row.Id
	}
	if err := l.svc.Analytics.InsertTrafficEvents(ctx, events); err != nil {
		logger.WithContext(ctx).Errorw("[TrafficAnalyticsSync] insert ClickHouse batch failed",
			logger.Field("error", err.Error()), logger.Field("event_count", len(events)))
		return len(rows), err
	}
	if err := l.svc.DB.WithContext(ctx).Where("id IN ?", ids).Delete(&trafficModel.EventOutbox{}).Error; err != nil {
		return len(rows), err
	}
	return len(rows), nil
}
