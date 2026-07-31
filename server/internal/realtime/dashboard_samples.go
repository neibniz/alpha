package realtime

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	DashboardNetworkSeriesIndexKey  = "dashboard:network_rate:samples:index"
	DashboardNetworkSeriesValuesKey = "dashboard:network_rate:samples:values"
	DashboardNetworkSampleTTL       = 72 * time.Hour
	DashboardNetworkSampleInterval  = 5 * time.Second
)

type DashboardNetworkSample struct {
	Timestamp   int64 `json:"timestamp"`
	SystemRxBps int64 `json:"system_rx_bps"`
	SystemTxBps int64 `json:"system_tx_bps"`
	XrayRxBps   int64 `json:"xray_rx_bps"`
	XrayTxBps   int64 `json:"xray_tx_bps"`
}

func RecordDashboardNetworkSnapshot(ctx context.Context, rdb redis.Cmdable, states []NodeStatus) error {
	if rdb == nil {
		return nil
	}
	now := time.Now()
	sample := DashboardNetworkSample{
		Timestamp: now.Truncate(DashboardNetworkSampleInterval).UnixMilli(),
	}
	for _, state := range states {
		if !state.Online {
			continue
		}
		sample.SystemRxBps += state.NetRxBps
		sample.SystemTxBps += state.NetTxBps
		sample.XrayRxBps += state.XrayRxBps
		sample.XrayTxBps += state.XrayTxBps
	}
	return RecordDashboardNetworkSample(ctx, rdb, now, sample)
}

func RecordDashboardNetworkSample(ctx context.Context, rdb redis.Cmdable, now time.Time, sample DashboardNetworkSample) error {
	if rdb == nil {
		return nil
	}
	member := strconv.FormatInt(sample.Timestamp, 10)
	value, err := json.Marshal(sample)
	if err != nil {
		return err
	}
	cutoff := strconv.FormatInt(now.Add(-DashboardNetworkSampleTTL).UnixMilli(), 10)
	if oldMembers, err := rdb.ZRangeByScore(ctx, DashboardNetworkSeriesIndexKey, &redis.ZRangeBy{Min: "-inf", Max: cutoff}).Result(); err == nil && len(oldMembers) > 0 {
		_ = rdb.HDel(ctx, DashboardNetworkSeriesValuesKey, oldMembers...).Err()
	}
	pipe := rdb.Pipeline()
	pipe.ZAdd(ctx, DashboardNetworkSeriesIndexKey, redis.Z{Score: float64(sample.Timestamp), Member: member})
	pipe.HSet(ctx, DashboardNetworkSeriesValuesKey, member, string(value))
	pipe.Expire(ctx, DashboardNetworkSeriesIndexKey, DashboardNetworkSampleTTL)
	pipe.Expire(ctx, DashboardNetworkSeriesValuesKey, DashboardNetworkSampleTTL)
	pipe.ZRemRangeByScore(ctx, DashboardNetworkSeriesIndexKey, "-inf", cutoff)
	_, err = pipe.Exec(ctx)
	return err
}
