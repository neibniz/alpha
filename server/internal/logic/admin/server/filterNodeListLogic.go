package server

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type FilterNodeListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewFilterNodeListLogic Filter Node List
func NewFilterNodeListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilterNodeListLogic {
	return &FilterNodeListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FilterNodeListLogic) FilterNodeList(req *types.FilterNodeListRequest) (resp *types.FilterNodeListResponse, err error) {
	total, data, err := l.svcCtx.NodeModel.FilterNodeList(l.ctx, &node.FilterNodeParams{
		Page:   req.Page,
		Size:   req.Size,
		Search: req.Search,
	})

	if err != nil {
		l.Errorw("[FilterNodeList] Query Database Error: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "[FilterNodeList] Query Database Error")
	}

	return &types.FilterNodeListResponse{
		List:  nodeRowsToListItems(data),
		Total: total,
	}, nil
}
