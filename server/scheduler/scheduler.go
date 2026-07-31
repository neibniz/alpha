package scheduler

import (
	"time"

	"github.com/neibniz/alpha/server/pkg/logger"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/queue/types"
	"github.com/hibiken/asynq"
)

type Service struct {
	svc    *svc.ServiceContext
	server *asynq.Scheduler
}

func NewService(svc *svc.ServiceContext) *Service {
	return &Service{
		svc:    svc,
		server: initService(svc),
	}
}

func (m *Service) Start() {
	logger.Infof("start scheduler service")
	// schedule check subscription task: every 60 seconds
	checkTask := asynq.NewTask(types.SchedulerCheckSubscription, nil)
	if _, err := m.server.Register("@every 60s", checkTask); err != nil {
		logger.Errorf("register check subscription task failed: %s", err.Error())
	}
	// Process indexed reset boundaries frequently; the worker is idempotent.
	resetTrafficTask := asynq.NewTask(types.SchedulerResetTraffic, nil)
	if _, err := m.server.Register("@every 5m", resetTrafficTask); err != nil {
		logger.Errorf("register reset traffic task failed: %s", err.Error())
	}

	// schedule traffic stat task: every day at 00:00
	trafficStatTask := asynq.NewTask(types.SchedulerTrafficStat, nil)
	if _, err := m.server.Register("0 0 * * *", trafficStatTask, asynq.MaxRetry(3)); err != nil {
		logger.Errorf("register traffic stat task failed: %s", err.Error())
	}
	trafficAnalyticsTask := asynq.NewTask(types.SchedulerTrafficAnalytics, nil)
	if _, err := m.server.Register("@every 30s", trafficAnalyticsTask, asynq.MaxRetry(3)); err != nil {
		logger.Errorf("register traffic analytics sync task failed: %s", err.Error())
	}

	// schedule update exchange rate task: every day at 01:00
	rateTask := asynq.NewTask(types.SchedulerExchangeRate, nil)
	if _, err := m.server.Register("0 1 * * *", rateTask, asynq.MaxRetry(3)); err != nil {
		logger.Errorf("register update exchange rate task failed: %s", err.Error())
	}

	if err := m.server.Run(); err != nil {
		logger.Errorf("run scheduler failed: %s", err.Error())
	}
}

func (m *Service) Stop() {
	logger.Info("stop scheduler service")
	m.server.Shutdown()
}

func initService(svcCtx *svc.ServiceContext) *asynq.Scheduler {
	location, _ := time.LoadLocation("Asia/Shanghai")
	return asynq.NewScheduler(
		asynq.RedisClientOpt{Addr: svcCtx.Config.Redis.Host, Password: svcCtx.Config.Redis.Pass, DB: svc.AsynqRedisDB},
		&asynq.SchedulerOpts{
			Location: location,
		},
	)
}
