package common

import (
	"context"

	"github.com/neibniz/alpha/server/internal/clientview"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetClientLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get Client
func NewGetClientLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetClientLogic {
	return &GetClientLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetClientLogic) GetClient() (resp *types.GetSubscribeClientResponse, err error) {
	data, err := l.svcCtx.ClientModel.ListEnabledSummary(l.ctx)
	if err != nil {
		l.Errorf("Failed to get subscribe application list: %v", err)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Failed to get subscribe application list")
	}
	list := clientview.SubscribeClientListFromModel(data)
	resp = &types.GetSubscribeClientResponse{
		Total: int64(len(list)),
		List:  list,
	}
	return
}
