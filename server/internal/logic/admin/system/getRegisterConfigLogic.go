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

type GetRegisterConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetRegisterConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRegisterConfigLogic {
	return &GetRegisterConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRegisterConfigLogic) GetRegisterConfig() (*types.RegisterConfig, error) {
	resp := &types.RegisterConfig{}

	// get register config from database
	configs, err := l.svcCtx.SystemModel.GetRegisterConfig(l.ctx)
	if err != nil {
		l.Errorw("[GetRegisterConfig] Database query error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get register config error: %v", err.Error())
	}

	// reflect to response
	tool.SystemConfigSliceReflectToStruct(configs, resp)
	return resp, nil
}
