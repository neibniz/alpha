package log

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/neibniz/alpha/server/internal/analytics"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
)

const maxProxyAccessRange = 31 * 24 * time.Hour

type FilterProxyAccessLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type proxyAccessCursor struct {
	Timestamp int64  `json:"timestamp"`
	EventID   string `json:"event_id"`
}

func NewFilterProxyAccessLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilterProxyAccessLogLogic {
	return &FilterProxyAccessLogLogic{Logger: logger.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *FilterProxyAccessLogLogic) FilterProxyAccessLog(req *types.FilterProxyAccessLogRequest) (*types.FilterProxyAccessLogResponse, error) {
	if !l.svcCtx.Analytics.Available() {
		return nil, xerr.NewErrMsg("analytics storage is unavailable")
	}
	filter, err := proxyAccessFilter(req, time.Now())
	if err != nil {
		return nil, xerr.NewErrCodeMsg(xerr.InvalidParams, err.Error())
	}
	page, err := l.svcCtx.Analytics.QueryAccessEvents(l.ctx, filter)
	if err != nil {
		l.Errorw("[FilterProxyAccessLog] query failed", logger.Field("error", err.Error()))
		return nil, xerr.NewErrMsg("failed to query proxy access logs")
	}
	list := make([]types.ProxyAccessLog, 0, len(page.List))
	for _, event := range page.List {
		list = append(list, proxyAccessLogView(event))
	}
	resp := &types.FilterProxyAccessLogResponse{List: list, HasMore: page.HasMore}
	if page.HasMore && len(page.List) > 0 {
		last := page.List[len(page.List)-1]
		resp.NextCursor = encodeProxyAccessCursor(last.EventTime, last.EventID)
	}
	return resp, nil
}

func proxyAccessFilter(req *types.FilterProxyAccessLogRequest, now time.Time) (analytics.AccessFilter, error) {
	if req == nil {
		req = &types.FilterProxyAccessLogRequest{}
	}
	end := time.UnixMilli(now.UnixMilli())
	if req.EndTime > 0 {
		end = time.UnixMilli(req.EndTime)
	}
	start := end.Add(-24 * time.Hour)
	if req.StartTime > 0 {
		start = time.UnixMilli(req.StartTime)
	}
	if !start.Before(end) {
		return analytics.AccessFilter{}, fmt.Errorf("start_time must be before end_time")
	}
	if end.Sub(start) > maxProxyAccessRange {
		return analytics.AccessFilter{}, fmt.Errorf("time range cannot exceed 31 days")
	}
	if req.ServerId < 0 || req.UserId < 0 || req.UserSubscribeId < 0 {
		return analytics.AccessFilter{}, fmt.Errorf("filter IDs cannot be negative")
	}
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status != "" && status != "accepted" && status != "rejected" {
		return analytics.AccessFilter{}, fmt.Errorf("status must be accepted or rejected")
	}
	limit := req.Size
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	sourceIP := strings.TrimSpace(req.SourceIP)
	if sourceIP != "" {
		parsed := net.ParseIP(strings.Trim(sourceIP, "[]"))
		if parsed == nil {
			return analytics.AccessFilter{}, fmt.Errorf("source_ip must be a valid IP address")
		}
		sourceIP = parsed.String()
	}
	destination := strings.TrimSpace(req.Destination)
	if parsed := net.ParseIP(strings.Trim(destination, "[]")); parsed != nil {
		destination = parsed.String()
	}
	filter := analytics.AccessFilter{
		Start:           start,
		End:             end,
		ServerID:        req.ServerId,
		UserID:          req.UserId,
		UserSubscribeID: req.UserSubscribeId,
		SourceIP:        sourceIP,
		Destination:     destination,
		Status:          status,
		Limit:           limit,
	}
	if req.Cursor != "" {
		cursor, err := decodeProxyAccessCursor(req.Cursor)
		if err != nil {
			return analytics.AccessFilter{}, fmt.Errorf("invalid cursor")
		}
		filter.CursorTime = time.UnixMilli(cursor.Timestamp)
		filter.CursorID, err = uuid.Parse(cursor.EventID)
		if err != nil {
			return analytics.AccessFilter{}, fmt.Errorf("invalid cursor")
		}
	}
	return filter, nil
}

func proxyAccessLogView(event analytics.AccessEvent) types.ProxyAccessLog {
	destination := event.DestinationHost
	if destination == "" {
		destination = event.DestinationIP
	}
	return types.ProxyAccessLog{
		EventId:         event.EventID.String(),
		Timestamp:       event.EventTime.UnixMilli(),
		ServerId:        event.ServerID,
		UserId:          event.UserID,
		UserSubscribeId: event.UserSubscribeID,
		SourceIP:        privacy.MaskIP(event.SourceIP),
		SourcePort:      event.SourcePort,
		Destination:     maskProxyDestination(destination),
		DestinationPort: event.DestinationPort,
		Network:         event.Network,
		Status:          event.Status,
		OutboundTag:     event.OutboundTag,
		Reason:          privacy.MaskIdentifier(event.Reason),
	}
}

func maskProxyDestination(value string) string {
	if net.ParseIP(value) != nil {
		return privacy.MaskIP(value)
	}
	return privacy.MaskIdentifier(value)
}

func encodeProxyAccessCursor(eventTime time.Time, eventID uuid.UUID) string {
	payload, _ := json.Marshal(proxyAccessCursor{Timestamp: eventTime.UnixMilli(), EventID: eventID.String()})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeProxyAccessCursor(value string) (proxyAccessCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return proxyAccessCursor{}, err
	}
	var cursor proxyAccessCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return proxyAccessCursor{}, err
	}
	if cursor.Timestamp <= 0 || cursor.EventID == "" {
		return proxyAccessCursor{}, fmt.Errorf("empty cursor")
	}
	return cursor, nil
}
