package ticket

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/ticket"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/ticketpolicy"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type CreateTicketFollowLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Create ticket follow
func NewCreateTicketFollowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTicketFollowLogic {
	return &CreateTicketFollowLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateTicketFollowLogic) CreateTicketFollow(req *types.CreateTicketFollowRequest) (err error) {
	content, images, err := ticketpolicy.NormalizeMessage(req.Type, req.Content, req.Images)
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid ticket reply")
	}
	err = l.svcCtx.TicketModel.InsertTicketFollowAndUpdateStatus(l.ctx, &ticket.Follow{
		TicketId: req.TicketId,
		From:     ticketpolicy.FromSystem,
		Type:     req.Type,
		Content:  content,
		Images:   ticket.ImageGallery(images),
	}, 0, ticket.Waiting)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, ticket.ErrInvalidTicketTransition) {
			return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "ticket is closed or unavailable")
		}
		l.Errorw("[CreateTicketFollow] Database transaction error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "create ticket follow failed: %v", err.Error())
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventTicket)
	}
	return
}
