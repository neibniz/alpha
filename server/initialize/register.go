package initialize

import (
	"context"

	"github.com/neibniz/alpha/server/pkg/logger"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/tool"
)

func Register(ctx *svc.ServiceContext) {
	logger.Debug("Register config initialization")
	configs, err := ctx.SystemModel.GetRegisterConfig(context.Background())
	if err != nil {
		logger.Errorf("[Init Register Config] Get Register Config Error: %s", err.Error())
		return
	}
	var registerConfig config.RegisterConfig
	tool.SystemConfigSliceReflectToStruct(configs, &registerConfig)
	ctx.SetRegisterConfig(registerConfig)
}
