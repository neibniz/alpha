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

func orderListItemFromDetails(item *modelOrder.Details) types.OrderListItem {
	if item == nil {
		return types.OrderListItem{}
	}
	return types.OrderListItem{
		Id:                  item.Id,
		UserId:              item.UserId,
		OrderNo:             item.OrderNo,
		Source:              item.Source,
		Type:                item.Type,
		Quantity:            item.Quantity,
		Price:               item.Price,
		Amount:              item.Amount,
		GiftAmount:          item.GiftAmount,
		CommissionDeduction: item.CommissionDeduction,
		Discount:            item.Discount,
		CouponDiscount:      item.CouponDiscount,
		Payment:             paymentSummaryFromModel(item.Payment),
		FeeAmount:           item.FeeAmount,
		TradeNo:             item.TradeNo,
		Status:              item.Status,
		SubscribeId:         item.SubscribeId,
		ProductName:         item.ProductName,
		UpdatedAt:           timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func orderDetailFromDetails(item *modelOrder.Details) types.OrderDetail {
	if item == nil {
		return types.OrderDetail{}
	}
	productName := item.ProductName
	if productName == "" {
		productName = ordersnapshot.ProductName(item.ProductSnapshot)
	}
	return types.OrderDetail{
		Id:                  item.Id,
		UserId:              item.UserId,
		OrderNo:             item.OrderNo,
		Source:              item.Source,
		Type:                item.Type,
		Quantity:            item.Quantity,
		Price:               item.Price,
		Amount:              item.Amount,
		GiftAmount:          item.GiftAmount,
		CommissionDeduction: item.CommissionDeduction,
		Discount:            item.Discount,
		Coupon:              item.Coupon,
		CouponDiscount:      item.CouponDiscount,
		Commission:          item.Commission,
		Payment:             paymentview.MethodFromModel(item.Payment),
		Method:              item.Method,
		FeeAmount:           item.FeeAmount,
		TradeNo:             item.TradeNo,
		Status:              item.Status,
		SubscribeId:         item.SubscribeId,
		Subscribe:           orderview.SubscribeProductFromModel(item.Subscribe),
		ProductName:         productName,
		CreatedAt:           timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:           timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func orderListRowsFromDetails(rows []*modelOrder.Details) []types.OrderListItem {
	if len(rows) == 0 {
		return []types.OrderListItem{}
	}
	collector := viewutil.NewRowsCollector[types.OrderListItem](len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(orderListItemFromDetails(row))
	}
	return collector.Rows()
}

func paymentSummaryFromModel(item *modelPayment.Payment) types.PaymentMethodSummary {
	if item == nil {
		return types.PaymentMethodSummary{}
	}
	return types.PaymentMethodSummary{
		Id:       item.Id,
		Name:     item.Name,
		Platform: item.Platform,
	}
}
