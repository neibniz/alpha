package order

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/couponpolicy"
	"github.com/neibniz/alpha/server/internal/orderaudit"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/productpolicy"
	"github.com/neibniz/alpha/server/pkg/constant"

	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/subscribeview"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm/clause"
)

type RenewalLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewRenewalLogic creates a new renewal logic instance for subscription renewal operations
func NewRenewalLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RenewalLogic {
	return &RenewalLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Renewal processes subscription renewal orders including discount calculation,
// coupon validation, gift amount deduction, fee calculation, and order creation
func (l *RenewalLogic) Renewal(req *types.RenewalOrderRequest) (resp *types.RenewalOrderResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	requestID, err := normalizeOrderRequestID(req.RequestId)
	if err != nil {
		return nil, err
	}
	if orderNo, found, lookupErr := findIdempotentOrderNo(l.ctx, l.svcCtx, u.Id, OrderTypeRenewal, requestID); lookupErr != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find idempotent order: %v", lookupErr)
	} else if found {
		return &types.RenewalOrderResponse{OrderNo: orderNo}, nil
	}
	if err = requireActiveMember(l.ctx, l.svcCtx, u); err != nil {
		return nil, err
	}

	orderNo, err := tool.GenerateTradeNo()
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "generate order number")
	}
	// find user subscribe
	userSubscribe, err := l.svcCtx.UserModel.FindOneUserSubscribeForOrderAction(l.ctx, req.UserSubscribeID)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find user subscribe error: %v", err.Error())
	}
	if userSubscribe.UserId != u.Id {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "subscription does not belong to current user")
	}
	// Renewals always use the current product record. Historical order and user
	// subscription snapshots remain immutable evidence of earlier purchases.
	sub, err := l.svcCtx.SubscribeModel.FindOneForPublicPurchase(l.ctx, userSubscribe.SubscribeId)
	if err != nil {
		l.Errorw("[Renewal] current product not found", logger.Field("user_subscribe_id", userSubscribe.Id), logger.Field("subscribe_id", userSubscribe.SubscribeId))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "current product not found: %v", err)
	}
	if err = ensureRenewalActionAllowed(userSubscribe.Status, userSubscribe.ExpireTime, sub, time.Now()); err != nil {
		return nil, err
	}
	if req.Quantity <= 0 {
		req.Quantity = productpolicy.NormalizeMinQuantity(sub.MinQuantity)
	}
	if err = ensureProductQuantity(sub, req.Quantity); err != nil {
		return nil, err
	}
	if err = EnforceMembershipSubscriptionExpiryPolicy(l.ctx, l.svcCtx, u, sub, req.Quantity, &userSubscribe.ExpireTime); err != nil {
		return nil, err
	}
	var discount float64 = 1
	if sub.Discount != "" {
		discount = getDiscount(subscribeview.ParseDiscountLenient(sub.Discount), req.Quantity)
	}
	price := sub.UnitPrice * req.Quantity
	amount := int64(float64(price) * discount)
	discountAmount := price - amount

	// Validate amount to prevent overflow
	if amount > MaxOrderAmount {
		l.Errorw("[Renewal] Order amount exceeds maximum limit",
			logger.Field("amount", amount),
			logger.Field("max", MaxOrderAmount),
			logger.Field("user_id", u.Id),
			logger.Field("subscribe_id", sub.Id))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "order amount exceeds maximum limit")
	}

	var coupon int64 = 0
	var couponUserLimit int64
	if req.Coupon != "" {
		couponInfo, err := l.svcCtx.CouponModel.FindOneByCode(l.ctx, req.Coupon)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.Wrapf(xerr.NewErrCode(xerr.CouponNotExist), "coupon not found")
			}
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find coupon error: %v", err.Error())
		}
		if err = couponpolicy.Validate(couponInfo, sub.Id, time.Now()); err != nil {
			return nil, err
		}
		limitReached, err := couponUsageReached(l.ctx, l.svcCtx.DB, u.Id, req.Coupon, couponInfo.UserLimit)
		if err != nil {
			l.Errorw("[Renewal] Database query error", logger.Field("error", err.Error()), logger.Field("user_id", u.Id), logger.Field("coupon", req.Coupon))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find coupon error: %v", err.Error())
		}
		if limitReached {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.CouponInsufficientUsage), "coupon limit exceeded")
		}
		couponUserLimit = couponInfo.UserLimit
		coupon = calculateCoupon(amount, couponInfo)
	}
	amount -= coupon
	walletDeduction := quoteOrderWalletDeduction(amount, u)
	amount = walletDeduction.Provider

	var feeAmount int64
	paymentID := int64(0)
	method := "free"
	paymentMethod, err := resolveOrderPayment(l.ctx, l.svcCtx, req.Payment, amount)
	if err != nil {
		return nil, err
	}
	// Calculate the handling fee
	if amount > 0 {
		paymentID = paymentMethod.Id
		method = paymentMethod.Platform
		feeAmount = calculateFee(amount, paymentMethod)
	}

	amount += feeAmount

	// Final validation after adding fee
	if amount > MaxOrderAmount {
		l.Errorw("[Renewal] Final order amount exceeds maximum limit after fee",
			logger.Field("amount", amount),
			logger.Field("max", MaxOrderAmount),
			logger.Field("user_id", u.Id))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "order amount exceeds maximum limit")
	}
	if err = enforceMembershipOrderPolicies(l.ctx, l.svcCtx, u, amount); err != nil {
		return nil, err
	}

	// create order
	productSnapshot := ordersnapshot.FromSubscribe(sub)
	orderInfo := order.Order{
		UserId:              u.Id,
		ParentId:            userSubscribe.OrderId,
		OrderNo:             orderNo,
		RequestId:           requestID,
		Type:                OrderTypeRenewal,
		Quantity:            req.Quantity,
		Price:               price,
		Amount:              amount,
		GiftAmount:          walletDeduction.Gift,
		CommissionDeduction: walletDeduction.Commission,
		Discount:            discountAmount,
		Coupon:              req.Coupon,
		CouponDiscount:      coupon,
		PaymentId:           paymentID,
		Method:              method,
		FeeAmount:           feeAmount,
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
		lockedUser := &user.User{}
		if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
			Model(&user.User{}).
			Select(orderWalletUserColumns).
			Where("id = ?", u.Id).
			First(lockedUser).Error; err != nil {
			return err
		}
		lockedProduct, err := lockCurrentSubscribeForOrder(l.ctx, db, sub.Id)
		if err != nil {
			return err
		}
		if err := ensureRenewalActionAllowed(lockedUserSubscribe.Status, lockedUserSubscribe.ExpireTime, lockedProduct, time.Now()); err != nil {
			return err
		}
		if err := ensureProductQuantity(lockedProduct, req.Quantity); err != nil {
			return err
		}
		if err := ensureProductSnapshotCurrent(lockedProduct, orderInfo.ProductSnapshot); err != nil {
			return err
		}
		if couponUserLimit > 0 {
			limitReached, err := couponUsageReached(l.ctx, db, u.Id, req.Coupon, couponUserLimit)
			if err != nil {
				return err
			}
			if limitReached {
				return errors.Wrapf(xerr.NewErrCode(xerr.CouponInsufficientUsage), "coupon limit exceeded")
			}
		}
		if err := applyOrderWalletDeduction(l.ctx, l.svcCtx, db, lockedUser, orderInfo.OrderNo, walletDeduction, "Renewal order deduction"); err != nil {
			l.Errorw("[Renewal] Wallet deduction failed", logger.Field("error", err.Error()), logger.Field("user_id", u.Id))
			return err
		}
		// insert order
		return db.Model(&order.Order{}).Create(&orderInfo).Error
	})
	if err != nil {
		if existingOrderNo, found, lookupErr := findIdempotentOrderNo(l.ctx, l.svcCtx, u.Id, OrderTypeRenewal, requestID); lookupErr == nil && found {
			return &types.RenewalOrderResponse{OrderNo: existingOrderNo}, nil
		}
		l.Errorw("[Renewal] Database insert error", logger.Field("error", err.Error()), logger.Field("order", orderInfo))
		return nil, errors.Wrapf(err, "insert order error: %v", err.Error())
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventActivity)
	}
	return &types.RenewalOrderResponse{
		OrderNo: orderInfo.OrderNo,
	}, nil
}
