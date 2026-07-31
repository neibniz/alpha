package user

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/log"
	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/subscribecache"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/neibniz/alpha/server/internal/model/user"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type UnsubscribeLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUnsubscribeLogic creates a new instance of UnsubscribeLogic for handling subscription cancellation
func NewUnsubscribeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnsubscribeLogic {
	return &UnsubscribeLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Unsubscribe handles the subscription cancellation process with proper refund distribution
// It prioritizes refunding to gift amount for balance-paid orders, then to regular balance
func (l *UnsubscribeLogic) Unsubscribe(req *types.UnsubscribeRequest) error {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}

	// find user subscription by ID
	userSub, err := l.svcCtx.UserModel.FindOneSubscribeForMutation(l.ctx, req.Id)
	if err != nil {
		l.Errorw("FindOneSubscribe failed", logger.Field("error", err.Error()), logger.Field("reqId", req.Id))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "FindOneSubscribe failed: %v", err.Error())
	}

	activate := []uint8{0, 1, 2}

	if !tool.Contains(activate, userSub.Status) {
		// Only active (2) or paused (5) subscriptions can be cancelled
		l.Errorw("Subscription status invalid for cancellation", logger.Field("userSubscribeId", userSub.Id), logger.Field("status", userSub.Status))
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "Subscription status invalid for cancellation")
	}

	// Calculate the remaining amount to refund based on unused subscription time/traffic
	remainingAmount, err := CalculateRemainingAmount(l.ctx, l.svcCtx, req.Id)
	if err != nil {
		return err
	}
	orderDetails, err := l.svcCtx.OrderModel.FindOneDetails(l.ctx, userSub.OrderId)
	if err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find subscription funding orders: %v", err)
	}
	commissionFunding, giftFunding := refundableWalletFunding(orderDetails)

	// Process unsubscription in a database transaction to ensure data consistency
	err = l.svcCtx.UserModel.Transaction(l.ctx, func(db *gorm.DB) error {
		// Find and update subscription status to cancelled (status = 4)
		cancelledStatus := uint8(4)
		if err = l.svcCtx.UserModel.UpdateSubscribeFields(l.ctx, userSub, user.UserSubscribeFieldsUpdate{
			Status: &cancelledStatus,
		}, db); err != nil {
			return err
		}
		userSub.Status = cancelledStatus

		var lockedUser user.User
		if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
			Model(&user.User{}).
			Select(unsubscribeRefundUserColumns).
			Where("id = ?", u.Id).
			First(&lockedUser).Error; err != nil {
			return err
		}
		refundRemainder := max(remainingAmount, 0)
		commissionRefund := min(max(commissionFunding, 0), refundRemainder)
		refundRemainder -= commissionRefund
		giftRefund := min(max(giftFunding, 0), refundRemainder)
		refundRemainder -= giftRefund
		balanceRefund := refundRemainder
		now := time.Now()

		if commissionRefund > 0 {
			lockedUser.Commission += commissionRefund
			if err := l.svcCtx.UserModel.UpdateUserCommission(l.ctx, lockedUser.Id, lockedUser.Commission, db); err != nil {
				return err
			}
			commissionLog := log.Commission{
				OrderNo:   orderDetails.OrderNo,
				Amount:    commissionRefund,
				Type:      log.CommissionTypePaymentRefund,
				Balance:   lockedUser.Commission,
				Timestamp: now.UnixMilli(),
			}
			content, marshalErr := commissionLog.Marshal()
			if marshalErr != nil {
				return marshalErr
			}
			if err := db.WithContext(l.ctx).Create(&log.SystemLog{
				Type:     log.TypeCommission.Uint8(),
				Date:     now.Format(time.DateOnly),
				ObjectID: u.Id,
				Content:  string(content),
			}).Error; err != nil {
				return err
			}
		}

		if giftRefund > 0 {
			lockedUser.GiftAmount += giftRefund
			if err := l.svcCtx.UserModel.UpdateUserGiftAmount(l.ctx, lockedUser.Id, lockedUser.GiftAmount, db); err != nil {
				return err
			}
			giftLog := log.Gift{
				SubscribeId: userSub.Id,
				OrderNo:     orderDetails.OrderNo,
				Type:        log.GiftTypeIncrease,
				Amount:      giftRefund,
				Balance:     lockedUser.GiftAmount,
				Remark:      "Unsubscribe refund",
				Timestamp:   now.UnixMilli(),
			}
			content, marshalErr := giftLog.Marshal()
			if marshalErr != nil {
				return marshalErr
			}
			if err := db.WithContext(l.ctx).Create(&log.SystemLog{
				Type:     log.TypeGift.Uint8(),
				Date:     now.Format(time.DateOnly),
				ObjectID: u.Id,
				Content:  string(content),
			}).Error; err != nil {
				return err
			}
		}

		if balanceRefund > 0 {
			lockedUser.Balance += balanceRefund
			if err := l.svcCtx.UserModel.UpdateUserBalance(l.ctx, lockedUser.Id, lockedUser.Balance, db); err != nil {
				return err
			}
			balanceLog := log.Balance{
				OrderNo:   orderDetails.OrderNo,
				Amount:    balanceRefund,
				Type:      log.BalanceTypeRefund,
				Balance:   lockedUser.Balance,
				Timestamp: now.UnixMilli(),
			}
			content, marshalErr := balanceLog.Marshal()
			if marshalErr != nil {
				return marshalErr
			}
			if err := db.WithContext(l.ctx).Create(&log.SystemLog{
				Type:     log.TypeBalance.Uint8(),
				Date:     now.Format(time.DateOnly),
				ObjectID: u.Id,
				Content:  string(content),
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		l.Errorw("Unsubscribe transaction failed", logger.Field("error", err.Error()), logger.Field("userId", u.Id), logger.Field("reqId", req.Id))
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "Unsubscribe transaction failed: %v", err.Error())
	}

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

	return err
}

func refundableWalletFunding(orderDetails *modelOrder.Details) (commission, gift int64) {
	if orderDetails == nil {
		return 0, 0
	}
	commission = max(orderDetails.CommissionDeduction, 0)
	gift = max(orderDetails.GiftAmount, 0)
	for _, subOrder := range orderDetails.SubOrders {
		if subOrder == nil || (subOrder.Status != 2 && subOrder.Status != 5) {
			continue
		}
		commission += max(subOrder.CommissionDeduction, 0)
		gift += max(subOrder.GiftAmount, 0)
	}
	return commission, gift
}
