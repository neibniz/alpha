package user

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/analytics"
	"github.com/neibniz/alpha/server/internal/model/pagination"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type GetUserSubscribeTrafficLogsLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get user subcribe traffic logs
func NewGetUserSubscribeTrafficLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserSubscribeTrafficLogsLogic {
	return &GetUserSubscribeTrafficLogsLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserSubscribeTrafficLogsLogic) GetUserSubscribeTrafficLogs(req *types.GetUserSubscribeTrafficLogsRequest) (resp *types.GetUserSubscribeTrafficLogsResponse, err error) {
	if !l.svcCtx.Analytics.Available() {
		return nil, xerr.NewErrMsg("analytics storage is unavailable")
	}
	req.Page, req.Size = pagination.NormalizeWithLimit(req.Page, req.Size, 10, pagination.MaxPageSize)
	retentionHours := l.svcCtx.Config.ClickHouse.TrafficRawRetentionHours
	if retentionHours <= 0 {
		retentionHours = 72
	}
	end := time.Now().Add(time.Hour)
	page, err := l.svcCtx.Analytics.QueryTrafficEvents(l.ctx, analytics.TrafficFilter{
		Start:           end.Add(-time.Duration(retentionHours) * time.Hour),
		End:             end,
		UserID:          req.UserId,
		UserSubscribeID: req.SubscribeId,
		Limit:           req.Size,
		Offset:          pagination.Offset(req.Page, req.Size),
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "GetUserSubscribeTrafficLogs failed: %v", err.Error())
	}
	list := make([]types.TrafficLog, 0, len(page.List))
	for _, event := range page.List {
		list = append(list, types.TrafficLog{
			ServerId:    event.ServerID,
			UserId:      event.UserID,
			SubscribeId: event.UserSubscribeID,
			Download:    event.Download,
			Upload:      event.Upload,
			Timestamp:   event.EventTime.UnixMilli(),
		})
	}
	return &types.GetUserSubscribeTrafficLogsResponse{
		Total: page.Total,
		List:  list,
	}, nil
}
