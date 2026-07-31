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

type GetCurrencyConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get Currency Config
func NewGetCurrencyConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCurrencyConfigLogic {
	return &GetCurrencyConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCurrencyConfigLogic) GetCurrencyConfig() (resp *types.CurrencyConfig, err error) {
	configs, err := l.svcCtx.SystemModel.GetCurrencyConfig(l.ctx)
	if err != nil {
		l.Errorw("[GetCurrencyConfigLogic] GetCurrencyConfig error: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "GetCurrencyConfig error: %v", err.Error())
	}
	resp = &types.CurrencyConfig{}
	tool.SystemConfigSliceReflectToStruct(configs, resp)
	return
}
