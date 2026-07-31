package traffic

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/model/traffic"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/hibiken/asynq"
)

type StatLogic struct {
	svc *svc.ServiceContext
}

func NewStatLogic(svc *svc.ServiceContext) *StatLogic {
	return &StatLogic{
		svc: svc,
	}
}

func (l *StatLogic) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	now := time.Now()
	logConfig := l.svc.LogConfig()
	if logConfig.AutoClear {
		cutoff := now.AddDate(0, 0, int(-logConfig.ClearDays))
		err := l.svc.DB.WithContext(ctx).Model(&traffic.TrafficLog{}).Where("timestamp < ?", cutoff).Delete(&traffic.TrafficLog{}).Error
		if err != nil {
			logger.Errorf("[Traffic Stat Queue] Delete legacy traffic logs failed: %v", err.Error())
			return err
		}
	}
	logger.Infof("[Traffic Stat Queue] Retention cleanup completed, consuming: %s", time.Since(now).String())
	return nil
}

func buildDailyUserTrafficRows(date string, rows []log.UserTraffic) []traffic.DailyUserTraffic {
	if len(rows) == 0 {
		return nil
	}
	result := make([]traffic.DailyUserTraffic, 0, len(rows))
	for _, row := range rows {
		result = append(result, traffic.DailyUserTraffic{
			Date:        date,
			UserId:      row.UserId,
			SubscribeId: row.SubscribeId,
			Upload:      row.Upload,
			Download:    row.Download,
			Total:       row.Total,
		})
	}
	return result
}

func buildDailyServerTrafficRows(date string, rows []log.ServerTraffic) []traffic.DailyServerTraffic {
	if len(rows) == 0 {
		return nil
	}
	result := make([]traffic.DailyServerTraffic, 0, len(rows))
	for _, row := range rows {
		result = append(result, traffic.DailyServerTraffic{
			Date:     date,
			ServerId: row.ServerId,
			Upload:   row.Upload,
			Download: row.Download,
			Total:    row.Total,
		})
	}
	return result
}
