package membershipview

import (
	modelMembership "github.com/neibniz/alpha/server/internal/model/membership"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/timex"
)

func PlanFromModel(plan *modelMembership.Plan) types.MembershipPlan {
	if plan == nil {
		return types.MembershipPlan{}
	}
	return types.MembershipPlan{
		Id:            plan.Id,
		Name:          plan.Name,
		Description:   plan.Description,
		UnitPrice:     plan.UnitPrice,
		DurationUnit:  plan.DurationUnit,
		DurationValue: plan.DurationValue,
		Enabled:       plan.Enabled != nil && *plan.Enabled,
		Policy: types.MembershipPolicy{
			MonthlyConsumptionLimitEnabled: plan.MonthlyConsumptionLimitEnabled,
			MonthlyConsumptionLimit:        plan.MonthlyConsumptionLimit,
			MonthlyOrderLimitEnabled:       plan.MonthlyOrderLimitEnabled,
			MonthlyOrderLimit:              plan.MonthlyOrderLimit,
			SubscriptionExpiryLimitEnabled: plan.SubscriptionExpiryLimitEnabled,
			SubscriptionExpiryGraceDays:    plan.SubscriptionExpiryGraceDays,
		},
		CreatedAt: timex.UnixMilliOrZero(plan.CreatedAt),
		UpdatedAt: timex.UnixMilliOrZero(plan.UpdatedAt),
	}
}
