package application

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetSubscribeApplicationLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetSubscribeApplicationLogic Get subscribe application
func NewGetSubscribeApplicationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSubscribeApplicationLogic {
	return &GetSubscribeApplicationLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSubscribeApplicationLogic) GetSubscribeApplication(req *types.GetSubscribeApplicationRequest) (resp *types.SubscribeApplication, err error) {
	data, err := l.svcCtx.ClientModel.FindOne(l.ctx, req.Id)
	if err != nil {
		l.Errorf("Failed to find subscribe application with ID %d: %v", req.Id, err)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Failed to find subscribe application with ID %d", req.Id)
	}
	application := subscribeApplicationFromModel(data, downloadLinkFromRaw(data.DownloadLink))
	resp = &application
	return
}
