package console

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QueryUserStatisticsLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type userSeriesRow struct {
	Bucket            time.Time
	Register          int64
	NewOrderUsers     int64
	RenewalOrderUsers int64
	MembershipUsers   int64
}

type userTotalsRow struct {
	TodayRegister          int64
	TodayNewOrderUsers     int64
	TodayRenewalOrderUsers int64
	TodayMembershipUsers   int64

	MonthlyRegister          int64
	MonthlyNewOrderUsers     int64
	MonthlyRenewalOrderUsers int64
	MonthlyMembershipUsers   int64

	AllRegister          int64
	AllNewOrderUsers     int64
	AllRenewalOrderUsers int64
	AllMembershipUsers   int64
}

func (r userTotalsRow) today() types.UserStatistics {
	return types.UserStatistics{
		Register:          r.TodayRegister,
		NewOrderUsers:     r.TodayNewOrderUsers,
		RenewalOrderUsers: r.TodayRenewalOrderUsers,
		MembershipUsers:   r.TodayMembershipUsers,
	}
}

func (r userTotalsRow) monthly() types.UserStatistics {
	return types.UserStatistics{
		Register:          r.MonthlyRegister,
		NewOrderUsers:     r.MonthlyNewOrderUsers,
		RenewalOrderUsers: r.MonthlyRenewalOrderUsers,
		MembershipUsers:   r.MonthlyMembershipUsers,
	}
}

func (r userTotalsRow) all() types.UserStatistics {
	return types.UserStatistics{
		Register:          r.AllRegister,
		NewOrderUsers:     r.AllNewOrderUsers,
		RenewalOrderUsers: r.AllRenewalOrderUsers,
		MembershipUsers:   r.AllMembershipUsers,
	}
}

// Query user statistics
func NewQueryUserStatisticsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryUserStatisticsLogic {
	return &QueryUserStatisticsLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryUserStatisticsLogic) QueryUserStatistics(req *types.StatisticsPeriodRequest) (resp *types.UserStatisticsResponse, err error) {
	if strings.ToLower(os.Getenv("ALPHA_MODE")) == "demo" {
		return l.mockRevenueStatistics(), nil
	}
	now := time.Now()
	period := ""
	if req != nil {
		period = req.Period
	}
	selectedRange := resolveStatisticsRange(period, now)
	series, err := l.queryUserSeries(selectedRange)
	if err != nil {
		l.Errorw("[QueryUserStatisticsLogic] queryUserSeries error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query user statistics series error: %v", err)
	}

	todayStart := startOfDay(now)
	todayEnd := todayStart.AddDate(0, 0, 1)
	monthStart := startOfMonth(now)
	monthEnd := monthStart.AddDate(0, 1, 0)
	totals, err := l.queryUserTotals(todayStart, todayEnd, monthStart, monthEnd)
	if err != nil {
		l.Errorw("[QueryUserStatisticsLogic] query user totals error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query user totals error: %v", err)
	}

	return &types.UserStatisticsResponse{
		Period:      selectedRange.Period,
		Granularity: selectedRange.Granularity,
		Series:      series,
		Today:       totals.today(),
		Monthly:     totals.monthly(),
		All:         totals.all(),
	}, nil
}

func (l *QueryUserStatisticsLogic) queryUserTotals(todayStart, todayEnd, monthStart, monthEnd time.Time) (userTotalsRow, error) {
	var result userTotalsRow
	if err := l.svcCtx.DB.WithContext(l.ctx).
		Raw(userTotalsSQL,
			todayStart, todayEnd,
			monthStart, monthEnd,
			todayStart, todayEnd, orderTypeSubscribe,
			todayStart, todayEnd, orderTypeRenewal,
			todayStart, todayEnd, orderTypeMembership,
			monthStart, monthEnd, orderTypeSubscribe,
			monthStart, monthEnd, orderTypeRenewal,
			monthStart, monthEnd, orderTypeMembership,
			orderTypeSubscribe,
			orderTypeRenewal,
			orderTypeMembership,
			orderStatusPaid, orderStatusFinished,
			orderTypeSubscribe, orderTypeRenewal, orderTypeResetTraffic, orderTypeMembership,
		).
		Scan(&result).Error; err != nil {
		return result, err
	}
	return result, nil
}

