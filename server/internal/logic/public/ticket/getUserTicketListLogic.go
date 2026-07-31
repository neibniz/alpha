package ticket

import (
	"context"

	ticketmodel "github.com/neibniz/alpha/server/internal/model/ticket"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/ticketview"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetUserTicketListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get ticket list
func NewGetUserTicketListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserTicketListLogic {
	return &GetUserTicketListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserTicketListLogic) GetUserTicketList(req *types.GetUserTicketListRequest) (resp *types.GetUserTicketListResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	l.Logger.Debugf("Current user: %v", u.Id)
	total, list, err := l.svcCtx.TicketModel.QueryTicketList(l.ctx, req.Page, req.Size, u.Id, req.Status, req.Search, ticketmodel.TicketListOptions{
		IncludeDescriptionPreview: true,
	})
	if err != nil {
		l.Errorw("[GetUserTicketListLogic] Database Error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "QueryTicketList error: %v", err)
	}
	resp = &types.GetUserTicketListResponse{
		Total: total,
		List:  ticketview.UserTicketListRows(list),
	}
	return
}
