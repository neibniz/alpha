package trafficview

import (
	"context"
	"fmt"
	"time"

	"github.com/neibniz/alpha/server/internal/analytics"
	"github.com/neibniz/alpha/server/internal/types"
)

type SeriesWindow struct {
	Start    time.Time
	End      time.Time
	Interval time.Duration
}

func ResolveSeriesWindow(period string, now time.Time) (SeriesWindow, error) {
	var duration, interval time.Duration
	switch period {
	case "5m":
		duration, interval = 5*time.Minute, 5*time.Second
	case "15m":
		duration, interval = 15*time.Minute, 5*time.Second
	case "", "24h":
		duration, interval = 24*time.Hour, 5*time.Minute
	case "1h":
		duration, interval = time.Hour, time.Minute
	case "7d":
		duration, interval = 7*24*time.Hour, time.Hour
	case "30d":
		duration, interval = 30*24*time.Hour, 6*time.Hour
	default:
		return SeriesWindow{}, fmt.Errorf("unsupported traffic series period %q", period)
	}

	end := now.UTC().Truncate(interval).Add(interval)
	return SeriesWindow{
		Start:    end.Add(-duration),
		End:      end,
		Interval: interval,
	}, nil
}

func QuerySeries(
	ctx context.Context,
	store analytics.Store,
	userID int64,
	userSubscribeID int64,
	period string,
	now time.Time,
) (*types.TrafficSeriesResponse, error) {
	window, err := ResolveSeriesWindow(period, now)
	if err != nil {
		return nil, err
	}
	series, err := store.QueryTrafficSeries(ctx, analytics.TrafficSeriesFilter{
		Start:           window.Start,
		End:             window.End,
		Interval:        window.Interval,
		UserID:          userID,
		UserSubscribeID: userSubscribeID,
	})
	if err != nil {
		return nil, err
	}
	return BuildSeriesResponse(window, series), nil
}

func BuildSeriesResponse(window SeriesWindow, series analytics.TrafficSeries) *types.TrafficSeriesResponse {
	pointCount := int(window.End.Sub(window.Start) / window.Interval)
	points := make([]types.TrafficSeriesPoint, pointCount)
	indexByTimestamp := make(map[int64]int, len(series.List))
	for index := range pointCount {
		timestamp := window.Start.Add(time.Duration(index) * window.Interval).UnixMilli()
		points[index].Timestamp = timestamp
		indexByTimestamp[timestamp] = index
	}
	for _, point := range series.List {
		index, ok := indexByTimestamp[point.Timestamp.UnixMilli()]
		if !ok {
			continue
		}
		points[index].Download = point.Download
		points[index].Upload = point.Upload
	}
	return &types.TrafficSeriesResponse{
		StartTime:       window.Start.UnixMilli(),
		EndTime:         window.End.UnixMilli(),
		IntervalSeconds: int64(window.Interval / time.Second),
		Download:        series.Download,
		Upload:          series.Upload,
		List:            points,
	}
}
