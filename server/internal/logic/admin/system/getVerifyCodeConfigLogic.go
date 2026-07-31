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

type GetVerifyCodeConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get Verify Code Config
func NewGetVerifyCodeConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetVerifyCodeConfigLogic {
	return &GetVerifyCodeConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetVerifyCodeConfigLogic) GetVerifyCodeConfig() (resp *types.VerifyCodeConfig, err error) {
	data, err := l.svcCtx.SystemModel.GetVerifyCodeConfig(l.ctx)
	if err != nil {
		l.Errorw("Get Verify Code Config Error: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Get Verify Code Config Error: %s", err.Error())
	}
	resp = &types.VerifyCodeConfig{}
	tool.SystemConfigSliceReflectToStruct(data, resp)
	return
}
