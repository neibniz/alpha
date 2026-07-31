package console

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QueryRevenueStatisticsLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type revenueTotalsRow struct {
	TodayAmountTotal        int64
	TodayNewOrderAmount     int64
	TodayRenewalOrderAmount int64
	TodayMembershipAmount   int64
	TodayResetTrafficAmount int64

	MonthlyAmountTotal        int64
	MonthlyNewOrderAmount     int64
	MonthlyRenewalOrderAmount int64
	MonthlyMembershipAmount   int64
	MonthlyResetTrafficAmount int64

	AllAmountTotal        int64
	AllNewOrderAmount     int64
	AllRenewalOrderAmount int64
	AllMembershipAmount   int64
	AllResetTrafficAmount int64
}

type revenueSeriesRow struct {
	Bucket             time.Time
	AmountTotal        int64
	NewOrderAmount     int64
	RenewalOrderAmount int64
	MembershipAmount   int64
	ResetTrafficAmount int64
}

func (r revenueTotalsRow) today() types.OrdersStatistics {
	return types.OrdersStatistics{
		AmountTotal:        r.TodayAmountTotal,
		NewOrderAmount:     r.TodayNewOrderAmount,
		RenewalOrderAmount: r.TodayRenewalOrderAmount,
		MembershipAmount:   r.TodayMembershipAmount,
		ResetTrafficAmount: r.TodayResetTrafficAmount,
	}
}

func (r revenueTotalsRow) monthly() types.OrdersStatistics {
	return types.OrdersStatistics{
		AmountTotal:        r.MonthlyAmountTotal,
		NewOrderAmount:     r.MonthlyNewOrderAmount,
		RenewalOrderAmount: r.MonthlyRenewalOrderAmount,
		MembershipAmount:   r.MonthlyMembershipAmount,
		ResetTrafficAmount: r.MonthlyResetTrafficAmount,
	}
}

func (r revenueTotalsRow) all() types.OrdersStatistics {
	return types.OrdersStatistics{
		AmountTotal:        r.AllAmountTotal,
		NewOrderAmount:     r.AllNewOrderAmount,
		RenewalOrderAmount: r.AllRenewalOrderAmount,
		MembershipAmount:   r.AllMembershipAmount,
		ResetTrafficAmount: r.AllResetTrafficAmount,
	}
}

// NewQueryRevenueStatisticsLogic Query revenue statistics
func NewQueryRevenueStatisticsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryRevenueStatisticsLogic {
	return &QueryRevenueStatisticsLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryRevenueStatisticsLogic) QueryRevenueStatistics(req *types.StatisticsPeriodRequest) (resp *types.RevenueStatisticsResponse, err error) {
	if strings.ToLower(os.Getenv("ALPHA_MODE")) == "demo" {
		return l.mockRevenueStatistics(), nil
	}

	now := time.Now()
	period := ""
	if req != nil {
		period = req.Period
	}
	selectedRange := resolveStatisticsRange(period, now)

	series, err := l.queryRevenueSeries(selectedRange)
	if err != nil {
		l.Errorw("[QueryRevenueStatisticsLogic] queryRevenueSeries error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query revenue series error: %v", err)
	}

	todayStart := startOfDay(now)
	todayEnd := todayStart.AddDate(0, 0, 1)
	monthStart := startOfMonth(now)
	monthEnd := monthStart.AddDate(0, 1, 0)
	totals, err := l.queryRevenueTotals(todayStart, todayEnd, monthStart, monthEnd)
	if err != nil {
		l.Errorw("[QueryRevenueStatisticsLogic] query revenue totals error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query revenue totals error: %v", err)
	}
	today := totals.today()
	monthly := totals.monthly()
	all := totals.all()

	return &types.RevenueStatisticsResponse{
		Period:      selectedRange.Period,
		Granularity: selectedRange.Granularity,
		Series:      series,
		Today:       today,
		Monthly:     monthly,
		All:         all,
	}, nil
}

func (l *QueryRevenueStatisticsLogic) queryRevenueTotals(todayStart, todayEnd, monthStart, monthEnd time.Time) (revenueTotalsRow, error) {
	var result revenueTotalsRow
	err := l.svcCtx.DB.WithContext(l.ctx).
		Raw(revenueTotalsSQL,
			todayStart, todayEnd,
			todayStart, todayEnd, orderTypeSubscribe,
			todayStart, todayEnd, orderTypeRenewal,
			todayStart, todayEnd, orderTypeMembership,
			todayStart, todayEnd, orderTypeResetTraffic,
			monthStart, monthEnd,
			monthStart, monthEnd, orderTypeSubscribe,
			monthStart, monthEnd, orderTypeRenewal,
			monthStart, monthEnd, orderTypeMembership,
			monthStart, monthEnd, orderTypeResetTraffic,
			orderTypeSubscribe,
			orderTypeRenewal,
			orderTypeMembership,
			orderTypeResetTraffic,
			orderStatusPaid, orderStatusFinished,
			orderTypeSubscribe, orderTypeRenewal, orderTypeResetTraffic, orderTypeMembership,
		).
		Scan(&result).Error
	return result, err
}

