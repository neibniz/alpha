package system

import (
	"context"

	"github.com/neibniz/alpha/server/initialize"
	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
)

type UpdateRegisterConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateRegisterConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRegisterConfigLogic {
	return &UpdateRegisterConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateRegisterConfigLogic) UpdateRegisterConfig(req *types.RegisterConfig) error {
	values := tool.SystemConfigUpdateValuesFromStruct(req)
	err := l.svcCtx.SystemModel.Transaction(l.ctx, func(db *gorm.DB) error {
		if err := tool.UpdateSystemConfigValues(db, "register", values); err != nil {
			return err
		}
		return l.svcCtx.Redis.Del(l.ctx, config.RegisterConfigKey, config.GlobalConfigKey).Err()
	})
	if err != nil {
		l.Errorw("[UpdateRegisterConfig] update register config error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update register config error: %v", err.Error())
	}
	// init system config
	initialize.Register(l.svcCtx)
	return nil
}
