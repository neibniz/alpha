package order

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/couponpolicy"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/productpolicy"
	"github.com/neibniz/alpha/server/internal/subscribeview"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type PreRenewalLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPreRenewalLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreRenewalLogic {
	return &PreRenewalLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreRenewalLogic) PreRenewal(req *types.PreRenewalOrderRequest) (*types.PreRenewalOrderResponse, error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "current user is not available")
	}
	if err := requireActiveMember(u); err != nil {
		return nil, err
	}

	userSubscribe, err := l.svcCtx.UserModel.FindOneUserSubscribeForOrderAction(l.ctx, req.UserSubscribeID)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find user subscription: %v", err)
	}
	if userSubscribe.UserId != u.Id {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "subscription does not belong to current user")
	}

	product, err := l.svcCtx.SubscribeModel.FindOneForPublicPurchase(l.ctx, userSubscribe.SubscribeId)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find current product: %v", err)
	}
	if err := ensureRenewalActionAllowed(userSubscribe.Status, userSubscribe.ExpireTime, product, time.Now()); err != nil {
		return nil, err
	}
	if req.Quantity <= 0 {
		req.Quantity = productpolicy.NormalizeMinQuantity(product.MinQuantity)
	}
	if err := ensureProductQuantity(product, req.Quantity); err != nil {
		return nil, err
	}
	if err := EnforceMembershipSubscriptionExpiryPolicy(l.ctx, l.svcCtx, u, product, req.Quantity, &userSubscribe.ExpireTime); err != nil {
		return nil, err
	}

	discountRate := 1.0
	if product.Discount != "" {
		discountRate = getDiscount(subscribeview.ParseDiscountLenient(product.Discount), req.Quantity)
	}
	price := product.UnitPrice * req.Quantity
	amount := int64(float64(price) * discountRate)
	discountAmount := price - amount
	if amount > MaxOrderAmount {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "order amount exceeds maximum limit")
	}

	var couponAmount int64
	if req.Coupon != "" {
		couponInfo, couponErr := l.svcCtx.CouponModel.FindOneByCode(l.ctx, req.Coupon)
		if couponErr != nil {
			if errors.Is(couponErr, gorm.ErrRecordNotFound) {
				return nil, errors.Wrap(xerr.NewErrCode(xerr.CouponNotExist), "coupon not found")
			}
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find coupon: %v", couponErr)
		}
		if err := couponpolicy.Validate(couponInfo, product.Id, time.Now()); err != nil {
			return nil, err
		}
		limitReached, usageErr := couponUsageReached(l.ctx, l.svcCtx.DB, u.Id, req.Coupon, couponInfo.UserLimit)
		if usageErr != nil {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "check coupon usage: %v", usageErr)
		}
		if limitReached {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.CouponInsufficientUsage), "coupon limit exceeded")
		}
		couponAmount = calculateCoupon(amount, couponInfo)
	}
	amount -= couponAmount
	walletDeduction := quoteOrderWalletDeduction(amount, u)
	amount = walletDeduction.Provider

	var feeAmount int64
	if req.Payment != 0 {
		payment, paymentErr := l.svcCtx.PaymentModel.FindOneForPublicOrder(l.ctx, req.Payment)
		if paymentErr != nil {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find payment method: %v", paymentErr)
		}
		if err := ensurePaymentEnabled(payment); err != nil {
			return nil, err
		}
		if amount > 0 {
			feeAmount = calculateFee(amount, payment)
			amount += feeAmount
		}
	}
	if amount > MaxOrderAmount {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "order amount exceeds maximum limit")
	}
	if err := enforceMembershipOrderPolicies(l.ctx, l.svcCtx, u, amount); err != nil {
		return nil, err
	}

	return &types.PreRenewalOrderResponse{
		Quantity:            req.Quantity,
		Price:               price,
		Amount:              amount,
		Discount:            discountAmount,
		GiftAmount:          walletDeduction.Gift,
		CommissionDeduction: walletDeduction.Commission,
		Coupon:              req.Coupon,
		CouponDiscount:      couponAmount,
		FeeAmount:           feeAmount,
		Product:             subscribeview.PublicSubscribeFromModel(product),
	}, nil
}
