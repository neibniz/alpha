package user

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QueryUserSubscribeLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Query User Subscribe
func NewQueryUserSubscribeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryUserSubscribeLogic {
	return &QueryUserSubscribeLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryUserSubscribeLogic) QueryUserSubscribe() (resp *types.QueryUserSubscribeListResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	data, err := l.svcCtx.UserModel.QueryUserSubscribe(l.ctx, u.Id, 0, 1, 2, 3)
	if err != nil {
		l.Errorw("[QueryUserSubscribeLogic] Query User Subscribe Error:", logger.Field("err", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Query User Subscribe Error")
	}

	list := userSubscribeRowsFromDetailsAt(data, true, time.Now())
	resp = &types.QueryUserSubscribeListResponse{
		List:  list,
		Total: int64(len(list)),
	}
	return
}

func calculateNextResetTime(expireTime int64, resetCycle int64, now time.Time) int64 {
	resetTime := time.UnixMilli(expireTime)
	switch resetCycle {
	case 0:
		return 0
	case 1:
		return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).UnixMilli()
	case 2:
		if resetTime.Day() > now.Day() {
			return time.Date(now.Year(), now.Month(), resetTime.Day(), 0, 0, 0, 0, now.Location()).UnixMilli()
		} else {
			return time.Date(now.Year(), now.Month()+1, resetTime.Day(), 0, 0, 0, 0, now.Location()).UnixMilli()
		}
	case 3:
		targetTime := time.Date(now.Year(), resetTime.Month(), resetTime.Day(), 0, 0, 0, 0, now.Location())
		if targetTime.Before(now) {
			targetTime = time.Date(now.Year()+1, resetTime.Month(), resetTime.Day(), 0, 0, 0, 0, now.Location())
		}
		return targetTime.UnixMilli()
	default:
		return 0
	}
}
