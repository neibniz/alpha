package system

import (
	"context"

	"github.com/neibniz/alpha/server/internal/config"

	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type UpdateVerifyCodeConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Update Verify Code Config
func NewUpdateVerifyCodeConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateVerifyCodeConfigLogic {
	return &UpdateVerifyCodeConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateVerifyCodeConfigLogic) UpdateVerifyCodeConfig(req *types.VerifyCodeConfig) error {
	values := tool.SystemConfigUpdateValuesFromStruct(req)
	err := l.svcCtx.SystemModel.Transaction(l.ctx, func(db *gorm.DB) error {
		if err := tool.UpdateSystemConfigValues(db, "verify_code", values); err != nil {
			return err
		}
		return l.svcCtx.Redis.Del(l.ctx, config.VerifyCodeConfigKey).Err()
	})
	if err != nil {
		l.Errorw("[UpdateRegisterConfig] update verify code config error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update register config error: %v", err.Error())
	}
	return nil
}
