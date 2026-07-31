package order

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/orderaudit"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PurchaseMembershipLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPurchaseMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PurchaseMembershipLogic {
	return &PurchaseMembershipLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PurchaseMembershipLogic) PurchaseMembership(req *types.PurchaseMembershipOrderRequest) (resp *types.PurchaseMembershipOrderResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	requestID, err := normalizeOrderRequestID(req.RequestId)
	if err != nil {
		return nil, err
	}
	if orderNo, found, lookupErr := findIdempotentOrderNo(l.ctx, l.svcCtx, u.Id, OrderTypeMembership, requestID); lookupErr != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find idempotent order: %v", lookupErr)
	} else if found {
		return &types.PurchaseMembershipOrderResponse{OrderNo: orderNo}, nil
	}

	plan, err := queryDefaultMembershipPlan(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	price := plan.UnitPrice
	amount := price
	if amount > MaxOrderAmount {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "order amount exceeds maximum limit")
	}
	walletDeduction := quoteOrderWalletDeduction(amount, u)
	amount = walletDeduction.Provider

	var paymentId int64
	method := "free"
	var feeAmount int64
	payment, err := resolveOrderPayment(l.ctx, l.svcCtx, req.Payment, amount)
	if err != nil {
		return nil, err
	}
	if amount > 0 {
		feeAmount = calculateFee(amount, payment)
		amount += feeAmount
		if amount > MaxOrderAmount {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "order amount exceeds maximum limit")
		}
		paymentId = payment.Id
		method = payment.Platform
	}

	orderNo, err := tool.GenerateTradeNo()
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "generate order number")
	}
	productSnapshot := ordersnapshot.FromMembershipPlan(plan)
	orderInfo := &order.Order{
		UserId:              u.Id,
		OrderNo:             orderNo,
		RequestId:           requestID,
		Type:                OrderTypeMembership,
		Quantity:            1,
		Price:               price,
		Amount:              amount,
		GiftAmount:          walletDeduction.Gift,
		CommissionDeduction: walletDeduction.Commission,
		PaymentId:           paymentId,
		Method:              method,
		FeeAmount:           feeAmount,
		Status:              1,
		SubscribeId:         plan.Id,
		ProductName:         productSnapshot.Name,
		ProductSnapshot:     ordersnapshot.Marshal(productSnapshot),
	}
	orderaudit.FillNetwork(l.ctx, l.svcCtx, orderInfo)
	if err = enqueueDeferredCloseOrder(l.ctx, l.svcCtx.Queue, l.Logger, orderInfo.OrderNo); err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "enqueue close order task error: %v", err.Error())
	}
	if err = l.svcCtx.DB.Transaction(func(db *gorm.DB) error {
		if walletDeduction.Commission > 0 || walletDeduction.Gift > 0 {
			lockedUser := &user.User{}
			if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
				Model(&user.User{}).
				Select(orderWalletUserColumns).
				Where("id = ?", u.Id).
				First(lockedUser).Error; err != nil {
				return err
			}
			if err := applyOrderWalletDeduction(l.ctx, l.svcCtx, db, lockedUser, orderInfo.OrderNo, walletDeduction, "Membership order deduction"); err != nil {
				return err
			}
		}
		return l.svcCtx.OrderModel.Insert(l.ctx, orderInfo, db)
	}); err != nil {
		if existingOrderNo, found, lookupErr := findIdempotentOrderNo(l.ctx, l.svcCtx, u.Id, OrderTypeMembership, requestID); lookupErr == nil && found {
			return &types.PurchaseMembershipOrderResponse{OrderNo: existingOrderNo}, nil
		}
		l.Errorw("[PurchaseMembership] Database insert error", logger.Field("error", err.Error()), logger.Field("orderInfo", orderInfo))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "insert order error: %v", err.Error())
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventActivity)
	}

	return &types.PurchaseMembershipOrderResponse{OrderNo: orderInfo.OrderNo}, nil
}
