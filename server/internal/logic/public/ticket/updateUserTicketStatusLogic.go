package ticket

import (
	"context"

	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/ticketpolicy"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type UpdateUserTicketStatusLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Update ticket status
func NewUpdateUserTicketStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserTicketStatusLogic {
	return &UpdateUserTicketStatusLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserTicketStatusLogic) UpdateUserTicketStatus(req *types.UpdateUserTicketStatusRequest) error {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	if req.Status == nil || !ticketpolicy.UserStatusAllowed(*req.Status) {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid ticket status")
	}
	err := l.svcCtx.TicketModel.UpdateTicketStatus(l.ctx, req.Id, u.Id, *req.Status)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "invalid ticket access or transition")
		}
		l.Errorw("[UpdateUserTicketStatusLogic] Database Error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update ticket error: %v", err.Error())
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventTicket)
	}
	return nil
}
