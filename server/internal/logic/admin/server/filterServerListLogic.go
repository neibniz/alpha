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

type FilterServerListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewFilterServerListLogic Filter Server List
func NewFilterServerListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilterServerListLogic {
	return &FilterServerListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FilterServerListLogic) FilterServerList(req *types.FilterServerListRequest) (resp *types.FilterServerListResponse, err error) {
	total, data, err := l.svcCtx.NodeModel.FilterServerList(l.ctx, &node.FilterParams{
		Page:   req.Page,
		Size:   req.Size,
		Search: req.Search,
	})
	if err != nil {
		l.Errorw("[FilterServerList] Query Database Error: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "[FilterServerList] Query Database Error")
	}

	var lookup serverStatusLookup
	if l.svcCtx.NodeRealtime != nil {
		lookup = l.svcCtx.NodeRealtime.Get
	}

	return &types.FilterServerListResponse{
		List:  serverRowsToListItems(data, lookup),
		Total: total,
	}, nil
}
