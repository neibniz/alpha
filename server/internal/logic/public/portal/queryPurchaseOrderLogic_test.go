package portal

import (
	"testing"
	"time"

	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/types"
)

func TestPurchaseOrderResponseFromModelMapsProjectedFields(t *testing.T) {
	createdAt := time.UnixMilli(1700000000123)
	subscribeInfo := types.OrderSubscribeProduct{Id: 7, Name: "Pro"}
	paymentInfo := types.PaymentMethod{Id: 9, Name: "Stripe"}
	orderInfo := &modelOrder.Order{
		OrderNo:        "alpha-order-1",
		Quantity:       2,
		Price:          1000,
		Amount:         1800,
		Discount:       100,
		Coupon:         "ALPHA",
		CouponDiscount: 100,
		FeeAmount:      30,
		Status:         1,
		CreatedAt:      createdAt,
	}

	got := purchaseOrderResponseFromModel(orderInfo, subscribeInfo, paymentInfo)
	if got.OrderNo != orderInfo.OrderNo || got.Quantity != orderInfo.Quantity || got.Price != orderInfo.Price || got.Amount != orderInfo.Amount {
		t.Fatalf("order amount fields not mapped: %#v", got)
	}
	if got.Discount != orderInfo.Discount || got.Coupon != orderInfo.Coupon || got.CouponDiscount != orderInfo.CouponDiscount || got.FeeAmount != orderInfo.FeeAmount {
		t.Fatalf("discount/fee fields not mapped: %#v", got)
	}
	if got.Status != orderInfo.Status || got.CreatedAt != createdAt.UnixMilli() {
		t.Fatalf("status/time fields not mapped: %#v", got)
	}
	if got.Subscribe.Id != subscribeInfo.Id || got.Payment.Id != paymentInfo.Id {
		t.Fatalf("related payloads not preserved: %#v", got)
	}
}

func TestPurchaseOrderResponseFromModelNormalizesZeroTime(t *testing.T) {
	got := purchaseOrderResponseFromModel(&modelOrder.Order{OrderNo: "alpha-order-zero"}, types.OrderSubscribeProduct{}, types.PaymentMethod{})
	if got.CreatedAt != 0 {
		t.Fatalf("zero timestamp should be encoded as 0: %#v", got)
	}
}
