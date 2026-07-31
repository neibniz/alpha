package application

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/client"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetSubscribeApplicationListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetSubscribeApplicationListLogic Get subscribe application list
func NewGetSubscribeApplicationListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSubscribeApplicationListLogic {
	return &GetSubscribeApplicationListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSubscribeApplicationListLogic) GetSubscribeApplicationList(req *types.GetSubscribeApplicationListRequest) (resp *types.GetSubscribeApplicationListResponse, err error) {
	total, data, err := l.svcCtx.ClientModel.Filter(l.ctx, req.Page, req.Size, client.SubscribeApplicationListFilter{
		Search:  req.Search,
		Enabled: req.Enabled,
	})
	if err != nil {
		l.Errorf("Failed to get subscribe application list: %v", err)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Failed to get subscribe application list")
	}
	resp = &types.GetSubscribeApplicationListResponse{
		Total: total,
		List:  subscribeApplicationListRows(data),
	}
	return
}
