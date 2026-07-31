package initialize

import (
	"context"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
)

func Invite(ctx *svc.ServiceContext) {
	// Initialize the system configuration
	logger.Debug("Register config initialization")
	configs, err := ctx.SystemModel.GetInviteConfig(context.Background())
	if err != nil {
		logger.Error("[Init Invite Config] Get Invite Config Error: ", logger.Field("error", err.Error()))
		return
	}
	var inviteConfig config.InviteConfig
	tool.SystemConfigSliceReflectToStruct(configs, &inviteConfig)
	ctx.SetInviteConfig(inviteConfig)
}
