package order

import (
	"reflect"
	"testing"
	"time"

	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	modelPayment "github.com/neibniz/alpha/server/internal/model/payment"
	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestOrderDetailFromDetailsMapsPublicShapeAndAssociatedSubscribe(t *testing.T) {
	trueValue := true
	falseValue := false
	createdAt := time.UnixMilli(1700000000123)
	updatedAt := time.UnixMilli(1700000100456)
	item := &modelOrder.Details{
		Id:             21,
		UserId:         7,
		OrderNo:        "alpha-order-public",
		Type:           OrderTypeSubscribe,
		Quantity:       1,
		Price:          1000,
		Amount:         900,
		GiftAmount:     100,
		Discount:       100,
		Coupon:         "PUBLIC",
		CouponDiscount: 50,
		Commission:     999,
		Payment: &modelPayment.Payment{
			Id:          5,
			Name:        "Balance",
			Platform:    "balance",
			Description: "wallet",
			Icon:        "wallet",
			FeeMode:     2,
			FeePercent:  0,
			FeeAmount:   5,
		},
		Method:      "balance",
		FeeAmount:   5,
		TradeNo:     "trade-public",
		Status:      2,
		SubscribeId: 31,
		Subscribe: &modelSubscribe.Subscribe{
			Id:                31,
			Name:              "Legacy Plan",
			Language:          "en",
			Description:       "legacy",
			UnitPrice:         1000,
			UnitTime:          "month",
			Discount:          `[{"quantity":3,"discount":90}]`,
			Replacement:       1,
			Inventory:         10,
			Traffic:           1024,
			SpeedLimit:        200,
			DeviceLimit:       3,
			Quota:             4,
			Nodes:             "101,102",
			NodeTags:          "asia, premium ",
			Show:              &trueValue,
			UserVisible:       &trueValue,
			Sell:              &trueValue,
			Sort:              8,
			DeductionRatio:    70,
			AllowDeduction:    &falseValue,
			AllowRenewal:      &trueValue,
			AllowResetTraffic: &falseValue,
			ResetCycle:        2,
			RenewalReset:      &trueValue,
			ShowOriginalPrice: true,
			CreatedAt:         createdAt,
			UpdatedAt:         updatedAt,
		},
		ProductName: "Legacy Plan",
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	got := orderDetailFromDetails(item)
	if got.Id != item.Id || got.UserId != item.UserId || got.OrderNo != item.OrderNo || got.Method != item.Method {
		t.Fatalf("order fields not mapped: %#v", got)
	}
	if got.Commission != 0 {
		t.Fatalf("public mapper must not expose commission: %#v", got)
	}
	if got.Payment.Id != 5 || got.Payment.Platform != "balance" || got.Payment.FeeAmount != 5 {
		t.Fatalf("payment not mapped: %#v", got.Payment)
	}
	if got.Subscribe.Id != 31 || got.Subscribe.Name != "Legacy Plan" || got.Subscribe.UnitPrice != 1000 || got.Subscribe.UnitTime != "month" {
		t.Fatalf("subscribe fields not mapped: %#v", got.Subscribe)
	}
	if got.Subscribe.Traffic != 1024 || got.Subscribe.SpeedLimit != 200 || got.Subscribe.DeviceLimit != 3 || !got.Subscribe.ShowOriginalPrice {
		t.Fatalf("subscribe resource fields not mapped: %#v", got.Subscribe)
	}
	if got.CreatedAt != createdAt.UnixMilli() || got.UpdatedAt != updatedAt.UnixMilli() {
		t.Fatalf("timestamps = %d/%d, want %d/%d", got.CreatedAt, got.UpdatedAt, createdAt.UnixMilli(), updatedAt.UnixMilli())
	}
}

func TestOrderDetailFromDetailsHandlesNilRelations(t *testing.T) {
	got := orderDetailFromDetails(&modelOrder.Details{Id: 22})
	if got.Id != 22 {
		t.Fatalf("id not mapped: %#v", got)
	}
	if got.Payment.Id != 0 || got.Subscribe.Id != 0 {
		t.Fatalf("nil relations should map to zero values: payment=%#v subscribe=%#v", got.Payment, got.Subscribe)
	}
	if got.CreatedAt != 0 || got.UpdatedAt != 0 {
		t.Fatalf("zero timestamps should stay zero: %#v", got)
	}
}

func TestOrderDetailRowsToTypesKeepsPublicListShape(t *testing.T) {
	rows := []*modelOrder.Details{
		{
			Id:         31,
			UserId:     7,
			Type:       OrderTypeSubscribe,
			Commission: 999,
			ProductSnapshot: ordersnapshot.Marshal(ordersnapshot.Snapshot{
				Kind:    ordersnapshot.KindSubscribe,
				ID:      88,
				Name:    "Snapshot Plan",
				Traffic: 4096,
			}),
		},
		nil,
	}

	got := orderDetailRowsToTypes(rows)
	if len(got) != 1 {
		t.Fatalf("public order rows = %d, want 1: %#v", len(got), got)
	}
	if got[0].Commission != 0 {
		t.Fatalf("public order rows must clear commission: %#v", got[0])
	}
	if got[0].ProductName != "Snapshot Plan" || got[0].Subscribe.Id != 88 || got[0].Subscribe.Traffic != 4096 {
		t.Fatalf("public order snapshot not applied: %#v", got[0])
	}
}

func TestPublicOrderRowsFromDetailsKeepsLeanListShape(t *testing.T) {
	createdAt := time.UnixMilli(1700000000123)
	updatedAt := time.UnixMilli(1700000100456)
	rows := []*modelOrder.Details{
		{
			Id:             41,
			UserId:         7,
			OrderNo:        "alpha-order-list",
			Type:           OrderTypeSubscribe,
			Quantity:       2,
			Price:          1000,
			Amount:         1800,
			GiftAmount:     100,
			Discount:       200,
			Coupon:         "PUBLIC",
			CouponDiscount: 50,
			Commission:     999,
			Method:         "balance",
			FeeAmount:      5,
			TradeNo:        "trade-public",
			Status:         2,
			SubscribeId:    31,
			ProductSnapshot: ordersnapshot.Marshal(ordersnapshot.Snapshot{
				Kind:    ordersnapshot.KindSubscribe,
				ID:      88,
				Name:    "Snapshot Plan",
				Traffic: 4096,
			}),
			Subscribe: &modelSubscribe.Subscribe{
				Id:   31,
				Name: "Live Plan",
			},
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		nil,
	}

	got := publicOrderRowsFromDetails(rows)
	if len(got) != 1 {
		t.Fatalf("public order rows = %d, want 1: %#v", len(got), got)
	}
	if got[0].Id != 41 || got[0].OrderNo != "alpha-order-list" || got[0].Amount != 1800 || got[0].Status != 2 {
		t.Fatalf("public order row fields not mapped: %#v", got[0])
	}
	if got[0].ProductName != "Snapshot Plan" {
		t.Fatalf("public order row should use snapshot product name: %#v", got[0])
	}
	if got[0].CreatedAt != createdAt.UnixMilli() || got[0].UpdatedAt != updatedAt.UnixMilli() {
		t.Fatalf("public order row timestamps = %d/%d, want %d/%d", got[0].CreatedAt, got[0].UpdatedAt, createdAt.UnixMilli(), updatedAt.UnixMilli())
	}
}

func TestPublicOrderRowsFromDetailsFallsBackToHydratedSubscribeName(t *testing.T) {
	got := publicOrderRowsFromDetails([]*modelOrder.Details{
		{
			Id:   42,
			Type: OrderTypeRenewal,
			Subscribe: &modelSubscribe.Subscribe{
				Id:   31,
				Name: "Live Plan",
			},
		},
	})
	if len(got) != 1 || got[0].ProductName != "Live Plan" {
		t.Fatalf("public order row should fall back to hydrated subscribe name: %#v", got)
	}
}

func TestOrderDetailRowsToTypesSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*modelOrder.Details, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &modelOrder.Details{Id: 1, ProductName: "first"}
	rows[len(rows)-1] = &modelOrder.Details{Id: 2, ProductName: "second"}

	got := orderDetailRowsToTypes(rows)
	if len(got) != 2 {
		t.Fatalf("public order rows = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy public order rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("public order row order changed: %#v", got)
	}
}

func TestOrderDetailRowsToTypesUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*modelOrder.Details, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &modelOrder.Details{Id: int64(index + 1), ProductName: "order"}
	}

	got := orderDetailRowsToTypes(rows)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("public order rows = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("public order rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("public order row order changed: %#v", got)
	}
}

func TestOrderDetailRowsToTypesKeepsEmptyListShape(t *testing.T) {
	got := orderDetailRowsToTypes(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("empty public order rows should return non-nil empty slice: %#v", got)
	}
}

func TestOrderWalletUserColumnsStayLightweight(t *testing.T) {
	want := []string{"id", "commission", "gift_amount"}
	if !reflect.DeepEqual(orderWalletUserColumns, want) {
		t.Fatalf("order wallet user columns = %#v, want %#v", orderWalletUserColumns, want)
	}
}
