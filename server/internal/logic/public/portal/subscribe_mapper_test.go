package portal

import (
	"testing"

	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
)

func TestSubscribeRowsFromModelsMapsPortalPublicShape(t *testing.T) {
	trueValue := true
	falseValue := false
	item := &modelSubscribe.Subscribe{
		Id:                17,
		Name:              "Portal Plan",
		Language:          "en",
		Description:       "portal",
		UnitPrice:         1200,
		UnitTime:          "month",
		Discount:          `[{"quantity":12,"discount":80}]`,
		Replacement:       1,
		Inventory:         100,
		Traffic:           2048,
		SpeedLimit:        300,
		DeviceLimit:       5,
		Quota:             6,
		Nodes:             "1,2",
		NodeTags:          "private",
		Show:              &trueValue,
		UserVisible:       &falseValue,
		Sell:              &trueValue,
		Sort:              9,
		DeductionRatio:    50,
		AllowDeduction:    &falseValue,
		AllowRenewal:      &trueValue,
		AllowResetTraffic: &falseValue,
		ResetCycle:        2,
		RenewalReset:      &trueValue,
		ShowOriginalPrice: true,
	}

	rows := subscribeRowsFromModels([]*modelSubscribe.Subscribe{item, nil})
	if len(rows) != 1 {
		t.Fatalf("portal public rows = %d, want 1: %#v", len(rows), rows)
	}
	got := rows[0]
	if got.Id != item.Id || got.Name != item.Name || got.UnitPrice != item.UnitPrice || got.Traffic != item.Traffic {
		t.Fatalf("portal subscribe fields not mapped: %#v", got)
	}
	if len(got.Discount) != 1 || got.Discount[0].Quantity != 12 || got.Discount[0].Discount != 80 {
		t.Fatalf("discount not parsed: %#v", got.Discount)
	}
	if got.UserVisible || !got.Sell || !got.ShowOriginalPrice {
		t.Fatalf("public booleans not mapped: %#v", got)
	}
}
