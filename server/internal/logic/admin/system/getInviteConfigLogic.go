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

type GetInviteConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetInviteConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetInviteConfigLogic {
	return &GetInviteConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetInviteConfigLogic) GetInviteConfig() (*types.InviteConfig, error) {
	resp := &types.InviteConfig{}
	// get invite config from db
	configs, err := l.svcCtx.SystemModel.GetInviteConfig(l.ctx)
	if err != nil {
		l.Errorw("[GetInviteConfigLogic] get invite config error: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get invite config error: %v", err.Error())
	}
	// reflect to response
	tool.SystemConfigSliceReflectToStruct(configs, resp)

	return resp, nil
}
