package membership

import "testing"

func TestPlanColumnSetsStayFocused(t *testing.T) {
	assertHasColumns(t, "snapshot", PlanSnapshotColumns(), []string{
		"id",
		"name",
		"description",
		"unit_price",
		"duration_unit",
		"duration_value",
		"monthly_consumption_limit_enabled",
		"monthly_consumption_limit",
		"monthly_order_limit_enabled",
		"monthly_order_limit",
		"subscription_expiry_limit_enabled",
		"subscription_expiry_grace_days",
	})
	assertNoColumns(t, "snapshot", PlanSnapshotColumns(), []string{"enabled", "sort", "created_at", "updated_at"})

	assertHasColumns(t, "type", PlanTypeColumns(), []string{
		"id",
		"name",
		"description",
		"unit_price",
		"duration_unit",
		"duration_value",
		"enabled",
		"created_at",
		"updated_at",
	})
	assertNoColumns(t, "type", PlanTypeColumns(), []string{"sort"})

	assertHasColumns(t, "duration", PlanDurationColumns(), []string{"id", "duration_unit", "duration_value"})
	assertNoColumns(t, "duration", PlanDurationColumns(), []string{
		"name",
		"description",
		"unit_price",
		"enabled",
		"sort",
		"created_at",
		"updated_at",
	})
}

func TestPlanColumnSetCopiesAreImmutable(t *testing.T) {
	columns := PlanSnapshotColumns()
	columns[0] = "mutated"
	if PlanSnapshotColumns()[0] != "id" {
		t.Fatal("PlanSnapshotColumns must return a copy")
	}
}

func TestPlanFieldsUpdateIncludesFalseAndZeroValues(t *testing.T) {
	updates := PlanFieldsUpdate{
		Name:                           "member",
		Description:                    "",
		UnitPrice:                      0,
		DurationUnit:                   "Month",
		DurationValue:                  1,
		Enabled:                        false,
		MonthlyConsumptionLimitEnabled: false,
		MonthlyConsumptionLimit:        0,
		MonthlyOrderLimitEnabled:       false,
		MonthlyOrderLimit:              0,
		SubscriptionExpiryLimitEnabled: false,
		SubscriptionExpiryGraceDays:    0,
	}.Updates()

	for _, key := range []string{
		"description",
		"unit_price",
		"enabled",
		"monthly_consumption_limit_enabled",
		"monthly_consumption_limit",
		"monthly_order_limit_enabled",
		"monthly_order_limit",
		"subscription_expiry_limit_enabled",
		"subscription_expiry_grace_days",
	} {
		if _, ok := updates[key]; !ok {
			t.Fatalf("updates must include zero-value field %q", key)
		}
	}
	if updates["enabled"] != false || updates["unit_price"] != int64(0) {
		t.Fatalf("zero values were not preserved: %#v", updates)
	}
}

func assertHasColumns(t *testing.T, name string, columns []string, required []string) {
	t.Helper()
	set := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		set[column] = struct{}{}
	}
	for _, column := range required {
		if _, ok := set[column]; !ok {
			t.Fatalf("%s columns must include %q", name, column)
		}
	}
}

func assertNoColumns(t *testing.T, name string, columns []string, forbidden []string) {
	t.Helper()
	set := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		set[column] = struct{}{}
	}
	for _, column := range forbidden {
		if _, ok := set[column]; ok {
			t.Fatalf("%s columns must not include %q", name, column)
		}
	}
}
