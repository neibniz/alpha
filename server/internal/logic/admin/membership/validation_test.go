package membership

import (
	"strings"
	"testing"

	"github.com/neibniz/alpha/server/internal/types"
)

func validMembershipPlanRequest() *types.UpdateMembershipPlanRequest {
	enabled := true
	return &types.UpdateMembershipPlanRequest{
		Id:            1,
		Name:          " Annual Membership ",
		Description:   " Access subscription products ",
		UnitPrice:     12_000,
		DurationUnit:  "Year",
		DurationValue: 1,
		Enabled:       &enabled,
		Policy: types.MembershipPolicy{
			MonthlyConsumptionLimitEnabled: true,
			MonthlyConsumptionLimit:        100_000,
			MonthlyOrderLimitEnabled:       true,
			MonthlyOrderLimit:              20,
			SubscriptionExpiryLimitEnabled: true,
			SubscriptionExpiryGraceDays:    7,
		},
	}
}

func TestNormalizeMembershipPlan(t *testing.T) {
	req := validMembershipPlanRequest()
	if err := normalizeMembershipPlan(req); err != nil {
		t.Fatal(err)
	}
	if req.Name != "Annual Membership" || req.Description != "Access subscription products" {
		t.Fatalf("normalized request = %#v", req)
	}
}

func TestNormalizeMembershipPlanRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*types.UpdateMembershipPlanRequest)
	}{
		{name: "blank name", mutate: func(req *types.UpdateMembershipPlanRequest) { req.Name = " " }},
		{name: "long name", mutate: func(req *types.UpdateMembershipPlanRequest) { req.Name = strings.Repeat("会", 256) }},
		{name: "negative price", mutate: func(req *types.UpdateMembershipPlanRequest) { req.UnitPrice = -1 }},
		{name: "invalid duration", mutate: func(req *types.UpdateMembershipPlanRequest) { req.DurationValue = 1001 }},
		{name: "zero enabled consumption", mutate: func(req *types.UpdateMembershipPlanRequest) { req.Policy.MonthlyConsumptionLimit = 0 }},
		{name: "zero enabled order count", mutate: func(req *types.UpdateMembershipPlanRequest) { req.Policy.MonthlyOrderLimit = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := validMembershipPlanRequest()
			test.mutate(req)
			if err := normalizeMembershipPlan(req); err == nil {
				t.Fatal("normalizeMembershipPlan() error = nil")
			}
		})
	}
}
