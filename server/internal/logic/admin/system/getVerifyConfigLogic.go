package system

import (
	"context"

	"github.com/neibniz/alpha/server/initialize"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetVerifyConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetVerifyConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetVerifyConfigLogic {
	return &GetVerifyConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetVerifyConfigLogic) GetVerifyConfig() (*types.VerifyConfig, error) {
	resp := &types.VerifyConfig{}
	// get verify config from db
	verifyConfigs, err := l.svcCtx.SystemModel.GetVerifyConfig(l.ctx)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get verify config failed: %v", err.Error())
	}
	// reflect to response
	tool.SystemConfigSliceReflectToStruct(verifyConfigs, resp)
	// update verify config to system
	initialize.Verify(l.svcCtx)
	return resp, nil
}
