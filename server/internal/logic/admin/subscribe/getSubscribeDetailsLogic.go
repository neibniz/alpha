package subscribe

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetSubscribeDetailsLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get subscribe details
func NewGetSubscribeDetailsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSubscribeDetailsLogic {
	return &GetSubscribeDetailsLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSubscribeDetailsLogic) GetSubscribeDetails(req *types.GetSubscribeDetailsRequest) (resp *types.Subscribe, err error) {
	sub, err := l.svcCtx.SubscribeModel.FindOne(l.ctx, req.Id)
	if err != nil {
		l.Logger.Error("[GetSubscribeDetailsLogic] get subscribe details failed: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get subscribe details failed: %v", err.Error())
	}
	subscribe, err := subscribeFromModel(sub, true)
	if err != nil {
		l.Logger.Error("[GetSubscribeDetailsLogic] parse discount failed: ", logger.Field("error", err.Error()), logger.Field("subscribe_id", sub.Id))
	}
	return &subscribe, nil
}
