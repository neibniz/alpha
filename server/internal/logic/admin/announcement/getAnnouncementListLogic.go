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

type GetAnnouncementListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get announcement list
func NewGetAnnouncementListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAnnouncementListLogic {
	return &GetAnnouncementListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAnnouncementListLogic) GetAnnouncementList(req *types.GetAnnouncementListRequest) (resp *types.GetAnnouncementListResponse, err error) {
	total, list, err := l.svcCtx.AnnouncementModel.GetAnnouncementListByPage(l.ctx, int(req.Page), int(req.Size), announcement.Filter{
		Show:   req.Show,
		Pinned: req.Pinned,
		Popup:  req.Popup,
		Search: req.Search,
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "GetAnnouncementListByPage error: %v", err.Error())
	}
	resp = &types.GetAnnouncementListResponse{}
	resp.Total = total
	resp.List = announcementRows(list, false)
	return
}
