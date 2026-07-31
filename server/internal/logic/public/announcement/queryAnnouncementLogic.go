package announcement

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/announcement"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type QueryAnnouncementLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Query announcement
func NewQueryAnnouncementLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryAnnouncementLogic {
	return &QueryAnnouncementLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryAnnouncementLogic) QueryAnnouncement(req *types.QueryAnnouncementRequest) (resp *types.QueryAnnouncementResponse, err error) {
	enable := true
	total, list, err := l.svcCtx.AnnouncementModel.GetAnnouncementListByPage(l.ctx, req.Page, req.Size, announcement.Filter{
		Show:           &enable,
		Pinned:         req.Pinned,
		Popup:          req.Popup,
		IncludeContent: true,
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "GetAnnouncementListByPage error: %v", err.Error())
	}
	resp = &types.QueryAnnouncementResponse{}
	resp.Total = total
	resp.List = announcementRows(list)
	return
}
