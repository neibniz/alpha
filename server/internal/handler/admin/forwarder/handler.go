package forwarder

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/analytics"
	"github.com/neibniz/alpha/server/internal/forwarderapi"
	"github.com/neibniz/alpha/server/internal/forwarderservice"
	forwardermodel "github.com/neibniz/alpha/server/internal/model/forwarder"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxConnectionQueryRange = 31 * 24 * time.Hour

type connectionCursor struct {
	Timestamp int64  `json:"timestamp"`
	EventID   string `json:"event_id"`
}

func ListHandler(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := svcCtx.ForwarderStore.List(c.Request.Context(), strings.TrimSpace(c.Query("search")))
		if err != nil {
			result.HttpResult(c, nil, xerr.NewErrMsg("failed to query forwarders"))
			return
		}
		cachedStatuses := loadCachedStatuses(c.Request.Context(), svcCtx, rows)
		list := make([]forwarderapi.InstanceView, 0, len(rows))
		for _, row := range rows {
			status, ok := svcCtx.ForwarderRealtime.Get(row.InstanceID.String())
			if !ok {
				status, ok = cachedStatuses[row.InstanceID.String()]
			}
			if !ok {
				status.InstanceID = row.InstanceID.String()
				status.BootID = row.BootID.String()
				status.Version = row.Version
				status.Online = false
				status.LastSeen = row.LastSeenAt.UnixMilli()
				status.AppliedVersion = row.AppliedVersion
				status.AppliedHash = row.AppliedHash
				status.ApplyStatus = row.ApplyStatus
				status.ApplyError = row.ApplyError
				status.Rules = []realtime.ForwarderRuleStatus{}
			}
			list = append(list, forwarderapi.InstanceView{
				ID:             row.ID,
				InstanceID:     row.InstanceID.String(),
				BootID:         row.BootID.String(),
				Name:           row.Name,
				Hostname:       row.Hostname,
				PublicIP:       row.PublicIP,
				Version:        row.Version,
				StartedAt:      row.StartedAt.UnixMilli(),
				FirstSeenAt:    row.FirstSeenAt.UnixMilli(),
				LastSeenAt:     row.LastSeenAt.UnixMilli(),
				DesiredVersion: row.DesiredVersion,
				DesiredHash:    row.DesiredHash,
				AppliedVersion: row.AppliedVersion,
				AppliedHash:    row.AppliedHash,
				ApplyStatus:    row.ApplyStatus,
				ApplyError:     row.ApplyError,
				Status:         status,
			})
		}
		result.HttpResult(c, gin.H{"list": list, "total": len(list)}, nil)
	}
}

func loadCachedStatuses(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
	rows []forwardermodel.Instance,
) map[string]realtime.ForwarderStatus {
	statuses := make(map[string]realtime.ForwarderStatus)
	if len(rows) == 0 {
		return statuses
	}
	keys := make([]string, 0, len(rows))
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		id := row.InstanceID.String()
		if _, ok := svcCtx.ForwarderRealtime.Get(id); ok {
			continue
		}
		ids = append(ids, id)
		keys = append(keys, "forwarder:status:"+id)
	}
	if len(keys) == 0 {
		return statuses
	}
	values, err := svcCtx.Redis.MGet(ctx, keys...).Result()
	if err != nil {
		return statuses
	}
	ttl := time.Duration(svcCtx.ConfigSnapshot().Forwarder.StatusTTL) * time.Second
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	for index, value := range values {
		payload, ok := value.(string)
		if !ok || payload == "" {
			continue
		}
		var status realtime.ForwarderStatus
		if json.Unmarshal([]byte(payload), &status) != nil ||
			time.Since(time.UnixMilli(status.LastSeen)) > ttl {
			continue
		}
		status.Online = true
		if status.Rules == nil {
			status.Rules = []realtime.ForwarderRuleStatus{}
		}
		statuses[ids[index]] = status
	}
	return statuses
}

func ConfigHandler(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		instanceID, err := uuid.Parse(strings.TrimSpace(c.Query("instance_id")))
		if err != nil {
			result.HttpResult(c, nil, xerr.NewErrCodeMsg(xerr.InvalidParams, "valid instance_id is required"))
			return
		}
		instance, err := svcCtx.ForwarderStore.FindByInstanceID(c.Request.Context(), instanceID)
		if err != nil {
			result.HttpResult(c, nil, xerr.NewErrMsg("forwarder was not found"))
			return
		}
		config, err := forwarderservice.BuildConfig(c.Request.Context(), svcCtx, instanceID.String(), instance.PublicIP)
		if err != nil {
			result.HttpResult(c, nil, err)
			return
		}
		config.GeneratedAt = time.Now().UnixMilli()
		result.HttpResult(c, config, nil)
	}
}

