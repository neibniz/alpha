package log

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/pagination"
	"github.com/neibniz/alpha/server/internal/model/traffic"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type FilterServerTrafficLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewFilterServerTrafficLogLogic Filter server traffic log
func NewFilterServerTrafficLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilterServerTrafficLogLogic {
	return &FilterServerTrafficLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}
func (l *FilterServerTrafficLogLogic) FilterServerTrafficLog(req *types.FilterServerTrafficLogRequest) (resp *types.FilterServerTrafficLogResponse, err error) {
	req.Page, req.Size = pagination.NormalizeWithLimit(req.Page, req.Size, 10, pagination.MaxPageSize)
	query := applyDailyServerTrafficFilter(l.svcCtx.DB.WithContext(l.ctx).Model(&traffic.DailyServerTraffic{}), req)
	if req.Date != "" {
		query = query.Where("date = ?", req.Date)
	}
	var total int64
	err = query.Count(&total).Error
	list, offset, ok := newTrafficPageRows[types.ServerTrafficLog](total, req.Page, req.Size)
	if err == nil && ok {
		err = query.
			Select(serverTrafficLogDailySelectColumns()).
			Order("date DESC, total DESC, id DESC").
			Limit(req.Size).
			Offset(offset).
			Scan(&list).Error
		if err == nil && l.svcCtx.Analytics.Available() {
			retentionHours := int64(l.svcCtx.Config.ClickHouse.TrafficRawRetentionHours)
			if retentionHours <= 0 {
				retentionHours = 72
			}
			retention := (retentionHours + 23) / 24
			detailsCache := newTrafficDetailsAvailabilityCacheWithHint(time.Now(), true, retention, len(list))
			if detailsErr := applyServerTrafficDetailsAvailability(list, detailsCache); detailsErr != nil {
				l.Errorw("[FilterServerTrafficLog] Parse Date Error", logger.Field("error", detailsErr.Error()))
			}
		}
	}
	if err != nil {
		l.Errorw("[FilterServerTrafficLog] Query Database Error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "history query error: %s", err.Error())
	}

	return &types.FilterServerTrafficLogResponse{
		List:  list,
		Total: total,
	}, nil
}

func buildTodayServerTrafficQuery(db *gorm.DB, req *types.FilterServerTrafficLogRequest, start, end time.Time) *gorm.DB {
	query := db.Model(&traffic.TrafficLog{}).
		Select("server_id, SUM(download + upload) AS total, SUM(download) AS download, SUM(upload) AS upload").
		Where("timestamp >= ? AND timestamp < ?", start, end)
	if req.ServerId != 0 {
		query = query.Where("server_id = ?", req.ServerId)
	}
	return query.Group("server_id")
}

func applyDailyServerTrafficFilter(query *gorm.DB, req *types.FilterServerTrafficLogRequest) *gorm.DB {
	if req.ServerId != 0 {
		query = query.Where("server_id = ?", req.ServerId)
	}
	return query
}
