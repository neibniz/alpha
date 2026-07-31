package subscribe

import (
	"testing"

	subscribeModel "github.com/neibniz/alpha/server/internal/model/subscribe"
)

func TestPublicSubscribeFromModelMapsPlanFieldsWithoutTopology(t *testing.T) {
	trueValue := true
	falseValue := false
	item := &subscribeModel.Subscribe{
		Id:                21,
		Name:              "Starter",
		Language:          "en",
		Description:       "public plan",
		UnitPrice:         500,
		UnitTime:          "month",
		Discount:          `[{"quantity":6,"discount":85}]`,
		Replacement:       1,
		Inventory:         50,
		Traffic:           256,
		SpeedLimit:        100,
		DeviceLimit:       2,
		Quota:             3,
		Nodes:             "1,2",
		NodeTags:          "private",
		Show:              &trueValue,
		UserVisible:       &trueValue,
		Sell:              &trueValue,
		Sort:              4,
		DeductionRatio:    70,
		AllowDeduction:    &falseValue,
		AllowRenewal:      &trueValue,
		AllowResetTraffic: &falseValue,
		ResetCycle:        1,
		RenewalReset:      &trueValue,
		ShowOriginalPrice: true,
	}

	got := publicSubscribeFromModel(item)
	if got.Id != item.Id || got.Name != item.Name || got.Description != item.Description || got.UnitPrice != item.UnitPrice || got.UnitTime != item.UnitTime {
		t.Fatalf("basic plan fields not mapped: %#v", got)
	}
	if got.Replacement != item.Replacement || got.Traffic != item.Traffic || got.SpeedLimit != item.SpeedLimit || got.DeviceLimit != item.DeviceLimit {
		t.Fatalf("resource/reset fields not mapped: %#v", got)
	}
	if len(got.Discount) != 1 || got.Discount[0].Quantity != 6 || got.Discount[0].Discount != 85 {
		t.Fatalf("discount not parsed: %#v", got.Discount)
	}
	if !got.UserVisible || !got.Sell || !got.ShowOriginalPrice {
		t.Fatalf("public booleans not mapped: %#v", got)
	}
}

func TestPublicSubscribeFromModelIgnoresInvalidDiscount(t *testing.T) {
	got := publicSubscribeFromModel(&subscribeModel.Subscribe{Id: 22, Name: "Plan", Discount: "not-json"})
	if got.Id != 22 || got.Name != "Plan" {
		t.Fatalf("fields should be preserved with invalid discount: %#v", got)
	}
	if got.Discount == nil || len(got.Discount) != 0 {
		t.Fatalf("invalid discount should be an empty array: %#v", got.Discount)
	}
}

func TestPublicSubscribeFromModelKeepsEmptyDiscountArray(t *testing.T) {
	got := publicSubscribeFromModel(&subscribeModel.Subscribe{})
	if got.Discount == nil || len(got.Discount) != 0 {
		t.Fatalf("empty discount should stay a non-nil empty array: %#v", got.Discount)
	}
}
