package portal

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/orderview"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/timex"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QueryPurchaseOrderLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewQueryPurchaseOrderLogic Query Purchase Order
func NewQueryPurchaseOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryPurchaseOrderLogic {
	return &QueryPurchaseOrderLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Centralized error handler for database issues
func wrapDatabaseError(err error) error {
	return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Database Query Error: %v", err.Error())
}

func (l *QueryPurchaseOrderLogic) QueryPurchaseOrder(req *types.QueryPurchaseOrderRequest) (resp *types.QueryPurchaseOrderResponse, err error) {
	orderInfo, err := l.svcCtx.OrderModel.FindOneForPurchaseQuery(l.ctx, req.OrderNo)
	if err != nil {
		return nil, wrapDatabaseError(err)
	}
	if _, ok := authenticatedOrderOwner(l.ctx, orderInfo); !ok {
		l.Errorw("[QueryPurchaseOrder] order owner mismatch",
			logger.Field("orderNo", req.OrderNo),
			logger.Field("orderUserId", orderInfo.UserId),
		)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "invalid order access")
	}
	// Fetch subscription and payment information
	subscribeInfo, paymentInfo, err := l.fetchOrderDetails(orderInfo)
	if err != nil {
		return nil, err
	}

	return purchaseOrderResponseFromModel(orderInfo, subscribeInfo, paymentInfo), nil
}

func purchaseOrderResponseFromModel(orderInfo *order.Order, subscribeInfo types.OrderSubscribeProduct, paymentInfo types.PaymentMethod) *types.QueryPurchaseOrderResponse {
	if orderInfo == nil {
		return &types.QueryPurchaseOrderResponse{
			Subscribe: subscribeInfo,
			Payment:   paymentInfo,
		}
	}
	return &types.QueryPurchaseOrderResponse{
		OrderNo:        orderInfo.OrderNo,
		Subscribe:      subscribeInfo,
		Quantity:       orderInfo.Quantity,
		Price:          orderInfo.Price,
		Amount:         orderInfo.Amount,
		Discount:       orderInfo.Discount,
		Coupon:         orderInfo.Coupon,
		CouponDiscount: orderInfo.CouponDiscount,
		FeeAmount:      orderInfo.FeeAmount,
		Payment:        paymentInfo,
		Status:         orderInfo.Status,
		CreatedAt:      timex.UnixMilliOrZero(orderInfo.CreatedAt),
	}
}

// fetchOrderDetails retrieves subscription and payment details
func (l *QueryPurchaseOrderLogic) fetchOrderDetails(orderInfo *order.Order) (types.OrderSubscribeProduct, types.PaymentMethod, error) {
	var subscribeInfo types.OrderSubscribeProduct
	if snapshot, ok := ordersnapshot.Parse(orderInfo.ProductSnapshot); ok && snapshot.Kind == ordersnapshot.KindSubscribe {
		subscribeInfo = orderview.SubscribeProductFromSnapshot(snapshot)
		paymentInfo, paymentErr := l.fetchPaymentInfo(orderInfo)
		return subscribeInfo, paymentInfo, paymentErr
	}

	sub, err := l.svcCtx.SubscribeModel.FindOneForOrderSnapshot(l.ctx, orderInfo.SubscribeId)
	if err != nil {
		return types.OrderSubscribeProduct{}, types.PaymentMethod{}, wrapDatabaseError(err)
	}

	subscribeInfo = orderview.SubscribeProductFromModel(sub)

	paymentInfo, err := l.fetchPaymentInfo(orderInfo)
	return subscribeInfo, paymentInfo, err
}

func (l *QueryPurchaseOrderLogic) fetchPaymentInfo(orderInfo *order.Order) (types.PaymentMethod, error) {
	payment, err := l.svcCtx.PaymentModel.FindOneForPublicOrder(l.ctx, orderInfo.PaymentId)
	if err != nil {
		return types.PaymentMethod{}, wrapDatabaseError(err)
	}

	return paymentMethodFromModel(payment), nil
}
