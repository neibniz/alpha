package svc

import (
	"github.com/hibiken/asynq"
	"github.com/neibniz/alpha/server/internal/config"
)

const AsynqRedisDB = 5

func NewAsynqClient(c config.Config) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{Addr: c.Redis.Host, Password: c.Redis.Pass, DB: AsynqRedisDB})
}
