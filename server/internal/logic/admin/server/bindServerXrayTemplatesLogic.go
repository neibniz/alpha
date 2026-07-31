package server

import (
	"context"

	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type BindServerXrayTemplatesLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBindServerXrayTemplatesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BindServerXrayTemplatesLogic {
	return &BindServerXrayTemplatesLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BindServerXrayTemplatesLogic) BindServerXrayTemplates(req *types.BindServerXrayTemplatesRequest) error {
	if req.ServerId == 0 {
		return errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "server id is required"), "server id is required")
	}
	srv, err := l.svcCtx.NodeModel.FindOneServerForRuntime(l.ctx, req.ServerId)
	if err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query server error: %v", err)
	}
	bindings, enabledBindings, err := buildServerXrayTemplateBindings(l.ctx, l.svcCtx.DB, req.ServerId, req.Bindings, xrayBindingBuildOptions{})
	if err != nil {
		return err
	}
	if _, err := buildXrayConfigResponse(srv, enabledBindings); err != nil {
		return errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, err.Error()), "validate xray template bindings")
	}
	if err := l.svcCtx.NodeModel.BindXrayTemplates(l.ctx, req.ServerId, bindings); err != nil {
		l.Errorf("[BindServerXrayTemplates] bind error: %v", err.Error())
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "bind server xray templates error: %v", err)
	}
	if err := l.svcCtx.NodeModel.ClearServerCache(l.ctx, req.ServerId); err != nil {
		l.Errorf("[BindServerXrayTemplates] clear cache error: %v", err.Error())
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventServer)
	}
	return nil
}
