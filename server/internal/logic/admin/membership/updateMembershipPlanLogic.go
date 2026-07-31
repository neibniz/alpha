package membership

import (
	"context"

	modelMembership "github.com/neibniz/alpha/server/internal/model/membership"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type UpdateMembershipPlanLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateMembershipPlanLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateMembershipPlanLogic {
	return &UpdateMembershipPlanLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateMembershipPlanLogic) UpdateMembershipPlan(req *types.UpdateMembershipPlanRequest) (resp *types.MembershipPlan, err error) {
	if err = normalizeMembershipPlan(req); err != nil {
		return nil, errors.Wrap(err, "validate membership plan")
	}

	var plan modelMembership.Plan
	if err = l.svcCtx.DB.WithContext(l.ctx).
		Model(&modelMembership.Plan{}).
		Select("id").
		Where("id = ?", req.Id).
		First(&plan).Error; err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find membership plan error: %v", err)
	}

	update := modelMembership.PlanFieldsUpdate{
		Name:                           req.Name,
		Description:                    req.Description,
		UnitPrice:                      req.UnitPrice,
		DurationUnit:                   req.DurationUnit,
		DurationValue:                  req.DurationValue,
		Enabled:                        *req.Enabled,
		MonthlyConsumptionLimitEnabled: req.Policy.MonthlyConsumptionLimitEnabled,
		MonthlyConsumptionLimit:        req.Policy.MonthlyConsumptionLimit,
		MonthlyOrderLimitEnabled:       req.Policy.MonthlyOrderLimitEnabled,
		MonthlyOrderLimit:              req.Policy.MonthlyOrderLimit,
		SubscriptionExpiryLimitEnabled: req.Policy.SubscriptionExpiryLimitEnabled,
		SubscriptionExpiryGraceDays:    req.Policy.SubscriptionExpiryGraceDays,
	}
	if err = l.svcCtx.DB.WithContext(l.ctx).Model(&plan).Updates(update.Updates()).Error; err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update membership plan error: %v", err)
	}

	if err = l.svcCtx.DB.WithContext(l.ctx).
		Model(&modelMembership.Plan{}).
		Select(modelMembership.PlanTypeColumns()).
		Where("id = ?", req.Id).
		First(&plan).Error; err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "reload membership plan error: %v", err)
	}

	resp = &types.MembershipPlan{}
	*resp = membershipPlanToType(&plan)
	return resp, nil
}
