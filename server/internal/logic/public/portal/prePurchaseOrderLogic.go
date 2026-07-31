package portal

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/couponpolicy"
	"github.com/neibniz/alpha/server/internal/productpolicy"
	"github.com/neibniz/alpha/server/internal/subscribeview"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/payment"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type PrePurchaseOrderLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Pre Purchase Order
func NewPrePurchaseOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PrePurchaseOrderLogic {
	return &PrePurchaseOrderLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PrePurchaseOrderLogic) PrePurchaseOrder(req *types.PrePurchaseOrderRequest) (resp *types.PrePurchaseOrderResponse, err error) {
	if req.Quantity <= 0 {
		req.Quantity = 1
	}
	// find subscribe plan
	sub, err := l.svcCtx.SubscribeModel.FindOneForPortalPrePurchase(l.ctx, req.SubscribeId)
	if err != nil {
		l.Errorw("[PreCreateOrder] Database query error", logger.Field("error", err.Error()), logger.Field("subscribe_id", req.SubscribeId))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find subscribe error: %v", err.Error())
	}
	if sub.Show == nil || !*sub.Show {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "subscribe not visible")
	}
	switch productpolicy.NewPurchaseAvailability(sub, time.Now()) {
	case productpolicy.AvailabilityAvailable:
	case productpolicy.AvailabilityOutOfStock:
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.SubscribeOutOfStock), "subscribe out of stock")
	case productpolicy.AvailabilityComingSoon:
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "product sale has not started")
	case productpolicy.AvailabilityEnded:
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "product sale has ended")
	default:
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "product is not available for purchase")
	}
	if err = productpolicy.ValidateQuantity(sub, req.Quantity, MaxQuantity); err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), err.Error())
	}
	var discount float64 = 1
	if sub.Discount != "" {
		discount = getDiscount(subscribeview.ParseDiscountLenient(sub.Discount), req.Quantity)
	}
	price := sub.UnitPrice * req.Quantity
	amount := int64(float64(price) * discount)
	discountAmount := price - amount
	if amount > MaxOrderAmount {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "order amount exceeds maximum limit")
	}
	var coupon int64
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

		coupon = calculateCoupon(amount, couponInfo)
	}
	amount -= coupon
	var feeAmount int64
	if req.Payment != 0 {
		paymentConfig, err := l.svcCtx.PaymentModel.FindOneForPublicOrder(l.ctx, req.Payment)
		if err != nil {
			l.Logger.Error("[PreCreateOrder] Database query error", logger.Field("error", err.Error()), logger.Field("payment", req.Payment))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find payment method error: %v", err.Error())
		}
		if err = ensurePaymentEnabled(paymentConfig); err != nil {
			return nil, err
		}
		if payment.ParsePlatform(paymentConfig.Platform) == payment.Balance {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.PaymentMethodNotFound), "balance error")
		}
		// Calculate the handling fee
		if amount > 0 {
			feeAmount = calculateFee(amount, paymentConfig)
		}
		amount += feeAmount
		if amount > MaxOrderAmount {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "order amount exceeds maximum limit")
		}
	}

	resp = &types.PrePurchaseOrderResponse{
		Price:          price,
		Amount:         amount,
		Discount:       discountAmount,
		Coupon:         req.Coupon,
		CouponDiscount: coupon,
		FeeAmount:      feeAmount,
	}
	return
}
