package user

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/subscribecache"
	"github.com/neibniz/alpha/server/internal/subscriptionstate"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type UpdateUserSubscribeLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUpdateUserSubscribeLogic Update user subscribe
func NewUpdateUserSubscribeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserSubscribeLogic {
	return &UpdateUserSubscribeLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserSubscribeLogic) UpdateUserSubscribe(req *types.UpdateUserSubscribeRequest) error {
	if req == nil || req.Traffic == nil || req.ExpiredAt == nil || req.Upload == nil || req.Download == nil ||
		req.UserSubscribeId <= 0 || req.SubscribeId <= 0 ||
		*req.Traffic < 0 || *req.ExpiredAt < 0 || *req.Upload < 0 || *req.Download < 0 {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid subscription adjustment")
	}

	expiredAt := time.UnixMilli(0)
	if *req.ExpiredAt > 0 {
		expiredAt = time.UnixMilli(*req.ExpiredAt)
	}
	operatorID := int64(0)
	if current, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User); ok && current != nil {
		operatorID = current.Id
	}

	var before *user.Subscribe
	var after user.Subscribe
	err := l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		userSub, err := l.svcCtx.UserModel.FindOneSubscribeForMutation(l.ctx, req.UserSubscribeId, tx)
		if err != nil {
			return err
		}
		adjustedAt := time.Now().UTC()

		productSnapshot := userSub.ProductSnapshot
		productName := userSub.ProductName
		resetCycle := userSub.ResetCycle
		if req.SubscribeId != userSub.SubscribeId {
			sub, err := l.svcCtx.SubscribeModel.FindOneForOrderSnapshot(l.ctx, req.SubscribeId)
			if err != nil {
				return err
			}
			snapshot := ordersnapshot.FromSubscribe(sub)
			productName = snapshot.Name
			productSnapshot = ordersnapshot.Marshal(snapshot)
			resetCycle = subscriptionstate.NormalizeResetCycle(snapshot.ResetCycle)
		}

		status := adjustedSubscriptionStatus(
			userSub.Status,
			expiredAt,
			*req.Traffic,
			*req.Download,
			*req.Upload,
			adjustedAt,
		)
		finishedAt, clearFinishedAt := adjustedSubscriptionFinishedAt(userSub, status, adjustedAt)
		nextResetAt := subscriptionstate.NextTrafficResetAt(
			resetCycle,
			userSub.StartTime,
			adjustedAt,
			expiredAt,
			subscriptionstate.TrafficResetLocation(),
		)

		update := user.UserSubscribeFieldsUpdate{
			SubscribeID:     &req.SubscribeId,
			ProductName:     &productName,
			ProductSnapshot: &productSnapshot,
			ExpireTime:      &expiredAt,
			FinishedAt:      finishedAt,
			ClearFinishedAt: clearFinishedAt,
			Traffic:         req.Traffic,
			Download:        req.Download,
			Upload:          req.Upload,
			ResetCycle:      &resetCycle,
			Status:          &status,
		}
		if userSub.Download != *req.Download || userSub.Upload != *req.Upload {
			update.TrafficResetAt = &adjustedAt
		}
		if nextResetAt == nil {
			update.ClearNextTrafficResetAt = true
		} else {
			update.NextTrafficResetAt = nextResetAt
		}

		after = *userSub
		after.SubscribeId = req.SubscribeId
		after.ProductName = productName
		after.ProductSnapshot = productSnapshot
		after.ExpireTime = expiredAt
		after.Traffic = *req.Traffic
		after.Download = *req.Download
		after.Upload = *req.Upload
		after.ResetCycle = resetCycle
		after.NextTrafficResetAt = nextResetAt
		after.Status = status
		if update.TrafficResetAt != nil {
			after.TrafficResetAt = adjustedAt
		}
		if clearFinishedAt {
			after.FinishedAt = nil
		} else if finishedAt != nil {
			value := *finishedAt
			after.FinishedAt = &value
		}

		if err := l.svcCtx.UserModel.UpdateSubscribeFields(l.ctx, userSub, update, tx); err != nil {
			return err
		}

		entry := &log.SubscribeAdjustment{
			UserId:         userSub.UserId,
			OperatorUserId: operatorID,
			Before:         subscriptionAdjustmentState(userSub),
			After:          subscriptionAdjustmentState(&after),
			Timestamp:      adjustedAt.UnixMilli(),
		}
		content, err := entry.Marshal()
		if err != nil {
			return err
		}
		if err := tx.Create(&log.SystemLog{
			Type:     log.TypeAdjustSubscribe.Uint8(),
			Date:     adjustedAt.Format(time.DateOnly),
			ObjectID: userSub.Id,
			EventAt:  adjustedAt.UnixMilli(),
			Success:  true,
			Content:  string(content),
		}).Error; err != nil {
			return err
		}
		before = userSub
		return nil
	})
	if err != nil {
		l.Errorw("failed to adjust user subscription", logger.Field("error", err.Error()), logger.Field("userSubscribeId", req.UserSubscribeId))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "failed to adjust user subscription: %v", err)
	}

	if err := l.svcCtx.UserModel.ClearSubscribeCache(l.ctx, before, &after); err != nil {
		l.Errorw("failed to clear user subscription cache after commit", logger.Field("error", err.Error()), logger.Field("userSubscribeId", req.UserSubscribeId))
	}
	if err := subscribecache.ClearBySnapshot(l.ctx, l.svcCtx.DB, l.svcCtx.Redis, before.ProductSnapshot); err != nil {
		l.Errorw("failed to clear old snapshot server user cache after commit", logger.Field("error", err.Error()), logger.Field("subscribeId", before.SubscribeId))
	}
	if err := subscribecache.ClearBySnapshot(l.ctx, l.svcCtx.DB, l.svcCtx.Redis, after.ProductSnapshot); err != nil {
		l.Errorw("failed to clear new snapshot server user cache after commit", logger.Field("error", err.Error()), logger.Field("subscribeId", after.SubscribeId))
	}
	if err := l.svcCtx.SubscribeModel.ClearCache(l.ctx, before.SubscribeId, after.SubscribeId); err != nil {
		l.Errorw("failed to clear product cache after commit", logger.Field("error", err.Error()), logger.Field("subscribeId", before.SubscribeId))
	}
	return nil
}

