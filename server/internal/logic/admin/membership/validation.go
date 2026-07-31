package membership

import (
	"strings"
	"unicode/utf8"

	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/xerr"
)

const (
	maxMembershipAmount     int64 = 2_000_000_000
	maxMembershipOrderLimit int64 = 1_000_000
	maxMembershipGraceDays  int64 = 36_500
)

func normalizeMembershipPlan(req *types.UpdateMembershipPlanRequest) error {
	if req == nil || req.Id <= 0 {
		return invalidMembershipPlan("membership plan id is required")
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.DurationUnit = strings.TrimSpace(req.DurationUnit)
	if count := utf8.RuneCountInString(req.Name); count == 0 || count > 255 {
		return invalidMembershipPlan("membership name must contain 1 to 255 characters")
	}
	if utf8.RuneCountInString(req.Description) > 5_000 {
		return invalidMembershipPlan("membership description must not exceed 5000 characters")
	}
	if req.UnitPrice < 0 || req.UnitPrice > maxMembershipAmount {
		return invalidMembershipPlan("membership price is out of range")
	}
	if !validDurationUnit(req.DurationUnit) || req.DurationValue < 1 || req.DurationValue > 1_000 {
		return invalidMembershipPlan("membership duration is invalid")
	}
	if req.Enabled == nil {
		return invalidMembershipPlan("membership enabled state is required")
	}
	policy := req.Policy
	if policy.MonthlyConsumptionLimit < 0 || policy.MonthlyConsumptionLimit > maxMembershipAmount ||
		policy.MonthlyOrderLimit < 0 || policy.MonthlyOrderLimit > maxMembershipOrderLimit ||
		policy.SubscriptionExpiryGraceDays < 0 || policy.SubscriptionExpiryGraceDays > maxMembershipGraceDays {
		return invalidMembershipPlan("membership policy limit is out of range")
	}
	if policy.MonthlyConsumptionLimitEnabled && policy.MonthlyConsumptionLimit == 0 {
		return invalidMembershipPlan("monthly consumption limit must be greater than zero when enabled")
	}
	if policy.MonthlyOrderLimitEnabled && policy.MonthlyOrderLimit == 0 {
		return invalidMembershipPlan("monthly order limit must be greater than zero when enabled")
	}
	return nil
}

func invalidMembershipPlan(message string) error {
	return xerr.NewErrCodeMsg(xerr.InvalidParams, message)
}
