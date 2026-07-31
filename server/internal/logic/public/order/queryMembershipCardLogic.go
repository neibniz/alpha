package order

import (
	"context"

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

	plan, err := queryDefaultMembershipPlan(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	return &types.MembershipCardResponse{
		IsMember:  isActiveMember(u),
		ExpiredAt: timex.UnixMilliOrZero(u.MemberExpiredAt),
		Plan:      membershipPlanToType(plan),
	}, nil
}
