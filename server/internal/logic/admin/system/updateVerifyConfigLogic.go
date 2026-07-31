package system

import (
	"context"

	"github.com/neibniz/alpha/server/initialize"
	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type UpdateVerifyConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateVerifyConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateVerifyConfigLogic {
	return &UpdateVerifyConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateVerifyConfigLogic) UpdateVerifyConfig(req *types.VerifyConfig) error {
	values := tool.SystemConfigUpdateValuesFromStruct(req)
	err := l.svcCtx.SystemModel.Transaction(l.ctx, func(db *gorm.DB) error {
		if err := tool.UpdateSystemConfigValues(db, "verify", values); err != nil {
			return err
		}
		// clear cache
		return l.svcCtx.Redis.Del(l.ctx, config.VerifyConfigKey, config.GlobalConfigKey).Err()
	})
	if err != nil {
		l.Errorw("[UpdateVerifyConfigLogic] update verify config error: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update verify config error: %v", err)
	}
	initialize.Verify(l.svcCtx)
	return nil
}
