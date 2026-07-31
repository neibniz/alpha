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

type UpdateSiteConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateSiteConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSiteConfigLogic {
	return &UpdateSiteConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateSiteConfigLogic) UpdateSiteConfig(req *types.SiteConfig) error {
	values := tool.SystemConfigUpdateValuesFromStruct(req)
	err := l.svcCtx.SystemModel.Transaction(l.ctx, func(db *gorm.DB) error {
		if err := tool.UpdateSystemConfigValues(db, "site", values); err != nil {
			return err
		}
		return l.svcCtx.Redis.Del(l.ctx, config.SiteConfigKey, config.GlobalConfigKey).Err()
	})
	if err != nil {
		l.Logger.Error("[UpdateSiteConfig] update site config error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update site config error: %v", err.Error())
	}
	initialize.Site(l.svcCtx)
	return nil
}
