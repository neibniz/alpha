package ticket

import (
	"context"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/ticket"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/ticketpolicy"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type CreateUserTicketFollowLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Create ticket follow
func NewCreateUserTicketFollowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserTicketFollowLogic {
	return &CreateUserTicketFollowLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateUserTicketFollowLogic) CreateUserTicketFollow(req *types.CreateUserTicketFollowRequest) error {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	content, images, err := ticketpolicy.NormalizeMessage(req.Type, req.Content, req.Images)
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid ticket reply")
	}
	err = l.svcCtx.TicketModel.InsertTicketFollowAndUpdateStatus(l.ctx, &ticket.Follow{
		TicketId: req.TicketId,
		From:     ticketpolicy.FromUser,
		Type:     req.Type,
		Content:  content,
		Images:   ticket.ImageGallery(images),
	}, u.Id, ticket.Pending)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "invalid ticket access")
		}
		l.Errorw("[CreateUserTicketFollow] Database transaction error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "create ticket follow failed: %v", err.Error())
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventTicket)
	}
	return nil
}
