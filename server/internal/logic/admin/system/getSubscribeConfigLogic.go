package system

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetSubscribeConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSubscribeConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSubscribeConfigLogic {
	return &GetSubscribeConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSubscribeConfigLogic) GetSubscribeConfig() (resp *types.SubscribeConfig, err error) {
	resp = &types.SubscribeConfig{}
	// get subscribe config from db
	subscribeConfigs, err := l.svcCtx.SystemModel.GetSubscribeConfig(l.ctx)
	if err != nil {
		l.Errorw("[GetSubscribeConfig] Database query error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get subscribe config failed: %v", err.Error())
	}

	// reflect to response
	tool.SystemConfigSliceReflectToStruct(subscribeConfigs, resp)
	return resp, nil
}
