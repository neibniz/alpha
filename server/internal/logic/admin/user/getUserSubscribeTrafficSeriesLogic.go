package user

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/trafficview"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetUserSubscribeTrafficSeriesLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserSubscribeTrafficSeriesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserSubscribeTrafficSeriesLogic {
	return &GetUserSubscribeTrafficSeriesLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserSubscribeTrafficSeriesLogic) GetUserSubscribeTrafficSeries(req *types.GetUserSubscribeTrafficSeriesRequest) (*types.TrafficSeriesResponse, error) {
	subscription, err := l.svcCtx.UserModel.FindOneSubscribeForMutation(l.ctx, req.UserSubscribeId)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find user subscription failed: %v", err)
	}
	if subscription.UserId != req.UserId {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "subscription does not belong to the requested user")
	}
	if !l.svcCtx.Analytics.Available() {
		return nil, xerr.NewErrMsg("analytics storage is unavailable")
	}
	response, err := trafficview.QuerySeries(
		l.ctx,
		l.svcCtx.Analytics,
		req.UserId,
		req.UserSubscribeId,
		req.Period,
		time.Now(),
	)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query subscription traffic series failed: %v", err)
	}
	return response, nil
}