func ConnectionsHandler(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !svcCtx.Analytics.Available() {
			result.HttpResult(c, nil, xerr.NewErrMsg("analytics storage is unavailable"))
			return
		}
		filter, err := connectionFilter(c)
		if err != nil {
			result.HttpResult(c, nil, xerr.NewErrCodeMsg(xerr.InvalidParams, err.Error()))
			return
		}
		page, err := svcCtx.Analytics.QueryForwardConnectionEvents(c.Request.Context(), filter)
		if err != nil {
			logger.WithContext(c.Request.Context()).Errorw("[ForwarderConnections] query failed", logger.Field("error", err.Error()))
			result.HttpResult(c, nil, xerr.NewErrMsg("failed to query forwarder connections"))
			return
		}
		list := make([]forwarderapi.ConnectionEventView, 0, len(page.List))
		for _, event := range page.List {
			list = append(list, connectionView(event))
		}
		response := forwarderapi.ConnectionPage{List: list, HasMore: page.HasMore}
		if page.HasMore && len(page.List) > 0 {
			last := page.List[len(page.List)-1]
			response.NextCursor = encodeCursor(last.EventTime, last.EventID)
		}
		result.HttpResult(c, response, nil)
	}
}

func connectionFilter(c *gin.Context) (analytics.ForwardConnectionFilter, error) {
	now := time.Now()
	end := queryTime(c.Query("end_time"), now)
	start := queryTime(c.Query("start_time"), end.Add(-24*time.Hour))
	if !start.Before(end) {
		return analytics.ForwardConnectionFilter{}, xerr.NewErrMsg("start_time must be before end_time")
	}
	if end.Sub(start) > maxConnectionQueryRange {
		return analytics.ForwardConnectionFilter{}, xerr.NewErrMsg("time range cannot exceed 31 days")
	}
	filter := analytics.ForwardConnectionFilter{
		Start:       start,
		End:         end,
		SourceIP:    strings.TrimSpace(c.Query("source_ip")),
		Destination: strings.TrimSpace(c.Query("destination")),
		EventType:   strings.ToLower(strings.TrimSpace(c.Query("event_type"))),
		Limit:       queryInt(c.Query("size"), 50),
	}
	if raw := strings.TrimSpace(c.Query("instance_id")); raw != "" {
		value, err := uuid.Parse(raw)
		if err != nil {
			return filter, xerr.NewErrMsg("invalid instance_id")
		}
		filter.InstanceID = value
	}
	if raw := strings.TrimSpace(c.Query("connection_id")); raw != "" {
		value, err := uuid.Parse(raw)
		if err != nil {
			return filter, xerr.NewErrMsg("invalid connection_id")
		}
		filter.ConnectionID = value
	}
	if raw := strings.TrimSpace(c.Query("listen_port")); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 16)
		if err != nil || value == 0 {
			return filter, xerr.NewErrMsg("invalid listen_port")
		}
		filter.ListenPort = uint16(value)
	}
	if raw := strings.TrimSpace(c.Query("cursor")); raw != "" {
		cursor, err := decodeCursor(raw)
		if err != nil {
			return filter, xerr.NewErrMsg("invalid cursor")
		}
		filter.CursorTime = time.UnixMilli(cursor.Timestamp)
		filter.CursorID, err = uuid.Parse(cursor.EventID)
		if err != nil {
			return filter, xerr.NewErrMsg("invalid cursor")
		}
	}
	return filter, nil
}

func connectionView(event analytics.ForwardConnectionEvent) forwarderapi.ConnectionEventView {
	view := forwarderapi.ConnectionEventView{
		EventID:               event.EventID.String(),
		ConnectionID:          event.ConnectionID.String(),
		InstanceID:            event.InstanceID.String(),
		BootID:                event.BootID.String(),
		EventTime:             event.EventTime.UnixMilli(),
		EventType:             event.EventType,
		SourceIP:              event.SourceIP,
		SourcePort:            event.SourcePort,
		ListenIP:              event.ListenIP,
		ListenPort:            event.ListenPort,
		ConfiguredTargetHost:  event.ConfiguredTargetHost,
		ConfiguredTargetPort:  event.ConfiguredTargetPort,
		ResolvedTargetIP:      event.ResolvedTargetIP,
		ResolvedTargetPort:    event.ResolvedTargetPort,
		ForwarderOutboundIP:   event.ForwarderOutboundIP,
		ForwarderOutboundPort: event.ForwarderOutboundPort,
		RuleID:                event.RuleID,
		ServerID:              event.ServerID,
		NodeID:                event.NodeID,
		Inbound:               event.Inbound,
		ConfigVersion:         event.ConfigVersion,
		DurationMS:            event.DurationMS,
		UploadBytes:           event.UploadBytes,
		DownloadBytes:         event.DownloadBytes,
		CloseInitiator:        event.CloseInitiator,
		CloseReason:           event.CloseReason,
		Error:                 event.Error,
	}
	if !event.ConnectedAt.IsZero() {
		view.ConnectedAt = event.ConnectedAt.UnixMilli()
	}
	if !event.ClosedAt.IsZero() {
		view.ClosedAt = event.ClosedAt.UnixMilli()
	}
	return view
}

func queryTime(raw string, fallback time.Time) time.Time {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return time.UnixMilli(value)
}

func queryInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 || value > 200 {
		return fallback
	}
	return value
}

func encodeCursor(eventTime time.Time, eventID uuid.UUID) string {
	payload, _ := json.Marshal(connectionCursor{Timestamp: eventTime.UnixMilli(), EventID: eventID.String()})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeCursor(raw string) (connectionCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return connectionCursor{}, err
	}
	var cursor connectionCursor
	err = json.Unmarshal(payload, &cursor)
	return cursor, err
}