const revenueTotalsSQL = `
SELECT
	COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN amount ELSE 0 END), 0) AS today_amount_total,
	COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? AND type = ? THEN amount ELSE 0 END), 0) AS today_new_order_amount,
	COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? AND type = ? THEN amount ELSE 0 END), 0) AS today_renewal_order_amount,
	COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? AND type = ? THEN amount ELSE 0 END), 0) AS today_membership_amount,
	COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? AND type = ? THEN amount ELSE 0 END), 0) AS today_reset_traffic_amount,

	COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN amount ELSE 0 END), 0) AS monthly_amount_total,
	COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? AND type = ? THEN amount ELSE 0 END), 0) AS monthly_new_order_amount,
	COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? AND type = ? THEN amount ELSE 0 END), 0) AS monthly_renewal_order_amount,
	COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? AND type = ? THEN amount ELSE 0 END), 0) AS monthly_membership_amount,
	COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? AND type = ? THEN amount ELSE 0 END), 0) AS monthly_reset_traffic_amount,

	COALESCE(SUM(amount), 0) AS all_amount_total,
	COALESCE(SUM(CASE WHEN type = ? THEN amount ELSE 0 END), 0) AS all_new_order_amount,
	COALESCE(SUM(CASE WHEN type = ? THEN amount ELSE 0 END), 0) AS all_renewal_order_amount,
	COALESCE(SUM(CASE WHEN type = ? THEN amount ELSE 0 END), 0) AS all_membership_amount,
	COALESCE(SUM(CASE WHEN type = ? THEN amount ELSE 0 END), 0) AS all_reset_traffic_amount
FROM "order"
WHERE status IN (?, ?) AND type IN (?, ?, ?, ?)
`

func (l *QueryRevenueStatisticsLogic) queryRevenueSeries(r statisticsRange) ([]types.OrdersStatistics, error) {
	var rows []revenueSeriesRow
	err := l.svcCtx.DB.WithContext(l.ctx).
		Model(&struct{}{}).
		Table(order.Order{}.TableName()).
		Where("status IN ? AND type IN ? AND created_at >= ? AND created_at < ?", paidOrderStatuses, businessOrderTypes, r.Start, r.End).
		Select(r.BucketSQL+` AS bucket,
			COALESCE(SUM(amount), 0) AS amount_total,
			COALESCE(SUM(CASE WHEN type = ? THEN amount ELSE 0 END), 0) AS new_order_amount,
			COALESCE(SUM(CASE WHEN type = ? THEN amount ELSE 0 END), 0) AS renewal_order_amount,
			COALESCE(SUM(CASE WHEN type = ? THEN amount ELSE 0 END), 0) AS membership_amount,
			COALESCE(SUM(CASE WHEN type = ? THEN amount ELSE 0 END), 0) AS reset_traffic_amount
		`, orderTypeSubscribe, orderTypeRenewal, orderTypeMembership, orderTypeResetTraffic).
		Group("bucket").
		Order("bucket ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return revenueSeriesRowsToStatistics(rows, r), nil
}

func revenueSeriesRowsToStatistics(rows []revenueSeriesRow, r statisticsRange) []types.OrdersStatistics {
	buckets := newStatisticsBucketSeries(r)
	if buckets.len() == 0 {
		return []types.OrdersStatistics{}
	}
	series := make([]types.OrdersStatistics, buckets.len())
	for i := range series {
		series[i].Date = buckets.date(i)
	}
	for _, item := range rows {
		index, ok := buckets.index(item.Bucket)
		if !ok {
			continue
		}
		series[index] = types.OrdersStatistics{
			Date:               series[index].Date,
			AmountTotal:        item.AmountTotal,
			NewOrderAmount:     item.NewOrderAmount,
			RenewalOrderAmount: item.RenewalOrderAmount,
			MembershipAmount:   item.MembershipAmount,
			ResetTrafficAmount: item.ResetTrafficAmount,
		}
	}
	return series
}

// mockRevenueStatistics is a mock function to simulate revenue statistics data.
func (l *QueryRevenueStatisticsLogic) mockRevenueStatistics() *types.RevenueStatisticsResponse {
	now := time.Now()

	// Generate daily data for the current month (from 1st to current date)
	series := make([]types.OrdersStatistics, 7)
	for i := 0; i < 7; i++ {
		dayDate := now.AddDate(0, 0, -(6 - i))
		baseAmount := int64(25000 + ((6 - i) * 3000) + ((6-i)%3)*8000)
		series[i] = types.OrdersStatistics{
			Date:               dayDate.Format("2006-01-02"),
			AmountTotal:        baseAmount,
			NewOrderAmount:     int64(float64(baseAmount) * 0.68),
			RenewalOrderAmount: int64(float64(baseAmount) * 0.32),
		}
	}

	return &types.RevenueStatisticsResponse{
		Period:      "7d",
		Granularity: "day",
		Series:      series,
		Today: types.OrdersStatistics{
			AmountTotal:        35888,
			NewOrderAmount:     22888,
			RenewalOrderAmount: 13000,
		},
		Monthly: types.OrdersStatistics{
			AmountTotal:        888888,
			NewOrderAmount:     588888,
			RenewalOrderAmount: 300000,
		},
		All: types.OrdersStatistics{
			AmountTotal:        12888888,
			NewOrderAmount:     8588888,
			RenewalOrderAmount: 4300000,
		},
	}
}
