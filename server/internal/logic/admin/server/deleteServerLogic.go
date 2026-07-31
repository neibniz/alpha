package server

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type DeleteServerLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeleteServerLogic Delete Server
func NewDeleteServerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteServerLogic {
	return &DeleteServerLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteServerLogic) DeleteServer(req *types.DeleteServerRequest) error {
	if _, err := l.svcCtx.NodeModel.FindOneServer(l.ctx, req.Id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return xerr.NewErrCode(xerr.ServerNotExist)
		}
		l.Errorw("[DeleteServer] Query Server Error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "[DeleteServer] Query Server Error")
	}
	nodeCount, _, err := l.svcCtx.NodeModel.FilterNodeList(l.ctx, &node.FilterNodeParams{
		Page:     1,
		Size:     1,
		ServerId: []int64{req.Id},
	})
	if err != nil {
		l.Errorw("[DeleteServer] Query Node References Error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "[DeleteServer] Query Node References Error")
	}
	bindings, err := l.svcCtx.NodeModel.QueryServerXrayTemplates(l.ctx, req.Id, false)
	if err != nil {
		l.Errorw("[DeleteServer] Query Template References Error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "[DeleteServer] Query Template References Error")
	}
	if nodeCount > 0 || len(bindings) > 0 {
		return xerr.NewErrCode(xerr.ServerReferenced)
	}
	err = l.svcCtx.NodeModel.DeleteServer(l.ctx, req.Id)
	if err != nil {
		l.Errorw("[DeleteServer] Delete Server Error: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseDeletedError), "[DeleteServer] Delete Server Error")
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventServer)
	}
	return l.svcCtx.NodeModel.ClearNodeCache(l.ctx, &node.FilterNodeParams{
		Page:     1,
		Size:     1000,
		ServerId: []int64{req.Id},
		Search:   "",
	})
}
