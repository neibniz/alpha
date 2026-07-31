package ticket

import (
	"context"

	ticketmodel "github.com/neibniz/alpha/server/internal/model/ticket"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/ticketview"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetTicketListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get ticket list
func NewGetTicketListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTicketListLogic {
	return &GetTicketListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTicketListLogic) GetTicketList(req *types.GetTicketListRequest) (resp *types.GetTicketListResponse, err error) {
	total, list, err := l.svcCtx.TicketModel.QueryTicketList(l.ctx, int(req.Page), int(req.Size), req.UserId, req.Status, req.Search, ticketmodel.TicketListOptions{
		IncludeUserID: true,
	})
	if err != nil {
		l.Errorw("[GetTicketList] Query Database Error: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "QueryTicketList error: %v", err)
	}
	resp = &types.GetTicketListResponse{
		Total: total,
		List:  ticketview.TicketListRows(list),
	}
	return
}
