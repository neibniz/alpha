package orderview

import (
	"testing"

	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
)

func TestSubscribeProductFromModelKeepsPaymentPageShapeLean(t *testing.T) {
	trueValue := true
	item := &modelSubscribe.Subscribe{
		Id:                18,
		Name:              "Historical Plan",
		Language:          "en",
		Description:       "historical",
		UnitPrice:         1500,
		UnitTime:          "month",
		Discount:          `[{"quantity":6,"discount":90}]`,
		Replacement:       100,
		Inventory:         50,
		Traffic:           4096,
		SpeedLimit:        500,
		DeviceLimit:       8,
		Quota:             2,
		Nodes:             "10,11",
		NodeTags:          "edge,vip",
		Show:              &trueValue,
		UserVisible:       &trueValue,
		Sell:              &trueValue,
		Sort:              9,
		DeductionRatio:    50,
		AllowDeduction:    &trueValue,
		AllowRenewal:      &trueValue,
		AllowResetTraffic: &trueValue,
		ResetCycle:        2,
		RenewalReset:      &trueValue,
		ShowOriginalPrice: true,
	}

	got := SubscribeProductFromModel(item)
	if got.Id != item.Id || got.Name != item.Name || got.UnitPrice != item.UnitPrice || got.UnitTime != item.UnitTime {
		t.Fatalf("order subscribe product identity/price fields not mapped: %#v", got)
	}
	if got.Traffic != item.Traffic || got.SpeedLimit != item.SpeedLimit || got.DeviceLimit != item.DeviceLimit || !got.ShowOriginalPrice {
		t.Fatalf("order subscribe product resource fields not mapped: %#v", got)
	}
}

func TestSubscribeProductFromSnapshotOmitsTopologyAndPolicyFields(t *testing.T) {
	snapshot := ordersnapshot.Snapshot{
		Kind:              ordersnapshot.KindSubscribe,
		ID:                88,
		Name:              "Snapshot Plan",
		Description:       "snapshot description",
		UnitPrice:         1299,
		UnitTime:          "month",
		Replacement:       1999,
		Traffic:           4096,
		SpeedLimit:        50,
		DeviceLimit:       3,
		Quota:             2,
		Nodes:             []int64{7, 9},
		NodeTags:          []string{"edge", "home"},
		AllowRenewal:      true,
		AllowResetTraffic: true,
		ResetCycle:        2,
		RenewalReset:      true,
		ShowOriginalPrice: true,
	}

	got := SubscribeProductFromSnapshot(snapshot)
	if got.Id != 88 || got.Name != "Snapshot Plan" || got.UnitPrice != 1299 || got.UnitTime != "month" {
		t.Fatalf("snapshot order product identity/price fields not mapped: %#v", got)
	}
	if got.Traffic != 4096 || got.SpeedLimit != 50 || got.DeviceLimit != 3 || !got.ShowOriginalPrice {
		t.Fatalf("snapshot order product resource fields not mapped: %#v", got)
	}
}
