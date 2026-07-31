package order

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/couponpolicy"
	"github.com/neibniz/alpha/server/internal/orderaudit"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/subscribeview"
	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type PurchaseLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

const (
	CloseOrderTimeMinutes = 15
)

// NewPurchaseLogic creates a new purchase logic instance for subscription purchase operations.
// It initializes the logger with context and sets up the service context for database operations.
func NewPurchaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PurchaseLogic {
	return &PurchaseLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Purchase processes new subscription purchase orders including validation, discount calculation,
// coupon processing, gift amount deduction, fee calculation, and order creation with database transaction.
// It handles the complete purchase workflow from user validation to order creation and task scheduling.
func (l *PurchaseLogic) Purchase(req *types.PurchaseOrderRequest) (resp *types.PurchaseOrderResponse, err error) {

	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	requestID, err := normalizeOrderRequestID(req.RequestId)
	if err != nil {
		return nil, err
	}
	if orderNo, found, lookupErr := findIdempotentOrderNo(l.ctx, l.svcCtx, u.Id, OrderTypeSubscribe, requestID); lookupErr != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find idempotent order: %v", lookupErr)
	} else if found {
		return &types.PurchaseOrderResponse{OrderNo: orderNo}, nil
	}

	if req.Quantity <= 0 {
		l.Debugf("[Purchase] Quantity is less than or equal to 0, setting to 1")
		req.Quantity = 1
	}

	if err = requireActiveMember(u); err != nil {
		return nil, err
	}

	// find subscribe plan
	sub, err := l.svcCtx.SubscribeModel.FindOneForPublicPurchase(l.ctx, req.SubscribeId)

	if err != nil {
		l.Errorw("[Purchase] Database query error", logger.Field("error", err.Error()), logger.Field("subscribe_id", req.SubscribeId))
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
			l.Errorw("[Purchase] Database query error", logger.Field("error", err.Error()), logger.Field("user_id", u.Id))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "count user subscription error: %v", err.Error())
		}
		if count > 0 {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.UserSubscribeExist), "user has subscription")
		}
	}

	// check subscribe plan limit
	if sub.Quota > 0 {
		count, err := l.svcCtx.UserModel.CountUserSubscribeByProduct(l.ctx, u.Id, req.SubscribeId)
		if err != nil {
			l.Errorw("[Purchase] Database query error", logger.Field("error", err.Error()), logger.Field("user_id", u.Id), logger.Field("subscribe_id", req.SubscribeId))
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
	// discount amount
	amount := int64(float64(price) * discount)
	discountAmount := price - amount

	// Validate amount to prevent overflow
	if amount > MaxOrderAmount {
		l.Errorw("[Purchase] Order amount exceeds maximum limit",
			logger.Field("amount", amount),
			logger.Field("max", MaxOrderAmount),
			logger.Field("user_id", u.Id),
			logger.Field("subscribe_id", req.SubscribeId))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "order amount exceeds maximum limit")
	}

	var coupon int64 = 0
	var couponUserLimit int64
	// Calculate the coupon deduction
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
			l.Errorw("[Purchase] Database query error", logger.Field("error", err.Error()), logger.Field("user_id", u.Id), logger.Field("coupon", req.Coupon))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find coupon error: %v", err.Error())
		}
		if limitReached {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.CouponInsufficientUsage), "coupon limit exceeded")
		}
		couponUserLimit = couponInfo.UserLimit
		coupon = calculateCoupon(amount, couponInfo)
	}
	// Calculate the handling fee
	amount -= coupon
	walletDeduction := quoteOrderWalletDeduction(amount, u)
	amount = walletDeduction.Provider
	paymentMethod, err := resolveOrderPayment(l.ctx, l.svcCtx, req.Payment, amount)
	if err != nil {
		return nil, err
	}
	paymentID := int64(0)
	method := "free"
	var feeAmount int64
	// Calculate the handling fee
	if amount > 0 {
		paymentID = paymentMethod.Id
		method = paymentMethod.Platform
		feeAmount = calculateFee(amount, paymentMethod)
		amount += feeAmount

		// Final validation after adding fee
		if amount > MaxOrderAmount {
			l.Errorw("[Purchase] Final order amount exceeds maximum limit after fee",
				logger.Field("amount", amount),
				logger.Field("max", MaxOrderAmount),
				logger.Field("user_id", u.Id))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "order amount exceeds maximum limit")
		}
	}
	if err = enforceMembershipOrderPolicies(l.ctx, l.svcCtx, u, amount); err != nil {
		return nil, err
	}
	// query user is new purchase or renewal
	isNew, err := l.svcCtx.OrderModel.IsUserEligibleForNewOrder(l.ctx, u.Id)
	if err != nil {
		l.Errorw("[Purchase] Database query error", logger.Field("error", err.Error()), logger.Field("user_id", u.Id))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find user order error: %v", err.Error())
	}
	// create order
	orderNo, err := tool.GenerateTradeNo()
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "generate order number")
	}
	productSnapshot := ordersnapshot.FromSubscribe(sub)
	orderInfo := &order.Order{
		UserId:              u.Id,
		OrderNo:             orderNo,
		RequestId:           requestID,
		Type:                OrderTypeSubscribe,
		Quantity:            req.Quantity,
		Price:               price,
		Amount:              amount,
		Discount:            discountAmount,
		GiftAmount:          walletDeduction.Gift,
		CommissionDeduction: walletDeduction.Commission,
		Coupon:              req.Coupon,
		CouponDiscount:      coupon,
		PaymentId:           paymentID,
		Method:              method,
		FeeAmount:           feeAmount,
		Status:              1,
		IsNew:               isNew,
		SubscribeId:         req.SubscribeId,
		ProductName:         productSnapshot.Name,
		ProductSnapshot:     ordersnapshot.Marshal(productSnapshot),
	}
	orderaudit.FillNetwork(l.ctx, l.svcCtx, orderInfo)
	if err = enqueueDeferredCloseOrder(l.ctx, l.svcCtx.Queue, l.Logger, orderInfo.OrderNo); err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "enqueue close order task error: %v", err.Error())
	}
	// Database transaction
	err = l.svcCtx.DB.Transaction(func(db *gorm.DB) error {
		lockedUser := &user.User{}
		if e := db.Clauses(clause.Locking{Strength: "UPDATE"}).
			Model(&user.User{}).
			Select(orderWalletUserColumns).
			Where("id = ?", u.Id).
			First(lockedUser).Error; e != nil {
			l.Errorw("[Purchase] Database query error", logger.Field("error", e.Error()), logger.Field("user", u))
			return e
		}
		lockedProduct, e := lockCurrentSubscribeForOrder(l.ctx, db, sub.Id)
		if e != nil {
			return e
		}
		if lockedProduct.UserVisible == nil || !*lockedProduct.UserVisible {
			return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "subscribe not visible")
		}
		if e = ensureNewPurchaseAvailable(lockedProduct, time.Now()); e != nil {
			return e
		}
		if e = ensureProductQuantity(lockedProduct, req.Quantity); e != nil {
			return e
		}
		if e = ensureProductSnapshotCurrent(lockedProduct, orderInfo.ProductSnapshot); e != nil {
			return e
		}
		if couponUserLimit > 0 {
			limitReached, e := couponUsageReached(l.ctx, db, u.Id, req.Coupon, couponUserLimit)
			if e != nil {
				return e
			}
			if limitReached {
				return errors.Wrapf(xerr.NewErrCode(xerr.CouponInsufficientUsage), "coupon limit exceeded")
			}
		}
		if l.svcCtx.ConfigSnapshot().Subscribe.SingleModel {
			var activeCount int64
			if e := db.Model(&user.Subscribe{}).Where("user_id = ? AND status = ?", u.Id, 1).Count(&activeCount).Error; e != nil {
				return e
			}
			if activeCount > 0 {
				return errors.Wrapf(xerr.NewErrCode(xerr.UserSubscribeExist), "user has subscription")
			}
		}
		if lockedProduct.Quota > 0 {
			var quotaCount int64
			if e := db.Model(&user.Subscribe{}).Where("user_id = ? AND subscribe_id = ? AND status = ?", u.Id, req.SubscribeId, 1).Count(&quotaCount).Error; e != nil {
				return e
			}
			if quotaCount >= lockedProduct.Quota {
				return errors.Wrapf(xerr.NewErrCode(xerr.SubscribeQuotaLimit), "quota limit")
			}
		}
		if e := applyOrderWalletDeduction(l.ctx, l.svcCtx, db, lockedUser, orderInfo.OrderNo, walletDeduction, "Purchase order deduction"); e != nil {
			l.Errorw("[Purchase] Wallet deduction failed", logger.Field("error", e.Error()), logger.Field("user_id", u.Id))
			return e
		}

		if lockedProduct.Inventory != -1 {
			if err = decreaseSubscribeInventory(l.ctx, db, lockedProduct.Id); err != nil {
				l.Errorw("[Purchase] Decrease subscribe inventory failed", logger.Field("error", err.Error()), logger.Field("subscribe_id", sub.Id))
				return err
			}
		}

		// insert order
		return db.WithContext(l.ctx).Model(&order.Order{}).Create(&orderInfo).Error
	})
	if err != nil {
		if existingOrderNo, found, lookupErr := findIdempotentOrderNo(l.ctx, l.svcCtx, u.Id, OrderTypeSubscribe, requestID); lookupErr == nil && found {
			return &types.PurchaseOrderResponse{OrderNo: existingOrderNo}, nil
		}
		l.Errorw("[Purchase] Database insert error", logger.Field("error", err.Error()), logger.Field("orderInfo", orderInfo))

		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "insert order error: %v", err.Error())
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventActivity)
	}

	return &types.PurchaseOrderResponse{
		OrderNo: orderInfo.OrderNo,
	}, nil
}
