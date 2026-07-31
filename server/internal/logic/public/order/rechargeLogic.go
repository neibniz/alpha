package order

import (
	"context"

	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/xerr"

	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/orderaudit"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/pkg/errors"
)

type RechargeLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Recharge
func NewRechargeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RechargeLogic {
	return &RechargeLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RechargeLogic) Recharge(req *types.RechargeOrderRequest) (resp *types.RechargeOrderResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	requestID, err := normalizeOrderRequestID(req.RequestId)
	if err != nil {
		return nil, err
	}
	if orderNo, found, lookupErr := findIdempotentOrderNo(l.ctx, l.svcCtx, u.Id, OrderTypeRecharge, requestID); lookupErr != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find idempotent order: %v", lookupErr)
	} else if found {
		return &types.RechargeOrderResponse{OrderNo: orderNo}, nil
	}

	// Validate recharge amount
	if req.Amount <= 0 {
		l.Errorw("[Recharge] Invalid recharge amount", logger.Field("amount", req.Amount), logger.Field("user_id", u.Id))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "recharge amount must be greater than 0")
	}

	if req.Amount > MaxRechargeAmount {
		l.Errorw("[Recharge] Recharge amount exceeds maximum limit",
			logger.Field("amount", req.Amount),
			logger.Field("max", MaxRechargeAmount),
			logger.Field("user_id", u.Id))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "recharge amount exceeds maximum limit")
	}

	// find payment method
	payment, err := l.svcCtx.PaymentModel.FindOneForPublicOrder(l.ctx, req.Payment)
	if err != nil {
		l.Errorw("[Recharge] Database query error", logger.Field("error", err.Error()), logger.Field("payment", req.Payment))
		return nil, errors.Wrapf(err, "find payment error: %v", err.Error())
	}
	if err = ensurePaymentEnabled(payment); err != nil {
		return nil, err
	}
	// Calculate the handling fee
	feeAmount := calculateFee(req.Amount, payment)
	totalAmount := req.Amount + feeAmount

	// Validate total amount after adding fee
	if totalAmount > MaxOrderAmount {
		l.Errorw("[Recharge] Total amount exceeds maximum limit after fee",
			logger.Field("amount", totalAmount),
			logger.Field("max", MaxOrderAmount),
			logger.Field("user_id", u.Id))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "total amount exceeds maximum limit")
	}

	// query user is new purchase or renewal
	isNew, err := l.svcCtx.OrderModel.IsUserEligibleForNewOrder(l.ctx, u.Id)
	if err != nil {
		l.Errorw("[Recharge] Database query error", logger.Field("error", err.Error()), logger.Field("user_id", u.Id))
		return nil, errors.Wrapf(err, "query user error: %v", err.Error())
	}
	orderNo, err := tool.GenerateTradeNo()
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "generate order number")
	}
	productSnapshot := ordersnapshot.FromRecharge(req.Amount)
	orderInfo := order.Order{
		UserId:          u.Id,
		OrderNo:         orderNo,
		RequestId:       requestID,
		Type:            OrderTypeRecharge,
		Price:           req.Amount,
		Amount:          totalAmount,
		FeeAmount:       feeAmount,
		PaymentId:       payment.Id,
		Method:          payment.Platform,
		Status:          1,
		IsNew:           isNew,
		ProductName:     productSnapshot.Name,
		ProductSnapshot: ordersnapshot.Marshal(productSnapshot),
	}
	orderaudit.FillNetwork(l.ctx, l.svcCtx, &orderInfo)
	if err = enqueueDeferredCloseOrder(l.ctx, l.svcCtx.Queue, l.Logger, orderInfo.OrderNo); err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "enqueue close order task error: %v", err.Error())
	}
	err = l.svcCtx.OrderModel.Insert(l.ctx, &orderInfo)
	if err != nil {
		if existingOrderNo, found, lookupErr := findIdempotentOrderNo(l.ctx, l.svcCtx, u.Id, OrderTypeRecharge, requestID); lookupErr == nil && found {
			return &types.RechargeOrderResponse{OrderNo: existingOrderNo}, nil
		}
		l.Errorw("[Recharge] Database insert error", logger.Field("error", err.Error()), logger.Field("order", orderInfo))
		return nil, errors.Wrapf(err, "insert order error: %v", err.Error())
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventActivity)
	}
	return &types.RechargeOrderResponse{
		OrderNo: orderInfo.OrderNo,
	}, nil
}
