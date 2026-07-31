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

type UpdatePrivacyPolicyConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Update Privacy Policy Config
func NewUpdatePrivacyPolicyConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdatePrivacyPolicyConfigLogic {
	return &UpdatePrivacyPolicyConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdatePrivacyPolicyConfigLogic) UpdatePrivacyPolicyConfig(req *types.PrivacyPolicyConfig) error {
	values := tool.SystemConfigUpdateValuesFromStruct(req)
	err := l.svcCtx.SystemModel.Transaction(l.ctx, func(db *gorm.DB) error {
		if err := tool.UpdateSystemConfigValues(db, "tos", values); err != nil {
			return err
		}
		return l.svcCtx.Redis.Del(l.ctx, config.TosConfigKey).Err()
	})
	if err != nil {
		l.Errorw("[UpdateTosConfigLogic] update tos config error: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update tos config error: %v", err)
	}

	return nil
}
