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
	"gorm.io/gorm"
)

type DeleteNodeLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeleteNodeLogic Delete Node
func NewDeleteNodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteNodeLogic {
	return &DeleteNodeLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteNodeLogic) DeleteNode(req *types.DeleteNodeRequest) error {
	data, err := l.svcCtx.NodeModel.FindOneNode(l.ctx, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return xerr.NewErrCode(xerr.NodeNotExist)
		}
		l.Errorw("[DeleteNode] Query Node Error: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "[DeleteNode] Query Node Error")
	}

	err = l.svcCtx.NodeModel.DeleteNode(l.ctx, req.Id)
	if err != nil {
		l.Errorw("[DeleteNode] Delete Database Error: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseDeletedError), "[DeleteNode] Delete Database Error")
	}

	return l.svcCtx.NodeModel.ClearNodeCache(l.ctx, &node.FilterNodeParams{
		Page:         1,
		Size:         1000,
		ServerId:     []int64{data.ServerId},
		Tag:          tool.StringMergeAndRemoveDuplicates(data.Tags),
		Search:       "",
		InboundAlias: data.Inbound(),
	})
}
