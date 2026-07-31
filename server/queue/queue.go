package queue

import (
	"github.com/hibiken/asynq"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/queue/handler"
)

type Service struct {
	svc    *svc.ServiceContext
	server *asynq.Server
}

func NewService(svc *svc.ServiceContext) *Service {
	return &Service{
		svc:    svc,
		server: initService(svc),
	}
}

func (m *Service) Start() {
	logger.Infof("start consumer service")
	mux := asynq.NewServeMux()
	// register tasks
	handler.RegisterHandlers(mux, m.svc)
	if err := m.server.Run(mux); err != nil {
		logger.Error("consumer service error", logger.LogField{
			Key:   "error",
			Value: err.Error(),
		})
	}
}

func (m *Service) Stop() {
	logger.Info("stop consumer service")
	m.server.Stop()
}

func initService(svcCtx *svc.ServiceContext) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: svcCtx.Config.Redis.Host, Password: svcCtx.Config.Redis.Pass, DB: svc.AsynqRedisDB},
		asynq.Config{
			IsFailure: func(err error) bool {
				logger.Error("consumer service error", logger.Field("error", err.Error()))
				return true
			},
			Concurrency: 20,
		},
	)
}
