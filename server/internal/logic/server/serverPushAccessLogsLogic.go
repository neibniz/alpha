package server

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/neibniz/alpha/server/internal/analytics"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
)

const (
	maxAccessBatchSize  = 2000
	maxAccessClockSkew  = 5 * time.Minute
	maxAccessReasonSize = 512
)

type ServerPushAccessLogsLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type accessSubscription struct {
	UserID      int64
	DeviceLimit int64
}

func NewServerPushAccessLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ServerPushAccessLogsLogic {
	return &ServerPushAccessLogsLogic{Logger: logger.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ServerPushAccessLogsLogic) ServerPushAccessLogs(req *types.ServerPushAccessLogsRequest) (*types.ServerPushAccessLogsResponse, error) {
	result := &types.ServerPushAccessLogsResponse{}
	if req == nil {
		return nil, xerr.NewErrCodeMsg(xerr.InvalidParams, "access log request is required")
	}
	if len(req.Events) == 0 {
		return result, nil
	}
	if len(req.Events) > maxAccessBatchSize {
		return nil, xerr.NewErrCodeMsg(xerr.InvalidParams, fmt.Sprintf("access event batch exceeds %d rows", maxAccessBatchSize))
	}
	if !l.svcCtx.Analytics.Available() {
		return nil, xerr.NewErrMsg("analytics storage is unavailable")
	}
	exists, err := l.svcCtx.NodeModel.ServerExists(l.ctx, req.ServerId)
	if err != nil || !exists {
		return nil, xerr.NewErrCodeMsg(xerr.InvalidParams, "server not found")
	}

	subscriptions, err := l.queryAccessEventSubscriptions(req.Events)
	if err != nil {
		return nil, err
	}
	userIDs := make(map[int64]int64, len(subscriptions))
	for id, subscription := range subscriptions {
		userIDs[id] = subscription.UserID
	}
	now := time.Now()
	events, rejected := normalizeAccessEvents(req.ServerId, req.Events, userIDs, now)
	result.Rejected = rejected
	if len(events) == 0 {
		if rejected > 0 {
			l.Infow("[ServerPushAccessLogs] rejected invalid access events",
				logger.Field("server_id", req.ServerId),
				logger.Field("rejected_count", rejected))
		}
		return result, nil
	}
	if err := l.svcCtx.Analytics.InsertAccessEvents(l.ctx, events); err != nil {
		l.Errorw("[ServerPushAccessLogs] insert access events failed",
			logger.Field("error", err.Error()),
			logger.Field("server_id", req.ServerId),
			logger.Field("event_count", len(events)))
		return nil, xerr.NewErrMsg("failed to store access events")
	}
	result.Accepted = len(events)
	result.IPLimitDecisions = l.evaluateIPLimits(events, subscriptions, now)
	if rejected > 0 {
		l.Infow("[ServerPushAccessLogs] partially rejected invalid access events",
			logger.Field("server_id", req.ServerId),
			logger.Field("accepted_count", result.Accepted),
			logger.Field("rejected_count", result.Rejected))
	}
	return result, nil
}

func normalizeAccessEvents(serverID int64, items []types.XrayAccessEvent, users map[int64]int64, now time.Time) ([]analytics.AccessEvent, int) {
	events := make([]analytics.AccessEvent, 0, len(items))
	rejected := 0
	for _, item := range items {
		event, err := normalizeAccessEvent(serverID, item, users, now)
		if err != nil {
			rejected++
			continue
		}
		events = append(events, event)
	}
	return events, rejected
}

func (l *ServerPushAccessLogsLogic) queryAccessEventSubscriptions(items []types.XrayAccessEvent) (map[int64]accessSubscription, error) {
	ids := make([]int64, 0, len(items))
	seen := make(map[int64]struct{}, len(items))
	for _, item := range items {
		if item.UserSubscribeId <= 0 {
			continue
		}
		if _, ok := seen[item.UserSubscribeId]; ok {
			continue
		}
		seen[item.UserSubscribeId] = struct{}{}
		ids = append(ids, item.UserSubscribeId)
	}
	if len(ids) == 0 {
		return map[int64]accessSubscription{}, nil
	}
	var rows []struct {
		Id              int64
		UserId          int64
		ProductSnapshot string
		DeviceLimit     int64
	}
	if err := l.svcCtx.DB.WithContext(l.ctx).Model(&user.Subscribe{}).
		Select("user_subscribe.id, user_subscribe.user_id, user_subscribe.product_snapshot, COALESCE(subscribe.device_limit, 0) AS device_limit").
		Joins("LEFT JOIN subscribe ON subscribe.id = user_subscribe.subscribe_id").
		Where("user_subscribe.id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[int64]accessSubscription, len(rows))
	for _, row := range rows {
		result[row.Id] = accessSubscription{
			UserID:      row.UserId,
			DeviceLimit: effectiveDeviceLimit(row.ProductSnapshot, row.DeviceLimit),
		}
	}
	return result, nil
}

func effectiveDeviceLimit(productSnapshot string, currentLimit int64) int64 {
	_, snapshotLimit, ok := ordersnapshot.SubscribeLimits(productSnapshot)
	if ok {
		if snapshotLimit < 0 {
			return 0
		}
		return snapshotLimit
	}
	if currentLimit < 0 {
		return 0
	}
	return currentLimit
}

func (l *ServerPushAccessLogsLogic) evaluateIPLimits(events []analytics.AccessEvent, subscriptions map[int64]accessSubscription, now time.Time) []types.XrayIPLimitDecision {
	if l.svcCtx.AccessIPLimiter == nil {
		return nil
	}
	type candidate struct {
		eventID         string
		userSubscribeID int64
		sourceNetwork   string
		limit           int64
	}
	type candidateKey struct {
		userSubscribeID int64
		sourceNetwork   string
	}
	candidates := make([]candidate, 0, len(events))
	seen := make(map[candidateKey]struct{}, len(events))
	oldestActive := now.Add(-l.svcCtx.AccessIPLimiter.ActiveTTL())
	for _, event := range events {
		if event.Status != "accepted" || event.UserSubscribeID <= 0 || event.EventTime.Before(oldestActive) {
			continue
		}
		subscription, ok := subscriptions[event.UserSubscribeID]
		if !ok {
			continue
		}
		sourceNetwork, ok := privacy.NormalizeIPNetwork(event.SourceIP)
		if !ok {
			continue
		}
		key := candidateKey{userSubscribeID: event.UserSubscribeID, sourceNetwork: sourceNetwork}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		candidates = append(candidates, candidate{
			eventID:         event.EventID.String(),
			userSubscribeID: event.UserSubscribeID,
			sourceNetwork:   sourceNetwork,
			limit:           subscription.DeviceLimit,
		})
	}

	decisions := make([]types.XrayIPLimitDecision, 0, len(candidates))
	for _, item := range candidates {
		decision, err := l.svcCtx.AccessIPLimiter.Evaluate(l.ctx, item.userSubscribeID, item.sourceNetwork, item.limit)
		if err != nil {
			l.Errorw("[ServerPushAccessLogs] evaluate active IP limit failed",
				logger.Field("error", err.Error()),
				logger.Field("user_subscribe_id", item.userSubscribeID))
			continue
		}
		decisions = append(decisions, types.XrayIPLimitDecision{
			EventId:         item.eventID,
			UserSubscribeId: item.userSubscribeID,
			SourceNetwork:   item.sourceNetwork,
			Blocked:         decision.Blocked,
			ExpiresAt:       decision.ExpiresAt.UnixMilli(),
			Active:          decision.Active,
			Limit:           decision.Limit,
		})
	}
	return decisions
}

func normalizeAccessEvent(serverID int64, item types.XrayAccessEvent, users map[int64]int64, now time.Time) (analytics.AccessEvent, error) {
	eventID, err := uuid.Parse(item.EventId)
	if err != nil || eventID == uuid.Nil {
		return analytics.AccessEvent{}, fmt.Errorf("invalid access event id")
	}
	eventTime := time.UnixMilli(item.Timestamp)
	if item.Timestamp <= 0 || eventTime.After(now.Add(maxAccessClockSkew)) {
		return analytics.AccessEvent{}, fmt.Errorf("invalid access event timestamp")
	}
	status := strings.ToLower(strings.TrimSpace(item.Status))
	if status != "accepted" && status != "rejected" {
		return analytics.AccessEvent{}, fmt.Errorf("invalid access event status")
	}
	network := strings.ToLower(strings.TrimSpace(item.Network))
	if network != "" && network != "tcp" && network != "udp" {
		return analytics.AccessEvent{}, fmt.Errorf("invalid access event network")
	}
	sourceIP, err := normalizeAccessIP(item.SourceIP)
	if err != nil || sourceIP == "" {
		return analytics.AccessEvent{}, fmt.Errorf("invalid access event source IP")
	}
	destinationIP, err := normalizeAccessIP(item.DestinationIP)
	if err != nil {
		return analytics.AccessEvent{}, fmt.Errorf("invalid access event destination IP")
	}
	destinationHost := strings.TrimSpace(item.DestinationHost)
	if parsed := net.ParseIP(strings.Trim(destinationHost, "[]")); parsed != nil {
		parsedIP := parsed.String()
		if destinationIP != "" && destinationIP != parsedIP {
			return analytics.AccessEvent{}, fmt.Errorf("conflicting access event destination IP")
		}
		destinationIP = parsedIP
		destinationHost = ""
	}
	userID := users[item.UserSubscribeId]
	userSubscribeID := item.UserSubscribeId
	if userID == 0 {
		userSubscribeID = 0
	}
	return analytics.AccessEvent{
		EventID:         eventID,
		EventTime:       eventTime,
		ServerID:        serverID,
		UserID:          userID,
		UserSubscribeID: userSubscribeID,
		SourceIP:        sourceIP,
		SourcePort:      item.SourcePort,
		DestinationHost: truncateAccessValue(destinationHost, 512),
		DestinationIP:   destinationIP,
		DestinationPort: item.DestinationPort,
		Network:         network,
		Status:          status,
		OutboundTag:     truncateAccessValue(item.OutboundTag, 128),
		Reason:          truncateAccessValue(item.Reason, maxAccessReasonSize),
		ParserVersion:   item.ParserVersion,
	}, nil
}

func normalizeAccessIP(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	ip := net.ParseIP(strings.Trim(value, "[]"))
	if ip == nil {
		return "", fmt.Errorf("invalid IP address")
	}
	return ip.String(), nil
}

func truncateAccessValue(value string, limit int) string {
	value = strings.ToValidUTF8(strings.TrimSpace(value), "")
	if len(value) <= limit {
		return value
	}
	for limit > 0 && !utf8.RuneStart(value[limit]) {
		limit--
	}
	return value[:limit]
}
