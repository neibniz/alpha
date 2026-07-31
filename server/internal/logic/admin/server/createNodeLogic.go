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

type CreateNodeLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewCreateNodeLogic Create Node
func NewCreateNodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateNodeLogic {
	return &CreateNodeLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateNodeLogic) CreateNode(req *types.CreateNodeRequest) error {
	inboundAlias := nodeRequestInboundAlias(req.InboundAlias, req.Protocol)
	if err := validateNodeInboundAlias(l.ctx, l.svcCtx, req.ServerId, inboundAlias); err != nil {
		return err
	}
	data := node.Node{
		Name:         req.Name,
		Tags:         tool.StringSliceToString(req.Tags),
		Enabled:      req.Enabled,
		Port:         req.Port,
		Address:      req.Address,
		ServerId:     req.ServerId,
		InboundAlias: inboundAlias,
		Protocol:     inboundAlias,
	}
	err := l.svcCtx.NodeModel.InsertNode(l.ctx, &data)
	if err != nil {
		l.Errorw("[CreateNode] Insert Database Error: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "[CreateNode] Insert Database Error")
	}

	return nil
}
