package order

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/orderaudit"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/xerr"

	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/pkg/errors"
	"gorm.io/gorm/clause"
)

type ResetTrafficLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Reset traffic
func NewResetTrafficLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetTrafficLogic {
	return &ResetTrafficLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResetTrafficLogic) ResetTraffic(req *types.ResetTrafficOrderRequest) (resp *types.ResetTrafficOrderResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	requestID, err := normalizeOrderRequestID(req.RequestId)
	if err != nil {
		return nil, err
	}
	if orderNo, found, lookupErr := findIdempotentOrderNo(l.ctx, l.svcCtx, u.Id, OrderTypeResetTraffic, requestID); lookupErr != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find idempotent order: %v", lookupErr)
	} else if found {
		return &types.ResetTrafficOrderResponse{OrderNo: orderNo}, nil
	}
	// find user subscription
	userSubscribe, err := l.svcCtx.UserModel.FindOneUserSubscribeForOrderAction(l.ctx, req.UserSubscribeID)
	if err != nil {
		l.Errorw("[ResetTraffic] Database query error", logger.Field("error", err.Error()), logger.Field("UserSubscribeID", req.UserSubscribeID))
		return nil, errors.Wrapf(err, "find user subscribe error: %v", err.Error())
	}
	if userSubscribe.UserId != u.Id {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "subscription does not belong to current user")
	}
	// Paid reset orders use the current product policy and replacement price.
	sub, err := l.svcCtx.SubscribeModel.FindOneForPublicPurchase(l.ctx, userSubscribe.SubscribeId)
	if err != nil {
		l.Errorw("[ResetTraffic] current product not found", logger.Field("UserSubscribeID", req.UserSubscribeID), logger.Field("subscribe_id", userSubscribe.SubscribeId))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "current product not found: %v", err)
	}
	if err = ensureResetTrafficActionAllowed(userSubscribe.Status, userSubscribe.ExpireTime, sub, time.Now()); err != nil {
		return nil, err
	}
	amount := sub.Replacement
	walletDeduction := quoteOrderWalletDeduction(amount, u)
	amount = walletDeduction.Provider
	var paymentID int64
	method := "free"
	var feeAmount int64
	if amount > 0 {
		// find payment method
		payment, err := l.svcCtx.PaymentModel.FindOneForPublicOrder(l.ctx, req.Payment)
		if err != nil {
			l.Errorw("[ResetTraffic] Database query error", logger.Field("error", err.Error()), logger.Field("payment", req.Payment))
			return nil, errors.Wrapf(err, "find payment error: %v", err.Error())
		}
		if err = ensurePaymentEnabled(payment); err != nil {
			return nil, err
		}
		paymentID = payment.Id
		method = payment.Platform
		// Calculate the handling fee
		feeAmount = calculateFee(amount, payment)
	}
	if err = enforceMembershipOrderPolicies(l.ctx, l.svcCtx, u, amount+feeAmount); err != nil {
		return nil, err
	}
	// create order
	orderNo, err := tool.GenerateTradeNo()
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "generate order number")
	}
	productSnapshot := ordersnapshot.FromSubscribe(sub)
	orderInfo := order.Order{
		Id:                  0,
		ParentId:            userSubscribe.OrderId,
		UserId:              u.Id,
		OrderNo:             orderNo,
		RequestId:           requestID,
		Type:                OrderTypeResetTraffic,
		Price:               sub.Replacement,
		Amount:              amount + feeAmount,
		GiftAmount:          walletDeduction.Gift,
		CommissionDeduction: walletDeduction.Commission,
		FeeAmount:           feeAmount,
		PaymentId:           paymentID,
		Method:              method,
		Status:              1,
		SubscribeId:         userSubscribe.SubscribeId,
		SubscribeToken:      userSubscribe.Token,
		ProductName:         productSnapshot.Name,
		ProductSnapshot:     ordersnapshot.Marshal(productSnapshot),
	}
	orderaudit.FillNetwork(l.ctx, l.svcCtx, &orderInfo)
	if err = enqueueDeferredCloseOrder(l.ctx, l.svcCtx.Queue, l.Logger, orderInfo.OrderNo); err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "enqueue close order task error: %v", err.Error())
	}
	// Database transaction
	err = l.svcCtx.DB.Transaction(func(db *gorm.DB) error {
		var lockedUserSubscribe user.Subscribe
		if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", userSubscribe.Id).
			First(&lockedUserSubscribe).Error; err != nil {
			return err
		}
		if lockedUserSubscribe.UserId != u.Id || lockedUserSubscribe.SubscribeId != sub.Id || lockedUserSubscribe.Token != userSubscribe.Token {
			return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "subscription changed; refresh and retry")
		}
		var lockedUser *user.User
		if orderInfo.CommissionDeduction > 0 || orderInfo.GiftAmount > 0 {
			lockedUser = &user.User{}
			if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
				Model(&user.User{}).
				Select(orderWalletUserColumns).
				Where("id = ?", u.Id).
				First(lockedUser).Error; err != nil {
				l.Errorw("[ResetTraffic] Database query error", logger.Field("error", err.Error()), logger.Field("user", u))
				return err
			}
		}
		lockedProduct, err := lockCurrentSubscribeForOrder(l.ctx, db, sub.Id)
		if err != nil {
			return err
		}
		if err := ensureResetTrafficActionAllowed(lockedUserSubscribe.Status, lockedUserSubscribe.ExpireTime, lockedProduct, time.Now()); err != nil {
			return err
		}
		if err := ensureProductSnapshotCurrent(lockedProduct, orderInfo.ProductSnapshot); err != nil {
			return err
		}
		if lockedUser != nil {
			if err := applyOrderWalletDeduction(l.ctx, l.svcCtx, db, lockedUser, orderInfo.OrderNo, walletDeduction, "Reset traffic order deduction"); err != nil {
				l.Errorw("[ResetTraffic] Wallet deduction failed", logger.Field("error", err.Error()), logger.Field("user_id", u.Id))
				return err
			}
		}
		// insert order
		return db.Model(&order.Order{}).Create(&orderInfo).Error
	})
	if err != nil {
		if existingOrderNo, found, lookupErr := findIdempotentOrderNo(l.ctx, l.svcCtx, u.Id, OrderTypeResetTraffic, requestID); lookupErr == nil && found {
			return &types.ResetTrafficOrderResponse{OrderNo: existingOrderNo}, nil
		}
		l.Errorw("[ResetTraffic] Database insert error", logger.Field("error", err.Error()), logger.Field("order", orderInfo))
		return nil, errors.Wrapf(err, "insert order error: %v", err.Error())
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventActivity)
	}
	return &types.ResetTrafficOrderResponse{
		OrderNo: orderInfo.OrderNo,
	}, nil
}
