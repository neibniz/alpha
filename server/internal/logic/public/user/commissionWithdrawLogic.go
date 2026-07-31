package user

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm/clause"
)

type CommissionWithdrawLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Commission Withdraw
func NewCommissionWithdrawLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CommissionWithdrawLogic {
	return &CommissionWithdrawLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CommissionWithdrawLogic) CommissionWithdraw(req *types.CommissionWithdrawRequest) (resp *types.WithdrawalLog, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}

	tx := l.svcCtx.DB.WithContext(l.ctx).Begin()
	var lockedUser user.User
	if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&user.User{}).
		Select(commissionWithdrawUserColumns).
		Where("id = ?", u.Id).
		First(&lockedUser).Error; err != nil {
		tx.Rollback()
		l.Errorf("Failed to lock user %d commission balance: %v", u.Id, err)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Failed to lock user %d commission balance: %v", u.Id, err)
	}

	if lockedUser.Commission < req.Amount {
		tx.Rollback()
		logger.Errorf("User %d has insufficient commission balance: %.2f, requested: %.2f", u.Id, float64(lockedUser.Commission)/100, float64(req.Amount)/100)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.UserCommissionNotEnough), "User %d has insufficient commission balance", u.Id)
	}

	// update user commission balance
	lockedUser.Commission -= req.Amount
	if err = l.svcCtx.UserModel.UpdateUserCommission(l.ctx, lockedUser.Id, lockedUser.Commission, tx); err != nil {
		tx.Rollback()
		l.Errorf("Failed to update user %d commission balance: %v", u.Id, err)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "Failed to update user %d commission balance: %v", u.Id, err)
	}

	now := time.Now()
	logInfo := log.Commission{
		Type:      log.CommissionTypeConvertBalance,
		Amount:    req.Amount,
		Timestamp: now.UnixMilli(),
	}
	b, err := logInfo.Marshal()

	if err != nil {
		tx.Rollback()
		l.Errorf("Failed to marshal commission log for user %d: %v", u.Id, err)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "Failed to marshal commission log for user %d: %v", u.Id, err)
	}

	err = tx.Model(log.SystemLog{}).Create(&log.SystemLog{
		Type:      log.TypeCommission.Uint8(),
		Date:      now.Format("2006-01-02"),
		ObjectID:  u.Id,
		Content:   string(b),
		CreatedAt: now,
	}).Error

	if err != nil {
		tx.Rollback()
		l.Errorf("Failed to create commission log for user %d: %v", u.Id, err)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "Failed to create commission log for user %d: %v", u.Id, err)
	}

	privacyCodec := l.svcCtx.Privacy
	if privacyCodec == nil {
		privacyCodec = privacy.DisabledCodec()
	}
	protectedContent, err := privacyCodec.Seal(privacy.WithdrawalScope(u.Id, "content"), req.Content)
	if err != nil {
		tx.Rollback()
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "encrypt withdrawal content")
	}
	withdrawal := user.Withdrawal{
		UserId:  u.Id,
		Amount:  req.Amount,
		Content: protectedContent,
		Status:  0,
		Reason:  "",
	}
	err = tx.Create(&withdrawal).Error

	if err != nil {
		tx.Rollback()
		l.Errorf("Failed to create withdrawal log for user %d: %v", u.Id, err)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "Failed to create withdrawal log for user %d: %v", u.Id, err)
	}
	if err = tx.Commit().Error; err != nil {
		l.Errorf("Transaction commit failed for user %d withdrawal: %v", u.Id, err)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "Transaction commit failed for user %d withdrawal: %v", u.Id, err)
	}

	withdrawal.Content = req.Content
	result := withdrawalFromModel(withdrawal)
	return &result, nil
}
