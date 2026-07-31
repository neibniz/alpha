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

type GetPrivacyPolicyLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get Privacy Policy
func NewGetPrivacyPolicyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPrivacyPolicyLogic {
	return &GetPrivacyPolicyLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPrivacyPolicyLogic) GetPrivacyPolicy() (resp *types.PrivacyPolicyConfig, err error) {
	resp = &types.PrivacyPolicyConfig{}
	// get tos config from db
	configs, err := l.svcCtx.SystemModel.GetTosConfig(l.ctx)
	if err != nil {
		l.Errorw("[GetTosConfig] GetTosConfig error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "GetTosConfig error: %v", err.Error())
	}
	// reflect to response
	tool.SystemConfigSliceReflectToStruct(configs, resp)
	return
}
