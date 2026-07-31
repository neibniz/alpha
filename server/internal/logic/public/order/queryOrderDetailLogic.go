package order

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/membership"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type QueryOrderDetailLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get order
func NewQueryOrderDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryOrderDetailLogic {
	return &QueryOrderDetailLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryOrderDetailLogic) QueryOrderDetail(req *types.QueryOrderDetailRequest) (resp *types.OrderDetail, err error) {
	currentUser, ok := l.ctx.Value(constant.CtxKeyUser).(*modelUser.User)
	if !ok || currentUser == nil || currentUser.Id <= 0 {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "current user is not available")
	}
	orderInfo, err := l.svcCtx.OrderModel.FindOneDetailsByOrderNoForUser(l.ctx, req.OrderNo, currentUser.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "order is not accessible")
		}
		l.Errorw("[QueryOrderDetail] Database query error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find order error: %v", err.Error())
	}
	orderDetail := orderDetailFromDetails(orderInfo)
	resp = &orderDetail
	// Prevent commission amount leakage
	resp.Commission = 0
	applyOrderSnapshot(resp, orderInfo)
	if resp.Type == OrderTypeMembership && resp.SubscribeId > 0 {
		if resp.MembershipPlan != nil {
			return resp, nil
		}
		resp.Subscribe = types.OrderSubscribeProduct{}
		var plan membership.Plan
		if err = l.svcCtx.DB.WithContext(l.ctx).
			Model(&membership.Plan{}).
			Select(membership.PlanTypeColumns()).
			Where("id = ?", resp.SubscribeId).
			First(&plan).Error; err != nil {
			l.Errorw("[QueryOrderDetail] Membership plan query error", logger.Field("error", err.Error()), logger.Field("order_no", req.OrderNo), logger.Field("plan_id", resp.SubscribeId))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find membership plan error: %v", err.Error())
		}
		planResp := membershipPlanToType(&plan)
		resp.MembershipPlan = &planResp
	}
	return
}
