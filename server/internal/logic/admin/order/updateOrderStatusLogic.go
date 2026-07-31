package order

import (
	"context"
	"encoding/json"

	orderModel "github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	queue "github.com/neibniz/alpha/server/queue/types"
)

type UpdateOrderStatusLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Update order status
func NewUpdateOrderStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateOrderStatusLogic {
	return &UpdateOrderStatusLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateOrderStatusLogic) UpdateOrderStatus(req *types.UpdateOrderStatusRequest) error {
	info, err := l.svcCtx.OrderModel.FindOneForStatusUpdate(l.ctx, req.Id)
	if err != nil {
		l.Errorw("[UpdateOrderStatus] FindOneForStatusUpdate error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "FindOneForStatusUpdate error: %v", err.Error())
	}
	if !allowedAdminOrderStatusTransition(info.Status, req.Status) {
		return errors.Wrapf(
			xerr.NewErrCode(xerr.OrderStatusError),
			"order status transition %d -> %d is not allowed",
			info.Status,
			req.Status,
		)
	}

	update := orderModel.OrderFieldsUpdate{
		Status: orderModel.UpdateValue(req.Status),
	}
	if req.PaymentId != 0 {
		paymentMethod, err := l.svcCtx.PaymentModel.FindOneForOrderRecord(l.ctx, req.PaymentId)
		if err != nil {
			l.Logger.Error("[CreateOrder] PaymentMethod Not Found", logger.Field("error", err.Error()))
			return errors.Wrapf(xerr.NewErrCode(xerr.PaymentMethodNotFound), "PaymentMethod not found: %v", err.Error())
		}
		update.PaymentID = orderModel.UpdateValue(paymentMethod.Id)
		update.Method = orderModel.UpdateValue(paymentMethod.Platform)
	}
	if req.TradeNo != "" {
		update.TradeNo = orderModel.UpdateValue(req.TradeNo)
	}
	err = l.svcCtx.OrderModel.Transaction(l.ctx, func(db *gorm.DB) error {
		if err := l.svcCtx.OrderModel.UpdateAdminStatusFields(l.ctx, info, update, db); err != nil {
			l.Errorw("[UpdateOrderStatus] Update error", logger.Field("error", err.Error()), logger.Field("OrderID", info.Id))
			return err
		}
		// If order status is 2, create user subscription
		if req.Status == 2 {
			payload := queue.ForthwithActivateOrderPayload{
				OrderNo: info.OrderNo,
			}
			p, _ := json.Marshal(payload)
			task := asynq.NewTask(queue.ForthwithActivateOrder, p)
			_, err = l.svcCtx.Queue.EnqueueContext(l.ctx, task)
			if err != nil {
				l.Errorw("[UpdateOrderStatus] Enqueue error", logger.Field("error", err.Error()))
				return errors.Wrapf(xerr.NewErrCode(xerr.QueueEnqueueError), "Enqueue error: %v", err.Error())
			}
		}
		return nil
	})
	if err != nil {
		l.Errorw("[UpdateOrderStatus] Transaction error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Transaction error: %v", err.Error())
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventBusiness)
	}
	return nil
}
