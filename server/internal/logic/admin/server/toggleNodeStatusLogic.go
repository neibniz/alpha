package server

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type ToggleNodeStatusLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewToggleNodeStatusLogic Toggle Node Status
func NewToggleNodeStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ToggleNodeStatusLogic {
	return &ToggleNodeStatusLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ToggleNodeStatusLogic) ToggleNodeStatus(req *types.ToggleNodeStatusRequest) error {
	data, err := l.svcCtx.NodeModel.FindOneNode(l.ctx, req.Id)
	if err != nil {
		l.Errorw("[ToggleNodeStatus] Query Database Error: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "[ToggleNodeStatus] Query Database Error")
	}
	data.Enabled = req.Enable

	err = l.svcCtx.NodeModel.UpdateNode(l.ctx, data)
	if err != nil {
		l.Errorw("[ToggleNodeStatus] Update Database Error: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "[ToggleNodeStatus] Update Database Error")
	}

	return l.svcCtx.NodeModel.ClearNodeCache(l.ctx, &node.FilterNodeParams{
		Page:     1,
		Size:     1000,
		ServerId: []int64{data.ServerId},
		Tag:      tool.StringMergeAndRemoveDuplicates(data.Tags),
		Search:   "",
	})
}
