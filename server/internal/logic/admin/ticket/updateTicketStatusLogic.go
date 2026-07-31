package ticket

import (
	"context"

	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/ticketpolicy"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type UpdateTicketStatusLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Update ticket status
func NewUpdateTicketStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateTicketStatusLogic {
	return &UpdateTicketStatusLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateTicketStatusLogic) UpdateTicketStatus(req *types.UpdateTicketStatusRequest) error {
	if req.Status == nil || !ticketpolicy.AdminStatusAllowed(*req.Status) {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid ticket status")
	}
	err := l.svcCtx.TicketModel.UpdateTicketStatus(l.ctx, req.Id, 0, *req.Status)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "ticket transition is not allowed")
		}
		l.Errorw("[UpdateTicketStatus] Update Database Error: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update ticket error: %v", err.Error())
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventTicket)
	}
	return nil
}
