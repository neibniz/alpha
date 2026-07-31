package ticket

import (
	"context"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/ticket"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/ticketpolicy"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type CreateUserTicketLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Create ticket
func NewCreateUserTicketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserTicketLogic {
	return &CreateUserTicketLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateUserTicketLogic) CreateUserTicket(req *types.CreateUserTicketRequest) error {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	title, description, err := ticketpolicy.NormalizeTicket(req.Title, req.Description)
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid ticket content")
	}
	images, err := ticketpolicy.NormalizeImages(req.Images)
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid ticket images")
	}
	err = l.svcCtx.TicketModel.Insert(l.ctx, &ticket.Ticket{
		Title:       title,
		Description: description,
		Images:      ticket.ImageGallery(images),
		UserId:      u.Id,
		Status:      ticket.Pending,
	})
	if err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "insert ticket error: %v", err.Error())
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventTicket)
	}
	return nil
}
