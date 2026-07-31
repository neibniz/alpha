package order

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/membership"
	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/orderaudit"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type CreateOrderLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Create order
func NewCreateOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOrderLogic {
	return &CreateOrderLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateOrderLogic) CreateOrder(req *types.CreateOrderRequest) error {
	paymentMethod, err := l.svcCtx.PaymentModel.FindOneForOrderRecord(l.ctx, req.PaymentId)
	if err != nil {
		l.Logger.Error("[CreateOrder] PaymentMethod Not Found", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.PaymentMethodNotFound), "PaymentMethod not found: %v", err.Error())
	}
	productName, productSnapshot := l.buildProductSnapshot(req)
	orderNo, err := tool.GenerateTradeNo()
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "generate order number")
	}

	orderInfo := &order.Order{
		UserId:          req.UserId,
		OrderNo:         orderNo,
		Type:            req.Type,
		Quantity:        req.Quantity,
		Price:           req.Price,
		Amount:          req.Amount,
		Discount:        req.Discount,
		Coupon:          req.Coupon,
		CouponDiscount:  req.CouponDiscount,
		PaymentId:       req.PaymentId,
		Method:          paymentMethod.Token,
		FeeAmount:       req.FeeAmount,
		TradeNo:         req.TradeNo,
		Status:          req.Status,
		SubscribeId:     req.SubscribeId,
		ProductName:     productName,
		ProductSnapshot: productSnapshot,
	}
	orderaudit.FillNetwork(l.ctx, l.svcCtx, orderInfo)

	err = l.svcCtx.OrderModel.Insert(l.ctx, orderInfo)
	if err != nil {
		l.Logger.Error("[CreateOrder] Database Error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "Insert error: %v", err.Error())
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventActivity)
	}
	return nil
}

func (l *CreateOrderLogic) buildProductSnapshot(req *types.CreateOrderRequest) (string, string) {
	if req.SubscribeId <= 0 {
		return "", ""
	}
	switch req.Type {
	case 1, 2, 3:
		sub, err := l.svcCtx.SubscribeModel.FindOneForOrderSnapshot(l.ctx, req.SubscribeId)
		if err != nil {
			l.Logger.Error("[CreateOrder] Subscribe not found", logger.Field("error", err.Error()), logger.Field("subscribe_id", req.SubscribeId))
			return "", ""
		}
		snapshot := ordersnapshot.FromSubscribe(sub)
		return snapshot.Name, ordersnapshot.Marshal(snapshot)
	case 5:
		var plan membership.Plan
		if err := l.svcCtx.DB.WithContext(l.ctx).
			Model(&membership.Plan{}).
			Select(membership.PlanSnapshotColumns()).
			Where("id = ?", req.SubscribeId).
			First(&plan).Error; err != nil {
			l.Logger.Error("[CreateOrder] Membership plan not found", logger.Field("error", err.Error()), logger.Field("plan_id", req.SubscribeId))
			return "", ""
		}
		snapshot := ordersnapshot.FromMembershipPlan(&plan)
		return snapshot.Name, ordersnapshot.Marshal(snapshot)
	default:
		return "", ""
	}
}
