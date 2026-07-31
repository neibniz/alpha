package server

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

const (
	defaultServerNetworkHistoryRange = 5 * time.Minute
	maxServerNetworkHistoryRange     = 72 * time.Hour
	maxServerNetworkHistoryPoints    = 2000
	maxServerNetworkHistoryServers   = 50
)

var serverNetworkHistoryIntervals = map[string]time.Duration{
	"5s": 5 * time.Second,
	"1m": time.Minute,
	"5m": 5 * time.Minute,
	"1h": time.Hour,
}

type QueryServerNetworkRateHistoryLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryServerNetworkRateHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryServerNetworkRateHistoryLogic {
	return &QueryServerNetworkRateHistoryLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryServerNetworkRateHistoryLogic) QueryServerNetworkRateHistory(req *types.QueryServerNetworkRateHistoryRequest) (*types.QueryServerNetworkRateHistoryResponse, error) {
	serverIDs, startTime, endTime, interval, err := normalizeServerNetworkHistoryRequest(req, time.Now())
	if err != nil {
		return nil, err
	}

	servers, err := l.svcCtx.NodeModel.QueryServerList(l.ctx, serverIDs)
	if err != nil {
		l.Errorw("[QueryServerNetworkRateHistory] query servers error", logger.Field("error", err.Error()))
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "query servers")
	}
	if len(servers) != len(serverIDs) {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.NodeNotExist), "one or more servers do not exist")
	}

	stored, err := realtime.QueryServerNetworkSamples(l.ctx, l.svcCtx.Redis, serverIDs, startTime, endTime)
	if err != nil {
		l.Errorw("[QueryServerNetworkRateHistory] query samples error", logger.Field("error", err.Error()))
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "query network rate samples")
	}

	series := make([]types.ServerNetworkRateSeries, 0, len(serverIDs))
	for _, serverID := range serverIDs {
		points := realtime.CompactServerNetworkSamples(stored[serverID], interval)
		mapped := make([]types.ServerNetworkRatePoint, 0, len(points))
		for _, point := range points {
			mapped = append(mapped, types.ServerNetworkRatePoint{
				Timestamp: point.Timestamp,
				NetRxBps:  point.NetRxBps,
				NetTxBps:  point.NetTxBps,
				XrayRxBps: point.XrayRxBps,
				XrayTxBps: point.XrayTxBps,
			})
		}
		series = append(series, types.ServerNetworkRateSeries{ServerId: serverID, Points: mapped})
	}

	return &types.QueryServerNetworkRateHistoryResponse{
		StartTime:  startTime,
		EndTime:    endTime,
		IntervalMs: interval.Milliseconds(),
		Series:     series,
	}, nil
}

func normalizeServerNetworkHistoryRequest(req *types.QueryServerNetworkRateHistoryRequest, now time.Time) ([]int64, int64, int64, time.Duration, error) {
	if req == nil {
		return nil, 0, 0, 0, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "request is required")
	}
	serverIDs, err := parseServerNetworkHistoryIDs(req.ServerIds)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	intervalName := strings.TrimSpace(req.Interval)
	if intervalName == "" {
		intervalName = "5s"
	}
	interval, ok := serverNetworkHistoryIntervals[intervalName]
	if !ok {
		return nil, 0, 0, 0, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "interval must be one of 5s, 1m, 5m, 1h")
	}

	endTime := req.EndTime
	if endTime == 0 || endTime > now.UnixMilli() {
		endTime = now.UnixMilli()
	}
	startTime := req.StartTime
	if startTime == 0 {
		startTime = endTime - defaultServerNetworkHistoryRange.Milliseconds()
	}
	if startTime < 0 || startTime >= endTime {
		return nil, 0, 0, 0, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "start_time must be before end_time")
	}
	rangeMs := endTime - startTime
	if rangeMs > maxServerNetworkHistoryRange.Milliseconds() {
		return nil, 0, 0, 0, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "history range cannot exceed 72 hours")
	}
	if rangeMs/interval.Milliseconds()+1 > maxServerNetworkHistoryPoints {
		return nil, 0, 0, 0, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "requested interval produces more than 2000 points per server")
	}
	return serverIDs, startTime, endTime, interval, nil
}

func parseServerNetworkHistoryIDs(raw string) ([]int64, error) {
	parts := strings.Split(raw, ",")
	serverIDs := make([]int64, 0, len(parts))
	seen := make(map[int64]struct{}, len(parts))
	for _, part := range parts {
		serverID, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || serverID <= 0 {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "server_ids must contain positive integers")
		}
		if _, ok := seen[serverID]; ok {
			continue
		}
		if len(serverIDs) >= maxServerNetworkHistoryServers {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "server_ids cannot contain more than 50 servers")
		}
		seen[serverID] = struct{}{}
		serverIDs = append(serverIDs, serverID)
	}
	if len(serverIDs) == 0 {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "server_ids is required")
	}
	return serverIDs, nil
}
