package order

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/membershipview"
	"github.com/neibniz/alpha/server/internal/model/membership"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	OrderTypeSubscribe    uint8 = 1
	OrderTypeRenewal      uint8 = 2
	OrderTypeResetTraffic uint8 = 3
	OrderTypeRecharge     uint8 = 4
	OrderTypeMembership   uint8 = 5
)

func isActiveMember(u *user.User) bool {
	return u != nil && !u.MemberExpiredAt.IsZero() && u.MemberExpiredAt.After(time.Now())
}

func membershipFeatureEnabled(ctx context.Context, svcCtx *svc.ServiceContext) (bool, error) {
	var plan membership.Plan
	err := svcCtx.DB.WithContext(ctx).
		Model(&membership.Plan{}).
		Select("id").
		Where("enabled = ?", true).
		Order("\"sort\" ASC, id ASC").
		First(&plan).Error
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query membership feature state error: %v", err)
}

func requireActiveMember(ctx context.Context, svcCtx *svc.ServiceContext, u *user.User) error {
	enabled, err := membershipFeatureEnabled(ctx, svcCtx)
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	if isActiveMember(u) {
		return nil
	}
	return errors.Wrapf(xerr.NewErrCode(xerr.MembershipRequired), "membership required")
}

func queryDefaultMembershipPlan(ctx context.Context, svcCtx *svc.ServiceContext) (*membership.Plan, error) {
	var plan membership.Plan
	err := svcCtx.DB.WithContext(ctx).
		Model(&membership.Plan{}).
		Select(membership.PlanTypeColumns()).
		Where("enabled = ?", true).
		Order("\"sort\" ASC, id ASC").
		First(&plan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "membership plan not available")
		}
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query membership plan error: %v", err)
	}
	return &plan, nil
}

func membershipPlanToType(plan *membership.Plan) types.MembershipPlan {
	return membershipview.PlanFromModel(plan)
}
