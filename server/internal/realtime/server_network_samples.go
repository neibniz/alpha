package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	ServerNetworkSampleTTL      = 72 * time.Hour
	ServerNetworkSampleInterval = 5 * time.Second
)

type ServerNetworkSample struct {
	Timestamp int64 `json:"timestamp"`
	NetRxBps  int64 `json:"net_rx_bps"`
	NetTxBps  int64 `json:"net_tx_bps"`
	XrayRxBps int64 `json:"xray_rx_bps"`
	XrayTxBps int64 `json:"xray_tx_bps"`
}

func serverNetworkSeriesKey(serverID int64) string {
	return fmt.Sprintf("server:network_rate:%d:samples", serverID)
}

// RecordServerNetworkSnapshot keeps the latest value in each five-second bucket.
func RecordServerNetworkSnapshot(ctx context.Context, rdb redis.Cmdable, now time.Time, states []NodeStatus) error {
	if rdb == nil || len(states) == 0 {
		return nil
	}

	timestamp := now.Truncate(ServerNetworkSampleInterval).UnixMilli()
	cutoff := now.Add(-ServerNetworkSampleTTL).UnixMilli()
	pipe := rdb.Pipeline()
	for _, state := range states {
		if state.ServerID <= 0 || !state.Online {
			continue
		}
		sample := ServerNetworkSample{
			Timestamp: timestamp,
			NetRxBps:  state.NetRxBps,
			NetTxBps:  state.NetTxBps,
			XrayRxBps: state.XrayRxBps,
			XrayTxBps: state.XrayTxBps,
		}
		value, err := json.Marshal(sample)
		if err != nil {
			return err
		}

		key := serverNetworkSeriesKey(state.ServerID)
		pipe.ZRemRangeByScore(ctx, key, strconv.FormatInt(timestamp, 10), strconv.FormatInt(timestamp, 10))
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(timestamp), Member: string(value)})
		pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(cutoff, 10))
		pipe.Expire(ctx, key, ServerNetworkSampleTTL)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func QueryServerNetworkSamples(
	ctx context.Context,
	rdb redis.Cmdable,
	serverIDs []int64,
	startTime int64,
	endTime int64,
) (map[int64][]ServerNetworkSample, error) {
	result := make(map[int64][]ServerNetworkSample, len(serverIDs))
	if rdb == nil || len(serverIDs) == 0 {
		return result, nil
	}

	pipe := rdb.Pipeline()
	commands := make(map[int64]*redis.StringSliceCmd, len(serverIDs))
	for _, serverID := range serverIDs {
		commands[serverID] = pipe.ZRangeByScore(ctx, serverNetworkSeriesKey(serverID), &redis.ZRangeBy{
			Min: strconv.FormatInt(startTime, 10),
			Max: strconv.FormatInt(endTime, 10),
		})
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, err
	}

	for _, serverID := range serverIDs {
		values, err := commands[serverID].Result()
		if err != nil && err != redis.Nil {
			return nil, err
		}
		points := make([]ServerNetworkSample, 0, len(values))
		for _, value := range values {
			var point ServerNetworkSample
			if err := json.Unmarshal([]byte(value), &point); err != nil {
				continue
			}
			points = append(points, point)
		}
		result[serverID] = points
	}
	return result, nil
}

func CompactServerNetworkSamples(points []ServerNetworkSample, interval time.Duration) []ServerNetworkSample {
	if len(points) == 0 {
		return []ServerNetworkSample{}
	}
	if interval <= ServerNetworkSampleInterval {
		result := make([]ServerNetworkSample, len(points))
		copy(result, points)
		return result
	}

	intervalMs := interval.Milliseconds()
	result := make([]ServerNetworkSample, 0, len(points))
	var current ServerNetworkSample
	count := int64(0)
	flush := func() {
		if count == 0 {
			return
		}
		current.NetRxBps /= count
		current.NetTxBps /= count
		current.XrayRxBps /= count
		current.XrayTxBps /= count
		result = append(result, current)
	}

	for _, point := range points {
		bucket := point.Timestamp / intervalMs * intervalMs
		if count > 0 && current.Timestamp != bucket {
			flush()
			current = ServerNetworkSample{}
			count = 0
		}
		if count == 0 {
			current.Timestamp = bucket
		}
		current.NetRxBps += point.NetRxBps
		current.NetTxBps += point.NetTxBps
		current.XrayRxBps += point.XrayRxBps
		current.XrayTxBps += point.XrayTxBps
		count++
	}
	flush()
	return result
}
