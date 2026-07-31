package user

import (
	"context"
	"time"

	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/trafficview"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QueryUserSubscribeTrafficSeriesLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryUserSubscribeTrafficSeriesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryUserSubscribeTrafficSeriesLogic {
	return &QueryUserSubscribeTrafficSeriesLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryUserSubscribeTrafficSeriesLogic) QueryUserSubscribeTrafficSeries(req *types.QueryUserSubscribeTrafficSeriesRequest) (*types.TrafficSeriesResponse, error) {
	currentUser, ok := l.ctx.Value(constant.CtxKeyUser).(*modelUser.User)
	if !ok {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "current user is not found in context")
	}
	subscription, err := l.svcCtx.UserModel.FindOneSubscribeForMutation(l.ctx, req.UserSubscribeId)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find user subscription failed: %v", err)
	}
	if subscription.UserId != currentUser.Id {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "subscription does not belong to the current user")
	}
	if !l.svcCtx.Analytics.Available() {
		return nil, xerr.NewErrMsg("analytics storage is unavailable")
	}
	response, err := trafficview.QuerySeries(
		l.ctx,
		l.svcCtx.Analytics,
		currentUser.Id,
		req.UserSubscribeId,
		req.Period,
		time.Now(),
	)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query subscription traffic series failed: %v", err)
	}
	return response, nil
}
