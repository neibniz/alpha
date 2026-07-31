package server

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type DeleteXrayTemplateLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteXrayTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteXrayTemplateLogic {
	return &DeleteXrayTemplateLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteXrayTemplateLogic) DeleteXrayTemplate(req *types.DeleteXrayTemplateRequest) error {
	if req.Id == 0 {
		return errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "template id is required"), "template id is required")
	}
	if err := l.svcCtx.NodeModel.Transaction(l.ctx, func(db *gorm.DB) error {
		if err := db.Where("template_id = ?", req.Id).Delete(&node.ServerXrayTemplate{}).Error; err != nil {
			return err
		}
		return l.svcCtx.NodeModel.DeleteXrayTemplate(l.ctx, req.Id, db)
	}); err != nil {
		l.Errorf("[DeleteXrayTemplate] delete error: %v", err.Error())
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseDeletedError), "delete xray template error: %v", err)
	}
	return nil
}
