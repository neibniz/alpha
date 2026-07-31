package user

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/subscribecache"
	"github.com/neibniz/alpha/server/internal/subscriptionstate"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/uuidx"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type CreateUserSubscribeLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Create user subcribe
func NewCreateUserSubscribeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserSubscribeLogic {
	return &CreateUserSubscribeLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateUserSubscribeLogic) CreateUserSubscribe(req *types.CreateUserSubscribeRequest) error {
	// validate user
	userInfo, err := l.svcCtx.UserModel.FindOneForContext(l.ctx, req.UserId)
	if err != nil {
		l.Errorw("FindOneForContext error", logger.Field("error", err.Error()), logger.Field("userId", req.UserId))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "FindOneForContext error: %v", err.Error())
	}
	subs, err := l.svcCtx.UserModel.QueryUserSubscribeRuntime(l.ctx, req.UserId)
	if err != nil {
		l.Errorw("QueryUserSubscribe error", logger.Field("error", err.Error()), logger.Field("userId", req.UserId))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "QueryUserSubscribe error: %v", err.Error())
	}
	if len(subs) >= 1 && l.svcCtx.ConfigSnapshot().Subscribe.SingleModel {
		return errors.Wrapf(xerr.NewErrCode(xerr.SingleSubscribeModeExceedsLimit), "Single subscribe mode exceeds limit")
	}
	sub, err := l.svcCtx.SubscribeModel.FindOneForOrderSnapshot(l.ctx, req.SubscribeId)
	if err != nil {
		l.Errorw("FindOne error", logger.Field("error", err.Error()), logger.Field("subscribeId", req.SubscribeId))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "FindOne error: %v", err.Error())
	}
	if req.Traffic == 0 {
		req.Traffic = sub.Traffic
	}
	productSnapshot := ordersnapshot.FromSubscribe(sub)
	subscribeToken, err := uuidx.NewSubscribeToken()
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "generate subscription token")
	}

	startAt := time.Now()
	expireAt := time.UnixMilli(req.ExpiredAt)
	resetCycle := subscriptionstate.NormalizeResetCycle(productSnapshot.ResetCycle)
	userSub := user.Subscribe{
		UserId:          req.UserId,
		SubscribeId:     req.SubscribeId,
		ProductName:     productSnapshot.Name,
		ProductSnapshot: ordersnapshot.Marshal(productSnapshot),
		StartTime:       startAt,
		ExpireTime:      expireAt,
		Traffic:         req.Traffic,
		Download:        0,
		Upload:          0,
		ResetCycle:      resetCycle,
		NextTrafficResetAt: subscriptionstate.NextTrafficResetAt(
			resetCycle,
			startAt,
			startAt,
			expireAt,
			subscriptionstate.TrafficResetLocation(),
		),
		Token:  subscribeToken,
		UUID:   uuid.New().String(),
		Status: 1,
	}
	if err = l.svcCtx.UserModel.InsertSubscribe(l.ctx, &userSub); err != nil {
		l.Errorw("InsertSubscribe error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "InsertSubscribe error: %v", err.Error())
	}

	err = l.svcCtx.UserModel.UpdateUserCache(l.ctx, userInfo)
	if err != nil {
		l.Errorw("UpdateUserCache error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "UpdateUserCache error: %v", err.Error())
	}

	err = l.svcCtx.SubscribeModel.ClearCache(l.ctx, userSub.SubscribeId)
	if err != nil {
		logger.Errorw("ClearSubscribe error", logger.Field("error", err.Error()))
	}
	if err = subscribecache.ClearBySubscribe(l.ctx, l.svcCtx.DB, l.svcCtx.Redis, sub); err != nil {
		logger.Errorw("Clear snapshot server user cache error", logger.Field("error", err.Error()))
	}
	return nil
}
