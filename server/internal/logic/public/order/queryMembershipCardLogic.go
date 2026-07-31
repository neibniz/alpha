package order

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/timex"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QueryMembershipCardLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryMembershipCardLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryMembershipCardLogic {
	return &QueryMembershipCardLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryMembershipCardLogic) QueryMembershipCard() (resp *types.MembershipCardResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}

	plan, err := queryConfiguredMembershipPlan(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return &types.MembershipCardResponse{}, nil
	}
	enabled := plan.Enabled != nil && *plan.Enabled
	usage := membershipUsageTotals{}
	usagePolicyEnabled := plan.MonthlyConsumptionLimitEnabled || plan.MonthlyOrderLimitEnabled
	if enabled && usagePolicyEnabled {
		usage, err = queryMembershipUsageTotals(l.ctx, l.svcCtx, u.Id, time.Now())
		if err != nil {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query membership usage error: %v", err)
		}
	}
	return &types.MembershipCardResponse{
		Enabled:   enabled,
		IsMember:  enabled && isActiveMember(u),
		ExpiredAt: timex.UnixMilliOrZero(u.MemberExpiredAt),
		Plan:      membershipPlanToType(plan),
		Usage: types.MembershipUsage{
			MonthlyConsumption: usage.MonthlyConsumption,
			MonthlyOrderCount:  usage.MonthlyOrderCount,
		},
	}, nil
}
