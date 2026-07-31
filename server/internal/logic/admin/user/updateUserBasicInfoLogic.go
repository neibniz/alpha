package user

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/authsession"
	"github.com/neibniz/alpha/server/internal/model/log"
	userModel "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type UpdateUserBasicInfoLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUpdateUserBasicInfoLogic Update user basic info
func NewUpdateUserBasicInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserBasicInfoLogic {
	return &UpdateUserBasicInfoLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserBasicInfoLogic) UpdateUserBasicInfo(req *types.UpdateUserBasiceInfoRequest) error {
	userInfo, err := l.svcCtx.UserModel.FindOneForBasicUpdate(l.ctx, req.UserId)
	if err != nil {
		l.Errorw("[UpdateUserBasicInfoLogic] Find User Error:", logger.Field("err", err.Error()), logger.Field("userId", req.UserId))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Find User Error")
	}

	isDemo := strings.ToLower(os.Getenv("ALPHA_MODE")) == "demo"

	if req.Avatar != "" && !tool.IsValidImageSize(req.Avatar, 1024) {
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "Invalid Image Size")
	}
	if err = l.ensureAdminStateTransition(userInfo, req); err != nil {
		return err
	}

	if userInfo.Balance != req.Balance {
		change := req.Balance - userInfo.Balance
		balanceLog := log.Balance{
			Type:      log.BalanceTypeAdjust,
			Amount:    change,
			OrderNo:   "",
			Balance:   req.Balance,
			Timestamp: time.Now().UnixMilli(),
		}
		content, _ := balanceLog.Marshal()

		err = l.svcCtx.LogModel.Insert(l.ctx, &log.SystemLog{
			Type:     log.TypeBalance.Uint8(),
			Date:     time.Now().Format(time.DateOnly),
			ObjectID: userInfo.Id,
			Content:  string(content),
		})
		if err != nil {
			l.Errorw("[UpdateUserBasicInfoLogic] Insert Balance Log Error:", logger.Field("err", err.Error()), logger.Field("userId", req.UserId))
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "Insert Balance Log Error")
		}
	}

	if userInfo.GiftAmount != req.GiftAmount {
		change := req.GiftAmount - userInfo.GiftAmount
		if change != 0 {
			var changeType uint16
			if userInfo.GiftAmount < req.GiftAmount {
				changeType = log.GiftTypeIncrease
			} else {
				changeType = log.GiftTypeReduce
			}
			giftLog := log.Gift{
				Type:      changeType,
				Amount:    change,
				Balance:   req.GiftAmount,
				Remark:    "Admin adjustment",
				Timestamp: time.Now().UnixMilli(),
			}
			content, _ := giftLog.Marshal()
			// Add gift amount change log
			err = l.svcCtx.LogModel.Insert(l.ctx, &log.SystemLog{
				Type:     log.TypeGift.Uint8(),
				Date:     time.Now().Format(time.DateOnly),
				ObjectID: userInfo.Id,
				Content:  string(content),
			})
			if err != nil {
				l.Errorw("[UpdateUserBasicInfoLogic] Insert Balance Log Error:", logger.Field("err", err.Error()), logger.Field("userId", req.UserId))
				return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "Insert Balance Log Error")
			}
		}
	}

	if req.Commission != userInfo.Commission {

		commentLog := log.Commission{
			Type:      log.CommissionTypeAdjust,
			Amount:    req.Commission - userInfo.Commission,
			Timestamp: time.Now().UnixMilli(),
		}

		content, _ := commentLog.Marshal()
		err = l.svcCtx.LogModel.Insert(l.ctx, &log.SystemLog{
			Type:     log.TypeCommission.Uint8(),
			Date:     time.Now().Format(time.DateOnly),
			ObjectID: userInfo.Id,
			Content:  string(content),
		})
		if err != nil {
			l.Errorw("[UpdateUserBasicInfoLogic] Insert Commission Log Error:", logger.Field("err", err.Error()), logger.Field("userId", req.UserId))
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "Insert Commission Log Error")
		}
	}

	update := userModel.UserBasicFieldsUpdate{
		Avatar:             userModel.UpdateValue(req.Avatar),
		Balance:            userModel.UpdateValue(req.Balance),
		Commission:         userModel.UpdateValue(req.Commission),
		ReferralPercentage: userModel.UpdateValue(req.ReferralPercentage),
		OnlyFirstPurchase:  userModel.UpdateValue(req.OnlyFirstPurchase),
		GiftAmount:         userModel.UpdateValue(req.GiftAmount),
		ReferCode:          userModel.UpdateValue(req.ReferCode),
		RefererID:          userModel.UpdateValue(req.RefererId),
		Enable:             userModel.UpdateValue(req.Enable),
		IsAdmin:            userModel.UpdateValue(req.IsAdmin),
	}

	if req.Password != "" {
		if userInfo.Id == 2 && isDemo {
			return errors.Wrapf(xerr.NewErrCodeMsg(503, "Demo mode does not allow modification of the admin user password"), "UpdateUserBasicInfo failed: cannot update admin user password in demo mode")
		}
		passwordHash, hashErr := tool.EncodePassWord(req.Password)
		if hashErr != nil {
			return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "password does not meet policy")
		}
		update.Password = userModel.UpdateValue(passwordHash)
	}

	err = l.svcCtx.UserModel.UpdateUserBasicFields(l.ctx, req.UserId, update)
	if err != nil {
		l.Errorw("[UpdateUserBasicInfoLogic] Update User Error:", logger.Field("err", err.Error()), logger.Field("userId", req.UserId))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "Update User Error")
	}
	adminChanged := userBoolValue(userInfo.IsAdmin) != req.IsAdmin
	if req.Password != "" || !req.Enable || adminChanged {
		if err := authsession.RevokeUser(l.ctx, l.svcCtx.Redis, req.UserId); err != nil {
			return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "revoke user authentication sessions")
		}
	}

	return nil
}

func (l *UpdateUserBasicInfoLogic) ensureAdminStateTransition(userInfo *userModel.User, req *types.UpdateUserBasiceInfoRequest) error {
	if userInfo == nil || req == nil {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid user update")
	}

	currentlyAdmin := userBoolValue(userInfo.IsAdmin)
	currentlyEnabled := userBoolValue(userInfo.Enable)
	current, _ := l.ctx.Value(constant.CtxKeyUser).(*userModel.User)
	if current != nil && current.Id == userInfo.Id {
		if currentlyAdmin && !req.IsAdmin {
			return errors.Wrap(xerr.NewErrCodeMsg(400, "Cannot demote the current admin user"), "cannot demote current admin user")
		}
		if currentlyEnabled && !req.Enable {
			return errors.Wrap(xerr.NewErrCodeMsg(400, "Cannot disable the current admin user"), "cannot disable current admin user")
		}
	}

	if !currentlyAdmin || !currentlyEnabled || (req.IsAdmin && req.Enable) {
		return nil
	}

	var activeAdminCount int64
	if err := l.svcCtx.DB.WithContext(l.ctx).
		Model(&userModel.User{}).
		Where("is_admin = ? AND enable = ?", true, true).
		Count(&activeAdminCount).Error; err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "count active admin users: %v", err)
	}
	if activeAdminCount <= 1 {
		return errors.Wrap(xerr.NewErrCodeMsg(400, "Cannot disable or demote the last active admin user"), "last active admin user must remain enabled")
	}
	return nil
}
