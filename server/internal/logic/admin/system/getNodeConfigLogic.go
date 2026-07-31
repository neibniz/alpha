package system

import (
	"context"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetNodeConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetNodeConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetNodeConfigLogic {
	return &GetNodeConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetNodeConfigLogic) GetNodeConfig() (*types.NodeConfig, error) {
	// get server config from db
	configs, err := l.svcCtx.SystemModel.GetNodeConfig(l.ctx)
	if err != nil {
		l.Errorw("[GetNodeConfigLogic] GetNodeConfig get server config error: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "GetNodeConfig get server config error: %v", err.Error())
	}
	var dbConfig config.NodeDBConfig
	tool.SystemConfigSliceReflectToStruct(configs, &dbConfig)
	c := &types.NodeConfig{
		NodePullInterval:       dbConfig.NodePullInterval,
		NodePushInterval:       dbConfig.NodePushInterval,
		IPStrategy:             dbConfig.IPStrategy,
		TrafficReportThreshold: dbConfig.TrafficReportThreshold,
	}

	return c, nil
}
