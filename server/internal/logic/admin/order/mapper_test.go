package order

import (
	"testing"
	"time"

	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	modelPayment "github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestOrderListItemFromDetailsMapsAdminListShape(t *testing.T) {
	createdAt := time.UnixMilli(1700000000123)
	updatedAt := time.UnixMilli(1700000100456)
	item := &modelOrder.Details{
		Id:             7,
		UserId:         11,
		OrderNo:        "alpha-order-1",
		Type:           1,
		Quantity:       2,
		Price:          1000,
		Amount:         1800,
		GiftAmount:     100,
		Discount:       200,
		Coupon:         "ALPHA",
		CouponDiscount: 50,
		Commission:     30,
		Payment: &modelPayment.Payment{
			Id:          3,
			Name:        "Stripe",
			Platform:    "stripe",
			Description: "card",
			Icon:        "icon",
			FeeMode:     1,
			FeePercent:  5,
			FeeAmount:   10,
		},
		FeeAmount:   15,
		TradeNo:     "trade-1",
		Status:      2,
		SubscribeId: 19,
		ProductName: "Pro",
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	got := orderListItemFromDetails(item)
	if got.Id != item.Id || got.UserId != item.UserId || got.OrderNo != item.OrderNo || got.Type != item.Type {
		t.Fatalf("basic order fields not mapped: %#v", got)
	}
	if got.Amount != item.Amount || got.GiftAmount != item.GiftAmount || got.Discount != item.Discount || got.CouponDiscount != item.CouponDiscount {
		t.Fatalf("amount fields not mapped: %#v", got)
	}
	if got.Payment.Id != 3 || got.Payment.Name != "Stripe" || got.Payment.Platform != "stripe" {
		t.Fatalf("payment not mapped: %#v", got.Payment)
	}
	if got.UpdatedAt != updatedAt.UnixMilli() {
		t.Fatalf("updated_at = %d, want %d", got.UpdatedAt, updatedAt.UnixMilli())
	}
	if createdAt.UnixMilli() == got.UpdatedAt {
		t.Fatalf("created_at should not be mapped into admin order list item: %#v", got)
	}
}

func TestOrderListItemFromDetailsHandlesNilPayment(t *testing.T) {
	got := orderListItemFromDetails(&modelOrder.Details{Id: 8})
	if got.Id != 8 {
		t.Fatalf("id not mapped: %#v", got)
	}
	if got.Payment.Id != 0 || got.Payment.Platform != "" {
		t.Fatalf("nil payment should map to zero value: %#v", got.Payment)
	}
	if got.UpdatedAt != 0 {
		t.Fatalf("zero timestamps should stay zero: %#v", got)
	}
}

func TestOrderDetailFromDetailsMapsAdminOnlyFields(t *testing.T) {
	createdAt := time.UnixMilli(1700000000123)
	updatedAt := time.UnixMilli(1700000100456)
	item := &modelOrder.Details{
		Id:              17,
		UserId:          19,
		OrderNo:         "alpha-order-detail",
		Type:            2,
		Quantity:        3,
		Price:           3600,
		Amount:          3200,
		GiftAmount:      100,
		Discount:        200,
		Coupon:          "DETAIL",
		CouponDiscount:  150,
		Commission:      80,
		Method:          "stripe",
		FeeAmount:       30,
		TradeNo:         "trade-detail",
		Status:          2,
		SubscribeId:     11,
		ProductSnapshot: `{"version":1,"kind":"subscribe","name":"Snapshot Pro"}`,
		Payment: &modelPayment.Payment{
			Id:       3,
			Name:     "Stripe",
			Platform: "stripe",
		},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	got := orderDetailFromDetails(item)
	if got.Id != item.Id || got.UserId != item.UserId || got.OrderNo != item.OrderNo {
		t.Fatalf("basic detail fields not mapped: %#v", got)
	}
	if got.Coupon != "DETAIL" || got.Commission != 80 || got.Method != "stripe" || got.TradeNo != "trade-detail" {
		t.Fatalf("admin detail fields not mapped: %#v", got)
	}
	if got.ProductName != "Snapshot Pro" {
		t.Fatalf("snapshot product name = %q", got.ProductName)
	}
	if got.Payment.Id != 3 || got.Payment.Platform != "stripe" {
		t.Fatalf("payment not mapped: %#v", got.Payment)
	}
	if got.CreatedAt != createdAt.UnixMilli() || got.UpdatedAt != updatedAt.UnixMilli() {
		t.Fatalf("timestamps not mapped: %#v", got)
	}
}

func TestOrderListRowsFromDetailsSkipsNilAndKeepsProductName(t *testing.T) {
	rows := []*modelOrder.Details{
		{
			Id:          9,
			Type:        orderTypeMembership,
			SubscribeId: 18,
			ProductName: "Member Plan",
		},
		nil,
	}

	got := orderListRowsFromDetails(rows)
	if len(got) != 1 {
		t.Fatalf("admin order rows = %d, want 1: %#v", len(got), got)
	}
	if got[0].ProductName != "Member Plan" || got[0].SubscribeId != 18 {
		t.Fatalf("admin order list row mismatch: %#v", got[0])
	}
}

func TestOrderRowsToTypesSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*modelOrder.Details, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &modelOrder.Details{Id: 1, ProductName: "first"}
	rows[len(rows)-1] = &modelOrder.Details{Id: 2, ProductName: "second"}

	got := orderListRowsFromDetails(rows)
	if len(got) != 2 {
		t.Fatalf("admin order rows = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy admin order rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("admin order row order changed: %#v", got)
	}
}

func TestOrderRowsToTypesUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*modelOrder.Details, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &modelOrder.Details{Id: int64(index + 1), ProductName: "order"}
	}

	got := orderListRowsFromDetails(rows)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("admin order rows = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("admin order rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("admin order row order changed: %#v", got)
	}
}

func TestOrderRowsToTypesKeepsEmptyListShape(t *testing.T) {
	got := orderListRowsFromDetails(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("empty admin order rows should return non-nil empty slice: %#v", got)
	}
}
