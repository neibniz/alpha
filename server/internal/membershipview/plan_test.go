package membershipview

import (
	"testing"
	"time"

	modelMembership "github.com/neibniz/alpha/server/internal/model/membership"
)

func TestPlanFromModelMapsAPIShape(t *testing.T) {
	enabled := true
	createdAt := time.UnixMilli(1716192000000)
	updatedAt := time.UnixMilli(1716195600000)
	got := PlanFromModel(&modelMembership.Plan{
		Id:                             3,
		Name:                           "VIP",
		Description:                    "vip plan",
		UnitPrice:                      99,
		DurationUnit:                   "Month",
		DurationValue:                  1,
		Enabled:                        &enabled,
		CreatedAt:                      createdAt,
		UpdatedAt:                      updatedAt,
		MonthlyOrderLimitEnabled:       true,
		MonthlyOrderLimit:              10,
		SubscriptionExpiryLimitEnabled: true,
		SubscriptionExpiryGraceDays:    7,
	})
	if got.Id != 3 || got.Name != "VIP" || got.UnitPrice != 99 || !got.Enabled {
		t.Fatalf("plan fields mismatch: %+v", got)
	}
	if !got.Policy.MonthlyOrderLimitEnabled || got.Policy.MonthlyOrderLimit != 10 || got.Policy.SubscriptionExpiryGraceDays != 7 {
		t.Fatalf("policy fields mismatch: %+v", got.Policy)
	}
	if got.CreatedAt != createdAt.UnixMilli() || got.UpdatedAt != updatedAt.UnixMilli() {
		t.Fatalf("timestamps mismatch: %+v", got)
	}
}

func TestPlanFromModelKeepsZeroTimesAtZero(t *testing.T) {
	got := PlanFromModel(&modelMembership.Plan{})
	if got.CreatedAt != 0 || got.UpdatedAt != 0 {
		t.Fatalf("zero timestamps should stay zero: %+v", got)
	}
}
