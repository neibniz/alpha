package log

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/analytics"
	"github.com/neibniz/alpha/server/internal/model/pagination"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type FilterTrafficLogDetailsLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewFilterTrafficLogDetailsLogic Filter traffic log details
func NewFilterTrafficLogDetailsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilterTrafficLogDetailsLogic {
	return &FilterTrafficLogDetailsLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FilterTrafficLogDetailsLogic) FilterTrafficLogDetails(req *types.FilterTrafficLogDetailsRequest) (resp *types.FilterTrafficLogDetailsResponse, err error) {
	req.Page, req.Size = pagination.NormalizeWithLimit(req.Page, req.Size, 10, pagination.MaxPageSize)
	if !l.svcCtx.Analytics.Available() {
		return nil, xerr.NewErrMsg("analytics storage is unavailable")
	}
	var start, end time.Time
	if req.Date != "" {
		day, err := time.ParseInLocation("2006-01-02", req.Date, time.Local)
		if err != nil {
			l.Errorw("[FilterTrafficLogDetails] Date Parse Error", logger.Field("error", err.Error()))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), " date parse error: %s", err.Error())
		}
		start = day
		end = day.Add(24 * time.Hour)
	} else {
		// query today
		now := time.Now()
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = start.Add(24 * time.Hour)
	}
	page, err := l.svcCtx.Analytics.QueryTrafficEvents(l.ctx, analytics.TrafficFilter{
		Start:           start,
		End:             end,
		ServerID:        req.ServerId,
		UserID:          req.UserId,
		UserSubscribeID: req.SubscribeId,
		Limit:           req.Size,
		Offset:          pagination.Offset(req.Page, req.Size),
	})
	if err != nil {
		l.Errorw("[FilterTrafficLogDetails] Query Database Error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), " database query error: %s", err.Error())
	}

	list := make([]types.TrafficLogDetails, 0, len(page.List))
	for _, event := range page.List {
		list = append(list, types.TrafficLogDetails{
			ServerId:    event.ServerID,
			UserId:      event.UserID,
			SubscribeId: event.UserSubscribeID,
			Download:    event.Download,
			Upload:      event.Upload,
			Timestamp:   event.EventTime.UnixMilli(),
		})
	}
	return &types.FilterTrafficLogDetailsResponse{List: list, Total: page.Total}, nil
}
