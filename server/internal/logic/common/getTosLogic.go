package common

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetTosLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get Tos
func NewGetTosLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTosLogic {
	return &GetTosLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTosLogic) GetTos() (resp *types.GetTosResponse, err error) {
	resp = &types.GetTosResponse{}
	// get Tos config from db
	configs, err := l.svcCtx.SystemModel.GetTosConfig(l.ctx)
	if err != nil {
		l.Errorw("[GetTosLogic] GetTos error: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "GetTos error: %v", err.Error())
	}
	// reflect to response
	tool.SystemConfigSliceReflectToStruct(configs, resp)
	return
}
