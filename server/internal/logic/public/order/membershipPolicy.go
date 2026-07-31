package order

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/membership"
	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var membershipPolicyOrderTypes = []int64{1, 2, 3}
var membershipPolicyOrderStatuses = []int64{1, 2, 5}
var membershipPolicySelectColumns = []string{
	"id",
	"monthly_consumption_limit_enabled",
	"monthly_consumption_limit",
	"monthly_order_limit_enabled",
	"monthly_order_limit",
	"subscription_expiry_limit_enabled",
	"subscription_expiry_grace_days",
}

func enforceMembershipOrderPolicies(ctx context.Context, svcCtx *svc.ServiceContext, u *user.User, nextAmount int64) error {
	if u == nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	if nextAmount < 0 {
		nextAmount = 0
	}

	plan, err := queryMembershipPolicyPlan(ctx, svcCtx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query membership plan policy error: %v", err)
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	if plan.MonthlyConsumptionLimitEnabled && plan.MonthlyConsumptionLimit > 0 {
		var monthlyTotal int64
		if err = svcCtx.DB.WithContext(ctx).
			Model(&modelOrder.Order{}).
			Where("user_id = ?", u.Id).
			Where("type IN ?", membershipPolicyOrderTypes).
			Where("status IN ?", membershipPolicyOrderStatuses).
			Where("created_at >= ?", monthStart).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&monthlyTotal).Error; err != nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query membership monthly consumption error: %v", err)
		}
		if monthlyTotal+nextAmount > plan.MonthlyConsumptionLimit {
			return errors.Wrapf(xerr.NewErrCodeMsg(xerr.MembershipPolicyLimit, "monthly consumption limit exceeded"), "monthly consumption limit exceeded")
		}
	}

	if plan.MonthlyOrderLimitEnabled && plan.MonthlyOrderLimit > 0 {
		var monthlyOrders int64
		if err = svcCtx.DB.WithContext(ctx).
			Model(&modelOrder.Order{}).
			Where("user_id = ?", u.Id).
			Where("type IN ?", membershipPolicyOrderTypes).
			Where("status IN ?", membershipPolicyOrderStatuses).
			Where("created_at >= ?", monthStart).
			Count(&monthlyOrders).Error; err != nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query membership monthly order count error: %v", err)
		}
		if monthlyOrders+1 > plan.MonthlyOrderLimit {
			return errors.Wrapf(xerr.NewErrCodeMsg(xerr.MembershipPolicyLimit, "monthly order limit exceeded"), "monthly order limit exceeded")
		}
	}

	return nil
}

func EnforceMembershipSubscriptionExpiryPolicy(ctx context.Context, svcCtx *svc.ServiceContext, u *user.User, sub *subscribe.Subscribe, quantity int64, currentExpireAt *time.Time) error {
	if u == nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	if sub == nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "subscribe snapshot not found")
	}
	if quantity <= 0 {
		quantity = 1
	}

	plan, err := queryMembershipPolicyPlan(ctx, svcCtx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query membership plan policy error: %v", err)
	}
	if !plan.SubscriptionExpiryLimitEnabled {
		return nil
	}
	if !isActiveMember(u) {
		return errors.Wrapf(xerr.NewErrCode(xerr.MembershipRequired), "membership required")
	}
	if sub.UnitTime == "NoLimit" {
		return errors.Wrapf(xerr.NewErrCodeMsg(xerr.MembershipPolicyLimit, "subscription expiry exceeds membership limit"), "subscription expiry exceeds membership limit")
	}

	now := time.Now()
	base := now
	if currentExpireAt != nil && currentExpireAt.After(now) {
		base = *currentExpireAt
	}
	expireAt := CalculateSubscriptionExpireTime(sub.UnitTime, quantity, base)
	graceDays := plan.SubscriptionExpiryGraceDays
	if graceDays < 0 {
		graceDays = 0
	}
	allowedExpireAt := u.MemberExpiredAt.AddDate(0, 0, int(graceDays))
	if expireAt.After(allowedExpireAt) {
		return errors.Wrapf(xerr.NewErrCodeMsg(xerr.MembershipPolicyLimit, "subscription expiry exceeds membership limit"), "subscription expiry exceeds membership limit")
	}
	return nil
}

func queryMembershipPolicyPlan(ctx context.Context, svcCtx *svc.ServiceContext) (*membership.Plan, error) {
	var plan membership.Plan
	err := svcCtx.DB.WithContext(ctx).
		Model(&membership.Plan{}).
		Select(membershipPolicySelectColumns).
		Where("enabled = ?", true).
		Order("\"sort\" ASC, id ASC").
		First(&plan).Error
	return &plan, err
}

func CalculateSubscriptionExpireTime(unit string, quantity int64, base time.Time) time.Time {
	if quantity <= 0 {
		quantity = 1
	}
	if base.IsZero() {
		base = time.Now()
	}
	return tool.AddTime(unit, quantity, base)
}
