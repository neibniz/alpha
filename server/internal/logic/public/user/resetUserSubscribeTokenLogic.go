package user

import (
	"context"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/subscribecache"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/uuidx"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type ResetUserSubscribeTokenLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewResetUserSubscribeTokenLogic Reset User Subscribe Token
func NewResetUserSubscribeTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetUserSubscribeTokenLogic {
	return &ResetUserSubscribeTokenLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResetUserSubscribeTokenLogic) ResetUserSubscribeToken(req *types.ResetUserSubscribeTokenRequest) error {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	userSub, err := l.svcCtx.UserModel.FindOneSubscribeForMutation(l.ctx, req.UserSubscribeId)
	if err != nil {
		l.Errorw("FindOneSubscribe failed:", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "FindOneSubscribe failed: %v", err.Error())
	}
	if userSub.UserId != u.Id {
		l.Errorw("UserSubscribeId does not belong to the current user")
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "UserSubscribeId does not belong to the current user")
	}

	token, err := uuidx.NewSubscribeToken()
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "generate subscription token")
	}
	newUUID := uuid.New().String()

	err = l.svcCtx.UserModel.UpdateSubscribeFields(l.ctx, userSub, user.UserSubscribeFieldsUpdate{
		Token: &token,
		UUID:  &newUUID,
	})
	if err != nil {
		l.Errorw("UpdateSubscribe failed:", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "UpdateSubscribe failed: %v", err.Error())
	}
	userSub.Token = token
	userSub.UUID = newUUID
	//clear user subscription cache
	if err = l.svcCtx.UserModel.ClearSubscribeCache(l.ctx, userSub); err != nil {
		l.Errorw("ClearSubscribeCache failed", logger.Field("error", err.Error()), logger.Field("userSubscribeId", userSub.Id))
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "ClearSubscribeCache failed: %v", err.Error())
	}
	if err = subscribecache.ClearBySnapshot(l.ctx, l.svcCtx.DB, l.svcCtx.Redis, userSub.ProductSnapshot); err != nil {
		l.Errorw("Clear snapshot server user cache failed", logger.Field("error", err.Error()), logger.Field("userSubscribeId", userSub.Id))
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "Clear snapshot server user cache failed: %v", err.Error())
	}
	// Clear subscription cache
	if err = l.svcCtx.SubscribeModel.ClearCache(l.ctx, userSub.SubscribeId); err != nil {
		l.Errorw("ClearSubscribeCache failed", logger.Field("error", err.Error()), logger.Field("subscribeId", userSub.SubscribeId))
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "ClearSubscribeCache failed: %v", err.Error())
	}

	return nil
}
