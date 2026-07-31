package user

import (
	"context"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/subscribecache"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type UpdateUserSubscribeNoteLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUpdateUserSubscribeNoteLogic Update User Subscribe Note
func NewUpdateUserSubscribeNoteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserSubscribeNoteLogic {
	return &UpdateUserSubscribeNoteLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserSubscribeNoteLogic) UpdateUserSubscribeNote(req *types.UpdateUserSubscribeNoteRequest) error {
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

	note := req.Note

	err = l.svcCtx.UserModel.UpdateSubscribeFields(l.ctx, userSub, user.UserSubscribeFieldsUpdate{
		Note: &note,
	})
	if err != nil {
		l.Errorw("UpdateSubscribe failed:", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "UpdateSubscribe failed: %v", err.Error())
	}
	userSub.Note = note

	// Clear user subscription cache
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
