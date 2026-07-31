package initialize

import (
	"context"

	"github.com/neibniz/alpha/server/pkg/logger"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/tool"
)

func Subscribe(svc *svc.ServiceContext) {
	logger.Debug("Subscribe config initialization")
	configs, err := svc.SystemModel.GetSubscribeConfig(context.Background())
	if err != nil {
		logger.Error("[Init Subscribe Config] Get Subscribe Config Error: ", logger.Field("error", err.Error()))
		return
	}

	var subscribeConfig config.SubscribeConfig
	tool.SystemConfigSliceReflectToStruct(configs, &subscribeConfig)
	svc.SetSubscribeConfig(subscribeConfig)
}
