package order

import (
	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	modelPayment "github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/orderview"
	"github.com/neibniz/alpha/server/internal/paymentview"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
)

func orderDetailFromDetails(item *modelOrder.Details) types.OrderDetail {
	if item == nil {
		return types.OrderDetail{}
	}
	return types.OrderDetail{
		Id:                  item.Id,
		UserId:              item.UserId,
		OrderNo:             item.OrderNo,
		Type:                item.Type,
		Quantity:            item.Quantity,
		Price:               item.Price,
		Amount:              item.Amount,
		GiftAmount:          item.GiftAmount,
		CommissionDeduction: item.CommissionDeduction,
		Discount:            item.Discount,
		Coupon:              item.Coupon,
		CouponDiscount:      item.CouponDiscount,
		Payment:             paymentMethodFromModel(item.Payment),
		Method:              item.Method,
		FeeAmount:           item.FeeAmount,
		TradeNo:             item.TradeNo,
		Status:              item.Status,
		SubscribeId:         item.SubscribeId,
		Subscribe:           orderview.SubscribeProductFromModel(item.Subscribe),
		ProductName:         item.ProductName,
		CreatedAt:           timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:           timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func orderDetailRowsToTypes(rows []*modelOrder.Details) []types.OrderDetail {
	if len(rows) == 0 {
		return []types.OrderDetail{}
	}
	collector := viewutil.NewRowsCollector[types.OrderDetail](len(rows))
	snapshotCache := ordersnapshot.NewParseCache(len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		orderInfo := orderDetailFromDetails(row)
		orderInfo.Commission = 0
		applyOrderSnapshotWithCache(&orderInfo, row, snapshotCache)
		collector.Add(orderInfo)
	}
	return collector.Rows()
}

func publicOrderRowsFromDetails(rows []*modelOrder.Details) []types.PublicOrder {
	if len(rows) == 0 {
		return []types.PublicOrder{}
	}
	collector := viewutil.NewRowsCollector[types.PublicOrder](len(rows))
	snapshotCache := ordersnapshot.NewParseCache(len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		orderInfo := publicOrderFromDetails(row)
		applyPublicOrderSnapshotWithCache(&orderInfo, row, snapshotCache)
		collector.Add(orderInfo)
	}
	return collector.Rows()
}

func publicOrderFromDetails(item *modelOrder.Details) types.PublicOrder {
	if item == nil {
		return types.PublicOrder{}
	}
	resp := types.PublicOrder{
		Id:                  item.Id,
		OrderNo:             item.OrderNo,
		Type:                item.Type,
		Quantity:            item.Quantity,
		Price:               item.Price,
		Amount:              item.Amount,
		GiftAmount:          item.GiftAmount,
		CommissionDeduction: item.CommissionDeduction,
		Discount:            item.Discount,
		CouponDiscount:      item.CouponDiscount,
		FeeAmount:           item.FeeAmount,
		Status:              item.Status,
		SubscribeId:         item.SubscribeId,
		ProductName:         item.ProductName,
		CreatedAt:           timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:           timex.UnixMilliOrZero(item.UpdatedAt),
	}
	return resp
}

func paymentMethodFromModel(item *modelPayment.Payment) types.PaymentMethod {
	return paymentview.MethodFromModel(item)
}