const userTotalsSQL = `
SELECT
	user_totals.today_register,
	user_totals.monthly_register,
	user_totals.all_register,
	order_totals.today_new_order_users,
	order_totals.today_renewal_order_users,
	order_totals.today_membership_users,
	order_totals.monthly_new_order_users,
	order_totals.monthly_renewal_order_users,
	order_totals.monthly_membership_users,
	order_totals.all_new_order_users,
	order_totals.all_renewal_order_users,
	order_totals.all_membership_users
FROM (
	SELECT
		COUNT(CASE WHEN created_at >= ? AND created_at < ? THEN 1 END) AS today_register,
		COUNT(CASE WHEN created_at >= ? AND created_at < ? THEN 1 END) AS monthly_register,
		COUNT(*) AS all_register
	FROM "user"
) AS user_totals
CROSS JOIN (
	SELECT
		COUNT(CASE WHEN created_at >= ? AND created_at < ? AND type = ? THEN 1 END) AS today_new_order_users,
		COUNT(CASE WHEN created_at >= ? AND created_at < ? AND type = ? THEN 1 END) AS today_renewal_order_users,
		COUNT(CASE WHEN created_at >= ? AND created_at < ? AND type = ? THEN 1 END) AS today_membership_users,
		COUNT(CASE WHEN created_at >= ? AND created_at < ? AND type = ? THEN 1 END) AS monthly_new_order_users,
		COUNT(CASE WHEN created_at >= ? AND created_at < ? AND type = ? THEN 1 END) AS monthly_renewal_order_users,
		COUNT(CASE WHEN created_at >= ? AND created_at < ? AND type = ? THEN 1 END) AS monthly_membership_users,
		COUNT(CASE WHEN type = ? THEN 1 END) AS all_new_order_users,
		COUNT(CASE WHEN type = ? THEN 1 END) AS all_renewal_order_users,
		COUNT(CASE WHEN type = ? THEN 1 END) AS all_membership_users
	FROM "order"
	WHERE status IN (?, ?) AND type IN (?, ?, ?, ?)
) AS order_totals
`

func (l *QueryUserStatisticsLogic) queryUserSeries(r statisticsRange) ([]types.UserStatistics, error) {
	var rows []userSeriesRow
	if err := l.svcCtx.DB.WithContext(l.ctx).
		Raw(userSeriesSQL(r.BucketSQL),
			r.Start, r.End,
			orderTypeSubscribe, orderTypeRenewal, orderTypeMembership,
			paidOrderStatuses, businessOrderTypes,
			r.Start, r.End,
		).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	return userSeriesRowsToStatistics(rows, r), nil
}

func userSeriesSQL(bucketSQL string) string {
	return `
SELECT
	bucket,
	SUM(register) AS register,
	SUM(new_order_users) AS new_order_users,
	SUM(renewal_order_users) AS renewal_order_users,
	SUM(membership_users) AS membership_users
FROM (
	SELECT
		` + bucketSQL + ` AS bucket,
		COUNT(*) AS register,
		CAST(0 AS BIGINT) AS new_order_users,
		CAST(0 AS BIGINT) AS renewal_order_users,
		CAST(0 AS BIGINT) AS membership_users
	FROM "user"
	WHERE created_at >= ? AND created_at < ?
	GROUP BY bucket
	UNION ALL
	SELECT
		` + bucketSQL + ` AS bucket,
		CAST(0 AS BIGINT) AS register,
		COUNT(CASE WHEN type = ? THEN 1 END) AS new_order_users,
		COUNT(CASE WHEN type = ? THEN 1 END) AS renewal_order_users,
		COUNT(CASE WHEN type = ? THEN 1 END) AS membership_users
	FROM "order"
	WHERE status IN ? AND type IN ? AND created_at >= ? AND created_at < ?
	GROUP BY bucket
) AS user_series
GROUP BY bucket
ORDER BY bucket ASC
`
}

func userSeriesRowsToStatistics(rows []userSeriesRow, r statisticsRange) []types.UserStatistics {
	buckets := newStatisticsBucketSeries(r)
	if buckets.len() == 0 {
		return []types.UserStatistics{}
	}
	series := make([]types.UserStatistics, buckets.len())
	for i := range series {
		series[i].Date = buckets.date(i)
	}
	for _, item := range rows {
		index, ok := buckets.index(item.Bucket)
		if !ok {
			continue
		}
		series[index] = types.UserStatistics{
			Date:              series[index].Date,
			Register:          item.Register,
			NewOrderUsers:     item.NewOrderUsers,
			RenewalOrderUsers: item.RenewalOrderUsers,
			MembershipUsers:   item.MembershipUsers,
		}
	}
	return series
}

func (l *QueryUserStatisticsLogic) mockRevenueStatistics() *types.UserStatisticsResponse {
	now := time.Now()

	// Generate daily user statistics for the current month (from 1st to current date)
	series := make([]types.UserStatistics, 7)
	for i := 0; i < 7; i++ {
		dayDate := now.AddDate(0, 0, -(6 - i))
		baseRegister := int64(18 + ((6 - i) * 3) + ((6-i)%3)*8)
		series[i] = types.UserStatistics{
			Date:              dayDate.Format("2006-01-02"),
			Register:          baseRegister,
			NewOrderUsers:     int64(float64(baseRegister) * 0.65),
			RenewalOrderUsers: int64(float64(baseRegister) * 0.35),
		}
	}

	return &types.UserStatisticsResponse{
		Period:      "7d",
		Granularity: "day",
		Series:      series,
		Today: types.UserStatistics{
			Register:          28,
			NewOrderUsers:     18,
			RenewalOrderUsers: 10,
		},
		Monthly: types.UserStatistics{
			Register:          888,
			NewOrderUsers:     588,
			RenewalOrderUsers: 300,
		},
		All: types.UserStatistics{
			Register:          18888,
			NewOrderUsers:     0, // This field is not used in All statistics
			RenewalOrderUsers: 0, // This field is not used in All statistics
		},
	}
}
