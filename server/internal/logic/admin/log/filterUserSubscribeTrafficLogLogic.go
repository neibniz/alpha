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

type FilterUserSubscribeTrafficLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewFilterUserSubscribeTrafficLogLogic Filter user subscribe traffic log
func NewFilterUserSubscribeTrafficLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilterUserSubscribeTrafficLogLogic {
	return &FilterUserSubscribeTrafficLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FilterUserSubscribeTrafficLogLogic) FilterUserSubscribeTrafficLog(req *types.FilterSubscribeTrafficRequest) (resp *types.FilterSubscribeTrafficResponse, err error) {
	req.Page, req.Size = pagination.NormalizeWithLimit(req.Page, req.Size, 10, pagination.MaxPageSize)
	query := applyDailyUserTrafficFilter(l.svcCtx.DB.WithContext(l.ctx).Model(&traffic.DailyUserTraffic{}), req)
	if req.Date != "" {
		query = query.Where("date = ?", req.Date)
	}
	var total int64
	err = query.Count(&total).Error
	list, offset, ok := newTrafficPageRows[types.UserSubscribeTrafficLog](total, req.Page, req.Size)
	if err == nil && ok {
		err = query.
			Select(userSubscribeTrafficLogDailySelectColumns()).
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
			if detailsErr := applyUserSubscribeTrafficDetailsAvailability(list, detailsCache); detailsErr != nil {
				l.Errorw("[FilterUserSubscribeTrafficLog] Parse Date Error", logger.Field("error", detailsErr.Error()))
			}
		}
	}
	if err != nil {
		l.Errorw("[FilterUserSubscribeTrafficLog] Query Database Error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "[FilterUserSubscribeTrafficLog] Query Database Error")
	}
	return &types.FilterSubscribeTrafficResponse{
		List:  list,
		Total: total,
	}, nil
}

func buildTodayUserSubscribeTrafficQuery(db *gorm.DB, req *types.FilterSubscribeTrafficRequest, start, end time.Time) *gorm.DB {
	query := db.Model(&traffic.TrafficLog{}).
		Select("user_id, subscribe_id, SUM(download + upload) AS total, SUM(download) AS download, SUM(upload) AS upload").
		Where("timestamp >= ? AND timestamp < ?", start, end)
	if req.UserId != 0 {
		query = query.Where("user_id = ?", req.UserId)
	}
	if req.UserSubscribeId != 0 {
		query = query.Where("subscribe_id = ?", req.UserSubscribeId)
	}
	return query.Group("user_id, subscribe_id")
}

func applyDailyUserTrafficFilter(query *gorm.DB, req *types.FilterSubscribeTrafficRequest) *gorm.DB {
	if req.UserId != 0 {
		query = query.Where("user_id = ?", req.UserId)
	}
	if req.UserSubscribeId != 0 {
		query = query.Where("subscribe_id = ?", req.UserSubscribeId)
	}
	return query
}

func countGroupedTraffic(db *gorm.DB, query *gorm.DB) (int64, error) {
	var total int64
	err := db.Table("(?) AS grouped_traffic", query).Count(&total).Error
	return total, err
}
