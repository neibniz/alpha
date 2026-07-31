package membership

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type QueryMembershipPlanLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryMembershipPlanLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryMembershipPlanLogic {
	return &QueryMembershipPlanLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryMembershipPlanLogic) QueryMembershipPlan() (resp *types.MembershipPlan, err error) {
	plan, err := queryMembershipPlan(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	resp = &types.MembershipPlan{}
	*resp = membershipPlanToType(plan)
	return resp, nil
}
