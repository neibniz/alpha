package order

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type orderWalletDeduction struct {
	Commission int64
	Gift       int64
	Provider   int64
}

// quoteOrderWalletDeduction applies the fixed settlement order:
// commission balance, gift balance, then the selected payment provider.
func quoteOrderWalletDeduction(amount int64, u *user.User) orderWalletDeduction {
	result := orderWalletDeduction{Provider: max(amount, 0)}
	if result.Provider == 0 || u == nil {
		return result
	}

	result.Commission = min(max(u.Commission, 0), result.Provider)
	result.Provider -= result.Commission
	result.Gift = min(max(u.GiftAmount, 0), result.Provider)
	result.Provider -= result.Gift
	return result
}

// applyOrderWalletDeduction expects lockedUser to be selected FOR UPDATE by the
// caller. Both balance changes and their audit records are committed together.
func applyOrderWalletDeduction(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
	db *gorm.DB,
	lockedUser *user.User,
	orderNo string,
	deduction orderWalletDeduction,
	giftRemark string,
) error {
	if deduction.Commission == 0 && deduction.Gift == 0 {
		return nil
	}
	if lockedUser == nil || lockedUser.Id <= 0 {
		return errors.Wrap(xerr.NewErrCode(xerr.UserNotExist), "locked user is required")
	}
	if deduction.Commission < 0 || deduction.Gift < 0 {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "wallet deduction must not be negative")
	}
	if lockedUser.Commission < deduction.Commission || lockedUser.GiftAmount < deduction.Gift {
		return errors.Wrap(xerr.NewErrCode(xerr.InsufficientBalance), "wallet balance changed; refresh and retry")
	}

	now := time.Now()
	if deduction.Commission > 0 {
		lockedUser.Commission -= deduction.Commission
		if err := svcCtx.UserModel.UpdateUserCommission(ctx, lockedUser.Id, lockedUser.Commission, db); err != nil {
			return err
		}
		commissionLog := log.Commission{
			Type:      log.CommissionTypePayment,
			Amount:    deduction.Commission,
			OrderNo:   orderNo,
			Balance:   lockedUser.Commission,
			Timestamp: now.UnixMilli(),
		}
		content, err := commissionLog.Marshal()
		if err != nil {
			return err
		}
		if err := db.WithContext(ctx).Create(&log.SystemLog{
			Type:     log.TypeCommission.Uint8(),
			Date:     now.Format(time.DateOnly),
			ObjectID: lockedUser.Id,
			Content:  string(content),
		}).Error; err != nil {
			return err
		}
	}

	if deduction.Gift > 0 {
		lockedUser.GiftAmount -= deduction.Gift
		if err := svcCtx.UserModel.UpdateUserGiftAmount(ctx, lockedUser.Id, lockedUser.GiftAmount, db); err != nil {
			return err
		}
		giftLog := log.Gift{
			Type:        log.GiftTypeReduce,
			OrderNo:     orderNo,
			SubscribeId: 0,
			Amount:      deduction.Gift,
			Balance:     lockedUser.GiftAmount,
			Remark:      giftRemark,
			Timestamp:   now.UnixMilli(),
		}
		content, err := giftLog.Marshal()
		if err != nil {
			return err
		}
		if err := db.WithContext(ctx).Create(&log.SystemLog{
			Type:     log.TypeGift.Uint8(),
			Date:     now.Format(time.DateOnly),
			ObjectID: lockedUser.Id,
			Content:  string(content),
		}).Error; err != nil {
			return err
		}
	}

	return nil
}
