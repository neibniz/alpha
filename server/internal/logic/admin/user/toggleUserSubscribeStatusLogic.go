package user

import (
	"context"

	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/subscribecache"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type ToggleUserSubscribeStatusLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewToggleUserSubscribeStatusLogic Stop user subscribe
func NewToggleUserSubscribeStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ToggleUserSubscribeStatusLogic {
	return &ToggleUserSubscribeStatusLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ToggleUserSubscribeStatusLogic) ToggleUserSubscribeStatus(req *types.ToggleUserSubscribeStatusRequest) error {
	userSub, err := l.svcCtx.UserModel.FindOneSubscribeForMutation(l.ctx, req.UserSubscribeId)
	if err != nil {
		l.Errorw("FindOneSubscribe error", logger.Field("error", err.Error()), logger.Field("userSubscribeId", req.UserSubscribeId))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), " FindOneSubscribe error: %v", err.Error())
	}

	status := userSub.Status
	switch userSub.Status {
	case 0, 1: // pending or active
		status = 5 // set status to stopped
	case 5: // stopped
		status = 1 // set status to active
	default:
		l.Errorw("invalid user subscribe status", logger.Field("userSubscribeId", req.UserSubscribeId), logger.Field("status", userSub.Status))
		return errors.Wrapf(xerr.NewErrCodeMsg(xerr.ERROR, "invalid subscribe status"), "invalid user subscribe status: %d", userSub.Status)
	}

	err = l.svcCtx.UserModel.UpdateSubscribeFields(l.ctx, userSub, modelUser.UserSubscribeFieldsUpdate{
		Status: &status,
	})
	if err != nil {
		l.Errorw("UpdateSubscribe error", logger.Field("error", err.Error()), logger.Field("userSubscribeId", req.UserSubscribeId))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), " UpdateSubscribe error: %v", err.Error())
	}
	userSub.Status = status

	// Clear user subscribe cache
	if err = l.svcCtx.UserModel.ClearSubscribeCache(l.ctx, userSub); err != nil {
		l.Errorw("ClearSubscribeCache failed:", logger.Field("error", err.Error()), logger.Field("userSubscribeId", userSub.Id))
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "ClearSubscribeCache failed: %v", err.Error())
	}
	if err = subscribecache.ClearBySnapshot(l.ctx, l.svcCtx.DB, l.svcCtx.Redis, userSub.ProductSnapshot); err != nil {
		l.Errorw("failed to clear snapshot server user cache", logger.Field("error", err.Error()), logger.Field("subscribeId", userSub.SubscribeId))
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "failed to clear snapshot server user cache: %v", err.Error())
	}
	// Clear subscribe cache
	if err = l.svcCtx.SubscribeModel.ClearCache(l.ctx, userSub.SubscribeId); err != nil {
		l.Errorw("failed to clear subscribe cache", logger.Field("error", err.Error()), logger.Field("subscribeId", userSub.SubscribeId))
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "failed to clear subscribe cache: %v", err.Error())
	}

	return nil
}
