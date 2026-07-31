package authMethod

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetAuthMethodConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetAuthMethodConfigLogic Get auth method config
func NewGetAuthMethodConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAuthMethodConfigLogic {
	return &GetAuthMethodConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAuthMethodConfigLogic) GetAuthMethodConfig(req *types.GetAuthMethodConfigRequest) (resp *types.AuthMethodConfig, err error) {
	if isRemovedAuthMethod(req.Method) {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "authentication method is no longer supported")
	}
	method, err := l.svcCtx.AuthModel.FindOneByMethod(l.ctx, req.Method)
	if err != nil {
		l.Errorw("find one by method failed", logger.Field("method", req.Method), logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find one by method failed: %v", err.Error())
	}

	config, err := authMethodConfigForAdmin(method)
	if err != nil {
		l.Errorw("invalid auth method config", logger.Field("method", req.Method), logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "invalid auth method config: %v", err.Error())
	}
	resp = &config
	return
}
