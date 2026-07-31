package ticket

import (
	"context"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/ticketview"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetUserTicketDetailsLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get ticket detail
func NewGetUserTicketDetailsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserTicketDetailsLogic {
	return &GetUserTicketDetailsLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserTicketDetailsLogic) GetUserTicketDetails(req *types.GetUserTicketDetailRequest) (resp *types.Ticket, err error) {

	data, err := l.svcCtx.TicketModel.QueryTicketDetail(l.ctx, req.Id)
	if err != nil {
		l.Errorw("[GetUserTicketDetailsLogic] Database Error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get ticket detail failed: %v", err.Error())
	}
	// check access
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	if data.UserId != u.Id {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "invalid access")
	}
	resp = &types.Ticket{}
	*resp = ticketview.FromDetails(data)
	return resp, nil
}
