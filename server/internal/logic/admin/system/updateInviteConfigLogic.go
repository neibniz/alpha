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

type UpdateInviteConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateInviteConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateInviteConfigLogic {
	return &UpdateInviteConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateInviteConfigLogic) UpdateInviteConfig(req *types.InviteConfig) error {
	values := tool.SystemConfigUpdateValuesFromStruct(req)
	err := l.svcCtx.SystemModel.Transaction(l.ctx, func(db *gorm.DB) error {
		if err := tool.UpdateSystemConfigValues(db, "invite", values); err != nil {
			return err
		}
		// clear cache
		return l.svcCtx.Redis.Del(l.ctx, config.InviteConfigKey, config.GlobalConfigKey).Err()
	})
	if err != nil {
		l.Errorw("[UpdateInviteConfig] update invite config error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update invite config error: %v", err)
	}
	initialize.Invite(l.svcCtx)
	return nil
}
