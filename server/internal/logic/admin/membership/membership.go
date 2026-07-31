package membership

import (
	"context"

	"github.com/neibniz/alpha/server/internal/membershipview"
	modelMembership "github.com/neibniz/alpha/server/internal/model/membership"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const defaultPlanSort = 1

func queryMembershipPlan(ctx context.Context, svcCtx *svc.ServiceContext) (*modelMembership.Plan, error) {
	var plan modelMembership.Plan
	err := svcCtx.DB.WithContext(ctx).
		Model(&modelMembership.Plan{}).
		Select(modelMembership.PlanTypeColumns()).
		Order("\"sort\" ASC, id ASC").
		First(&plan).Error
	if err == nil {
		return &plan, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query membership plan error: %v", err)
	}

	enabled := true
	plan = modelMembership.Plan{
		Name:                           "年费会员",
		Description:                    "开通后可购买订阅商品，默认有效期为一年。",
		UnitPrice:                      0,
		DurationUnit:                   "Year",
		DurationValue:                  1,
		Enabled:                        &enabled,
		MonthlyConsumptionLimitEnabled: false,
		MonthlyConsumptionLimit:        0,
		MonthlyOrderLimitEnabled:       false,
		MonthlyOrderLimit:              0,
		SubscriptionExpiryLimitEnabled: false,
		SubscriptionExpiryGraceDays:    0,
		Sort:                           defaultPlanSort,
	}
	if err = svcCtx.DB.WithContext(ctx).Create(&plan).Error; err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "create default membership plan error: %v", err)
	}
	return &plan, nil
}

func membershipPlanToType(plan *modelMembership.Plan) types.MembershipPlan {
	return membershipview.PlanFromModel(plan)
}

func validDurationUnit(unit string) bool {
	switch unit {
	case "Year", "Month", "Day", "Hour", "Minute":
		return true
	default:
		return false
	}
}