func adjustedSubscriptionStatus(current uint8, expireAt time.Time, traffic, download, upload int64, now time.Time) uint8 {
	switch current {
	case subscriptionstate.StatusPending, subscriptionstate.StatusDeducted, subscriptionstate.StatusStopped:
		return current
	case subscriptionstate.StatusActive, subscriptionstate.StatusExhausted, subscriptionstate.StatusExpired:
		if subscriptionstate.IsExpired(expireAt, now) {
			return subscriptionstate.StatusExpired
		}
		if subscriptionstate.TrafficExhausted(traffic, download, upload) {
			return subscriptionstate.StatusExhausted
		}
		return subscriptionstate.StatusActive
	default:
		return current
	}
}

func adjustedSubscriptionFinishedAt(current *user.Subscribe, nextStatus uint8, now time.Time) (*time.Time, bool) {
	if nextStatus == subscriptionstate.StatusActive {
		return nil, true
	}
	if nextStatus != subscriptionstate.StatusExhausted && nextStatus != subscriptionstate.StatusExpired {
		return nil, false
	}
	if current.Status == nextStatus && current.FinishedAt != nil {
		return nil, false
	}
	value := now
	return &value, false
}

func subscriptionAdjustmentState(subscribe *user.Subscribe) log.SubscribeAdjustmentState {
	return log.SubscribeAdjustmentState{
		SubscribeId: subscribe.SubscribeId,
		Traffic:     subscribe.Traffic,
		Download:    subscribe.Download,
		Upload:      subscribe.Upload,
		ExpireAt:    subscribe.ExpireTime.UnixMilli(),
		Status:      subscribe.Status,
	}
}
