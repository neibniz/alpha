package membership

var planSnapshotColumns = []string{
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
}

var planTypeColumns = append(append([]string{}, planSnapshotColumns...),
	"enabled",
	"created_at",
	"updated_at",
)

var planDurationColumns = []string{
	"id",
	"duration_unit",
	"duration_value",
}

type PlanFieldsUpdate struct {
	Name                           string
	Description                    string
	UnitPrice                      int64
	DurationUnit                   string
	DurationValue                  int64
	Enabled                        bool
	MonthlyConsumptionLimitEnabled bool
	MonthlyConsumptionLimit        int64
	MonthlyOrderLimitEnabled       bool
	MonthlyOrderLimit              int64
	SubscriptionExpiryLimitEnabled bool
	SubscriptionExpiryGraceDays    int64
}

func (u PlanFieldsUpdate) Updates() map[string]any {
	return map[string]any{
		"name":                              u.Name,
		"description":                       u.Description,
		"unit_price":                        u.UnitPrice,
		"duration_unit":                     u.DurationUnit,
		"duration_value":                    u.DurationValue,
		"enabled":                           u.Enabled,
		"monthly_consumption_limit_enabled": u.MonthlyConsumptionLimitEnabled,
		"monthly_consumption_limit":         u.MonthlyConsumptionLimit,
		"monthly_order_limit_enabled":       u.MonthlyOrderLimitEnabled,
		"monthly_order_limit":               u.MonthlyOrderLimit,
		"subscription_expiry_limit_enabled": u.SubscriptionExpiryLimitEnabled,
		"subscription_expiry_grace_days":    u.SubscriptionExpiryGraceDays,
	}
}

func PlanSnapshotColumns() []string {
	return append([]string{}, planSnapshotColumns...)
}

func PlanTypeColumns() []string {
	return append([]string{}, planTypeColumns...)
}

func PlanDurationColumns() []string {
	return append([]string{}, planDurationColumns...)
}
