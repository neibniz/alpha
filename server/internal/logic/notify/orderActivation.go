package notify

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/money"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/queue/types"
	"github.com/hibiken/asynq"
)

const (
	notifyOrderStatusPending  = 1
	notifyOrderStatusPaid     = 2
	notifyOrderStatusFinished = 5
)

func markOrderPaid(ctx context.Context, svcCtx *svc.ServiceContext, orderNo, tradeNo string) (bool, error) {
	paid, err := svcCtx.OrderModel.MarkPaid(ctx, orderNo, tradeNo)
	if err == nil && paid && svcCtx.DashboardRealtime != nil {
		svcCtx.DashboardRealtime.Publish(realtime.DashboardEventBusiness)
	}
	return paid, err
}

func parseYuanToCents(amount string) (int64, error) {
	value, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return 0, err
	}
	return money.MajorToMinor(value)
}

func expectedCNYAmountCents(svcCtx *svc.ServiceContext, info *order.Order) (int64, error) {
	if info == nil {
		return 0, strconv.ErrSyntax
	}
	if info.SettlementAmount > 0 {
		if info.SettlementCurrency != "CNY" {
			return 0, strconv.ErrSyntax
		}
		return info.SettlementAmount, nil
	}
	if info.SettlementCurrency != "" {
		return 0, strconv.ErrSyntax
	}
	currency, _ := svcCtx.CurrencyRuntime()
	if currency.Unit != "CNY" {
		return 0, strconv.ErrSyntax
	}
	return info.Amount, nil
}

func validateNotifyAmount(svcCtx *svc.ServiceContext, info *order.Order, paidCNYCents int64) bool {
	if paidCNYCents <= 0 {
		return false
	}
	expected, err := expectedCNYAmountCents(svcCtx, info)
	if err != nil {
		return false
	}
	return expected == paidCNYCents
}

func enqueueActivateOrder(ctx context.Context, svcCtx *svc.ServiceContext, orderNo string) (*asynq.TaskInfo, error) {
	payload := types.ForthwithActivateOrderPayload{OrderNo: orderNo}
	bytes, err := json.Marshal(&payload)
	if err != nil {
		return nil, err
	}
	task := asynq.NewTask(types.ForthwithActivateOrder, bytes)
	return svcCtx.Queue.EnqueueContext(ctx, task)
}
