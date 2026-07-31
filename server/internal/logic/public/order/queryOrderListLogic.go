package order

import (
	"context"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QueryOrderListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get order list
func NewQueryOrderListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryOrderListLogic {
	return &QueryOrderListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryOrderListLogic) QueryOrderList(req *types.QueryOrderListRequest) (resp *types.QueryOrderListResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	total, data, err := l.svcCtx.OrderModel.QueryOrderListByPage(l.ctx, req.Page, req.Size, req.Status, u.Id, 0, req.Search)
	if err != nil {
		l.Errorw("[QueryOrderListLogic] Query order list failed", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Query order list failed")
	}
	list := publicOrderRowsFromDetails(data)
	if err = hydratePublicOrderMembershipPlans(l.ctx, l.svcCtx, list); err != nil {
		l.Errorw("[QueryOrderListLogic] Query membership plans failed", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Query membership plans failed")
	}
	resp = &types.QueryOrderListResponse{
		Total: total,
		List:  list,
	}

	return
}
