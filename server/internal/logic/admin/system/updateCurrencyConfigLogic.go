package system

import (
	"context"

	"github.com/neibniz/alpha/server/initialize"
	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type UpdateCurrencyConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Update Currency Config
func NewUpdateCurrencyConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCurrencyConfigLogic {
	return &UpdateCurrencyConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCurrencyConfigLogic) UpdateCurrencyConfig(req *types.CurrencyConfig) error {
	values := tool.SystemConfigUpdateValuesFromStruct(req)
	err := l.svcCtx.SystemModel.Transaction(l.ctx, func(db *gorm.DB) error {
		if err := tool.UpdateSystemConfigValues(db, "currency", values); err != nil {
			return err
		}
		// clear cache
		return l.svcCtx.Redis.Del(l.ctx, config.CurrencyConfigKey, config.GlobalConfigKey).Err()
	})
	initialize.Currency(l.svcCtx)
	if err != nil {
		l.Errorw("[UpdateCurrencyConfig] update currency config error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update invite config error: %v", err)
	}
	return nil
}
