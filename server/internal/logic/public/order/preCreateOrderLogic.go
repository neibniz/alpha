package order

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/couponpolicy"
	"github.com/neibniz/alpha/server/internal/subscribeview"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type PreCreateOrderLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewPreCreateOrderLogic creates a new pre-create order logic instance for order preview operations.
// It initializes the logger with context and sets up the service context for database operations.
func NewPreCreateOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreCreateOrderLogic {
	return &PreCreateOrderLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PreCreateOrder calculates order pricing preview including discounts, coupons, gift amounts, and fees
// without actually creating an order. It validates subscription plans, coupons, and payment methods
// to provide accurate pricing information for the frontend order preview.
func (l *PreCreateOrderLogic) PreCreateOrder(req *types.PurchaseOrderRequest) (resp *types.PreOrderResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}

	if req.Quantity <= 0 {
		l.Debugf("[PreCreateOrder] Quantity is less than or equal to 0, setting to 1")
		req.Quantity = 1
	}
	if err = requireActiveMember(u); err != nil {
		return nil, err
	}

	// find subscribe plan
	sub, err := l.svcCtx.SubscribeModel.FindOneForPublicPurchase(l.ctx, req.SubscribeId)
	if err != nil {
		l.Errorw("[PreCreateOrder] Database query error", logger.Field("error", err.Error()), logger.Field("subscribe_id", req.SubscribeId))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find subscribe error: %v", err.Error())
	}
	if sub.UserVisible == nil || !*sub.UserVisible {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "subscribe not visible")
	}
	if err = ensureNewPurchaseAvailable(sub, time.Now()); err != nil {
		return nil, err
	}
	if err = ensureProductQuantity(sub, req.Quantity); err != nil {
		return nil, err
	}
	if err = EnforceMembershipSubscriptionExpiryPolicy(l.ctx, l.svcCtx, u, sub, req.Quantity, nil); err != nil {
		return nil, err
	}
	if l.svcCtx.ConfigSnapshot().Subscribe.SingleModel {
		count, err := l.svcCtx.UserModel.CountUserSubscribe(l.ctx, u.Id)
		if err != nil {
			l.Errorw("[PreCreateOrder] Database query error", logger.Field("error", err.Error()), logger.Field("user_id", u.Id))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "count user subscription error: %v", err.Error())
		}
		if count > 0 {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.UserSubscribeExist), "user has subscription")
		}
	}
	if sub.Quota > 0 {
		count, err := l.svcCtx.UserModel.CountUserSubscribeByProduct(l.ctx, u.Id, req.SubscribeId)
		if err != nil {
			l.Errorw("[PreCreateOrder] Database query error", logger.Field("error", err.Error()), logger.Field("user_id", u.Id), logger.Field("subscribe_id", req.SubscribeId))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "count user subscription error: %v", err.Error())
		}
		if count >= sub.Quota {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.SubscribeQuotaLimit), "quota limit")
		}
	}
	var discount float64 = 1
	if sub.Discount != "" {
		discount = getDiscount(subscribeview.ParseDiscountLenient(sub.Discount), req.Quantity)
	}
	price := sub.UnitPrice * req.Quantity

	amount := int64(float64(price) * discount)
	discountAmount := price - amount
	if amount > MaxOrderAmount {
		l.Errorw("[PreCreateOrder] Order amount exceeds maximum limit",
			logger.Field("amount", amount),
			logger.Field("max", MaxOrderAmount),
			logger.Field("user_id", u.Id),
			logger.Field("subscribe_id", req.SubscribeId))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "order amount exceeds maximum limit")
	}
	var couponAmount int64
	if req.Coupon != "" {
		couponInfo, err := l.svcCtx.CouponModel.FindOneByCode(l.ctx, req.Coupon)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.Wrapf(xerr.NewErrCode(xerr.CouponNotExist), "coupon not found")
			}
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find coupon error: %v", err.Error())
		}
		if err = couponpolicy.Validate(couponInfo, req.SubscribeId, time.Now()); err != nil {
			return nil, err
		}
		limitReached, err := couponUsageReached(l.ctx, l.svcCtx.DB, u.Id, req.Coupon, couponInfo.UserLimit)
		if err != nil {
			l.Errorw("[PreCreateOrder] Database query error", logger.Field("error", err.Error()), logger.Field("user_id", u.Id), logger.Field("coupon", req.Coupon))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find coupon error: %v", err.Error())
		}
		if limitReached {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.CouponInsufficientUsage), "coupon limit exceeded")
		}

		couponAmount = calculateCoupon(amount, couponInfo)
	}
	amount -= couponAmount
	walletDeduction := quoteOrderWalletDeduction(amount, u)
	amount = walletDeduction.Provider
	var feeAmount int64
	if req.Payment != 0 {
		payment, err := l.svcCtx.PaymentModel.FindOneForPublicOrder(l.ctx, req.Payment)
		if err != nil {
			l.Errorw("[PreCreateOrder] Database query error", logger.Field("error", err.Error()), logger.Field("payment", req.Payment))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find payment method error: %v", err.Error())
		}
		if err = ensurePaymentEnabled(payment); err != nil {
			return nil, err
		}
		// Calculate the handling fee
		if amount > 0 {
			feeAmount = calculateFee(amount, payment)
		}
		amount += feeAmount
		if amount > MaxOrderAmount {
			l.Errorw("[PreCreateOrder] Final order amount exceeds maximum limit after fee",
				logger.Field("amount", amount),
				logger.Field("max", MaxOrderAmount),
				logger.Field("user_id", u.Id))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "order amount exceeds maximum limit")
		}
	}
	if err = enforceMembershipOrderPolicies(l.ctx, l.svcCtx, u, amount); err != nil {
		return nil, err
	}

	resp = &types.PreOrderResponse{
		Price:               price,
		Amount:              amount,
		Discount:            discountAmount,
		GiftAmount:          walletDeduction.Gift,
		CommissionDeduction: walletDeduction.Commission,
		Coupon:              req.Coupon,
		CouponDiscount:      couponAmount,
		FeeAmount:           feeAmount,
	}
	return
}
