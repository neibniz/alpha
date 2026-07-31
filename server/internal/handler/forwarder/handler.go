package forwarder

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/analytics"
	"github.com/neibniz/alpha/server/internal/forwarderapi"
	"github.com/neibniz/alpha/server/internal/forwarderauth"
	"github.com/neibniz/alpha/server/internal/forwarderservice"
	"github.com/neibniz/alpha/server/internal/middleware"
	forwardermodel "github.com/neibniz/alpha/server/internal/model/forwarder"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	maxForwarderEventsPerRequest = 1000
	maxForwarderTextLength       = 1024
	maxRegisterBodyBytes         = 32 << 10
	maxStatusBodyBytes           = 4 << 20
	maxEventsBodyBytes           = 8 << 20
)

var validForwarderEventTypes = map[string]struct{}{
	"connected":   {},
	"closed":      {},
	"dial_failed": {},
	"blocked":     {},
}

func RegisterHandler(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRegisterBodyBytes)
		var req forwarderapi.RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			result.ParamErrorResult(c, err)
			return
		}
		instanceID, err := uuid.Parse(strings.TrimSpace(req.InstanceID))
		if err != nil {
			result.HttpResult(c, nil, xerr.NewErrCodeMsg(xerr.InvalidParams, "valid instance_id is required"))
			return
		}
		bootID, err := uuid.Parse(strings.TrimSpace(req.BootID))
		if err != nil {
			result.HttpResult(c, nil, xerr.NewErrCodeMsg(xerr.InvalidParams, "valid boot_id is required"))
			return
		}
		publicIP, err := registrationPublicIP(req.PublicIP, c.ClientIP())
		if err != nil {
			result.HttpResult(c, nil, xerr.NewErrCodeMsg(xerr.InvalidParams, "valid public_ip is required"))
			return
		}
		startedAt := time.UnixMilli(req.StartedAt).UTC()
		if req.StartedAt <= 0 || startedAt.After(time.Now().Add(time.Minute)) {
			startedAt = time.Now().UTC()
		}
		instance := &forwardermodel.Instance{
			InstanceID: instanceID,
			BootID:     bootID,
			Name:       boundedText(req.Name, 100),
			Hostname:   boundedText(req.Hostname, 255),
			PublicIP:   publicIP,
			Version:    boundedText(req.Version, 64),
			StartedAt:  startedAt,
		}
		if err := svcCtx.ForwarderStore.Register(c.Request.Context(), instance); err != nil {
			logger.WithContext(c.Request.Context()).Errorw("[ForwarderRegister] persist failed", logger.Field("error", err.Error()))
			result.HttpResult(c, nil, xerr.NewErrMsg("failed to register forwarder"))
			return
		}
		token, err := forwarderauth.DeriveToken(svcCtx.ConfigSnapshot().Forwarder.BootstrapSecret, instanceID.String())
		if err != nil {
			result.HttpResult(c, nil, xerr.NewErrMsg("failed to issue forwarder credential"))
			return
		}
		cfg := svcCtx.ConfigSnapshot().Forwarder
		statusInterval := cfg.StatusInterval
		if statusInterval <= 0 {
			statusInterval = 5
		}
		statusTTL := cfg.StatusTTL
		if statusTTL <= statusInterval {
			statusTTL = 30
		}
		result.HttpResult(c, &forwarderapi.RegisterResponse{
			InstanceID:     instanceID.String(),
			Token:          token,
			StatusInterval: statusInterval,
			StatusTTL:      statusTTL,
		}, nil)
	}
}

func ConfigHandler(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		instanceID, instanceUUID, ok := authenticatedInstance(c)
		if !ok {
			c.String(http.StatusUnauthorized, "Unauthorized")
			return
		}
		instance, err := svcCtx.ForwarderStore.FindByInstanceID(c.Request.Context(), instanceUUID)
		if err != nil {
			result.HttpResult(c, nil, xerr.NewErrMsg("forwarder is not registered"))
			return
		}
		config, err := forwarderservice.BuildConfig(c.Request.Context(), svcCtx, instanceID, instance.PublicIP)
		if err != nil {
			result.HttpResult(c, nil, err)
			return
		}
		config.GeneratedAt = time.Now().UnixMilli()
		if err := svcCtx.ForwarderStore.UpdateDesired(c.Request.Context(), instanceUUID, config.Version, config.Hash); err != nil {
			logger.WithContext(c.Request.Context()).Errorw("[ForwarderConfig] update desired state failed", logger.Field("error", err.Error()))
		}
		result.HttpResult(c, config, nil)
	}
}

func StatusHandler(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		instanceID, instanceUUID, ok := authenticatedInstance(c)
		if !ok {
			c.String(http.StatusUnauthorized, "Unauthorized")
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxStatusBodyBytes)
		var req forwarderapi.StatusRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.String(http.StatusBadRequest, "Invalid status payload")
			return
		}
		bootID, err := uuid.Parse(strings.TrimSpace(req.BootID))
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid boot_id")
			return
		}
		if req.Uptime < 0 || req.ActiveConnections < 0 || req.UploadBps < 0 || req.DownloadBps < 0 || len(req.Rules) > 4096 {
			c.String(http.StatusBadRequest, "Invalid metrics")
			return
		}
		rules, err := normalizeRuleStatuses(req.Rules)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid rule metrics")
			return
		}
		state := realtime.ForwarderStatus{
			InstanceID:          instanceID,
			BootID:              bootID.String(),
			Version:             boundedText(req.Version, 64),
			Uptime:              req.Uptime,
			ActiveConnections:   req.ActiveConnections,
			AcceptedConnections: req.AcceptedConnections,
			FailedConnections:   req.FailedConnections,
			BlockedConnections:  req.BlockedConnections,
			UploadBps:           req.UploadBps,
			DownloadBps:         req.DownloadBps,
			UploadBytes:         req.UploadBytes,
			DownloadBytes:       req.DownloadBytes,
			DesiredVersion:      boundedText(req.DesiredVersion, 64),
			DesiredHash:         boundedText(req.DesiredHash, 128),
			AppliedVersion:      boundedText(req.AppliedVersion, 64),
			AppliedHash:         boundedText(req.AppliedHash, 128),
			ApplyStatus:         boundedText(req.ApplyStatus, 32),
			ApplyError:          boundedText(req.ApplyError, maxForwarderTextLength),
			Rules:               rules,
		}
		state = svcCtx.ForwarderRealtime.Update(state)
		if payload, err := json.Marshal(state); err == nil {
			statusTTL := time.Duration(svcCtx.ConfigSnapshot().Forwarder.StatusTTL) * time.Second
			if statusTTL <= 0 {
				statusTTL = 30 * time.Second
			}
			_ = svcCtx.Redis.Set(c.Request.Context(), "forwarder:status:"+instanceID, payload, statusTTL).Err()
		}
		if err := svcCtx.ForwarderStore.TouchRuntime(
			c.Request.Context(), instanceUUID, bootID, state.Version,
			state.AppliedVersion, state.AppliedHash, state.ApplyStatus, state.ApplyError,
		); err != nil {
			logger.WithContext(c.Request.Context()).Errorw("[ForwarderStatus] persist failed", logger.Field("error", err.Error()))
		}
		c.Status(http.StatusNoContent)
	}
}

func EventsHandler(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		instanceID, instanceUUID, ok := authenticatedInstance(c)
		if !ok {
			c.String(http.StatusUnauthorized, "Unauthorized")
			return
		}
		if !svcCtx.Analytics.Available() {
			c.String(http.StatusServiceUnavailable, "Analytics unavailable")
			return
		}
		payload, err := readEventsRequestBody(c.Request.Body, c.GetHeader("Content-Encoding"))
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid event payload")
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(payload))
		var req forwarderapi.ConnectionEventsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.String(http.StatusBadRequest, "Invalid event payload")
			return
		}
		if len(req.Events) == 0 {
			c.Status(http.StatusNoContent)
			return
		}
		batchLimit := svcCtx.ConfigSnapshot().Forwarder.EventBatchLimit
		if batchLimit <= 0 || batchLimit > maxForwarderEventsPerRequest {
			batchLimit = maxForwarderEventsPerRequest
		}
		if len(req.Events) > batchLimit {
			c.String(http.StatusRequestEntityTooLarge, "Too many events")
			return
		}
		events := make([]analytics.ForwardConnectionEvent, 0, len(req.Events))
		for _, raw := range req.Events {
			event, err := parseConnectionEvent(instanceUUID, raw)
			if err != nil {
				c.String(http.StatusBadRequest, err.Error())
				return
			}
			events = append(events, event)
		}
		if err := svcCtx.Analytics.InsertForwardConnectionEvents(c.Request.Context(), events); err != nil {
			logger.WithContext(c.Request.Context()).Errorw(
				"[ForwarderEvents] insert failed",
				logger.Field("instance_id", instanceID),
				logger.Field("events", len(events)),
				logger.Field("error", err.Error()),
			)
			c.String(http.StatusServiceUnavailable, "Failed to store events")
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func registrationPublicIP(reported, observed string) (string, error) {
	reportedIP := net.ParseIP(strings.TrimSpace(reported))
	if reportedIP == nil {
		return "", errors.New("invalid reported public IP")
	}
	observedIP := net.ParseIP(strings.TrimSpace(observed))
	if observedIP != nil && observedIP.IsGlobalUnicast() && !observedIP.IsPrivate() {
		return observedIP.String(), nil
	}
	return reportedIP.String(), nil
}

func readEventsRequestBody(body io.Reader, contentEncoding string) ([]byte, error) {
	compressed, err := readBodyLimit(body, maxEventsBodyBytes)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(contentEncoding)) {
	case "":
		return compressed, nil
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(compressed))
		if err != nil {
			return nil, fmt.Errorf("invalid gzip payload: %w", err)
		}
		defer reader.Close()
		return readBodyLimit(reader, maxEventsBodyBytes)
	default:
		return nil, errors.New("unsupported content encoding")
	}
}

func readBodyLimit(reader io.Reader, limit int64) ([]byte, error) {
	payload, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > limit {
		return nil, errors.New("payload too large")
	}
	return payload, nil
}

func parseConnectionEvent(instanceID uuid.UUID, raw forwarderapi.ConnectionEvent) (analytics.ForwardConnectionEvent, error) {
	eventID, err := uuid.Parse(strings.TrimSpace(raw.EventID))
	if err != nil {
		return analytics.ForwardConnectionEvent{}, errors.New("invalid event_id")
	}
	connectionID, err := uuid.Parse(strings.TrimSpace(raw.ConnectionID))
	if err != nil {
		return analytics.ForwardConnectionEvent{}, errors.New("invalid connection_id")
	}
	bootID, err := uuid.Parse(strings.TrimSpace(raw.BootID))
	if err != nil {
		return analytics.ForwardConnectionEvent{}, errors.New("invalid boot_id")
	}
	eventType := strings.ToLower(strings.TrimSpace(raw.EventType))
	if _, ok := validForwarderEventTypes[eventType]; !ok {
		return analytics.ForwardConnectionEvent{}, errors.New("invalid event_type")
	}
	eventTime := time.UnixMilli(raw.EventTime).UTC()
	now := time.Now().UTC()
	if raw.EventTime <= 0 || eventTime.After(now.Add(5*time.Minute)) || eventTime.Before(now.Add(-370*24*time.Hour)) ||
		raw.ListenPort == 0 || raw.SourcePort == 0 || raw.ConfiguredTargetPort == 0 ||
		raw.DurationMS < 0 || raw.ServerID < 0 || raw.NodeID < 0 ||
		strings.TrimSpace(raw.ConfiguredTargetHost) == "" || strings.TrimSpace(raw.RuleID) == "" {
		return analytics.ForwardConnectionEvent{}, errors.New("invalid event metadata")
	}
	if net.ParseIP(raw.SourceIP) == nil {
		return analytics.ForwardConnectionEvent{}, errors.New("invalid source_ip")
	}
	for _, candidate := range []struct {
		name  string
		value string
	}{
		{name: "listen_ip", value: raw.ListenIP},
		{name: "resolved_target_ip", value: raw.ResolvedTargetIP},
		{name: "forwarder_outbound_ip", value: raw.ForwarderOutboundIP},
	} {
		if candidate.value != "" && net.ParseIP(candidate.value) == nil {
			return analytics.ForwardConnectionEvent{}, errors.New("invalid " + candidate.name)
		}
	}
	if eventType == "connected" && raw.ConnectedAt <= 0 {
		return analytics.ForwardConnectionEvent{}, errors.New("connected_at is required")
	}
	if eventType == "closed" && (raw.ConnectedAt <= 0 || raw.ClosedAt < raw.ConnectedAt) {
		return analytics.ForwardConnectionEvent{}, errors.New("invalid connection timestamps")
	}
	event := analytics.ForwardConnectionEvent{
		EventID:               eventID,
		ConnectionID:          connectionID,
		InstanceID:            instanceID,
		BootID:                bootID,
		EventTime:             eventTime,
		EventType:             eventType,
		SourceIP:              raw.SourceIP,
		SourcePort:            raw.SourcePort,
		ListenIP:              boundedText(raw.ListenIP, 64),
		ListenPort:            raw.ListenPort,
		ConfiguredTargetHost:  boundedText(raw.ConfiguredTargetHost, 255),
		ConfiguredTargetPort:  raw.ConfiguredTargetPort,
		ResolvedTargetIP:      boundedText(raw.ResolvedTargetIP, 64),
		ResolvedTargetPort:    raw.ResolvedTargetPort,
		ForwarderOutboundIP:   boundedText(raw.ForwarderOutboundIP, 64),
		ForwarderOutboundPort: raw.ForwarderOutboundPort,
		RuleID:                boundedText(raw.RuleID, 128),
		ServerID:              raw.ServerID,
		NodeID:                raw.NodeID,
		Inbound:               boundedText(raw.Inbound, 128),
		ConfigVersion:         boundedText(raw.ConfigVersion, 64),
		DurationMS:            raw.DurationMS,
		UploadBytes:           raw.UploadBytes,
		DownloadBytes:         raw.DownloadBytes,
		CloseInitiator:        boundedText(raw.CloseInitiator, 32),
		CloseReason:           boundedText(raw.CloseReason, 64),
		Error:                 boundedText(raw.Error, maxForwarderTextLength),
	}
	if raw.ConnectedAt > 0 {
		event.ConnectedAt = time.UnixMilli(raw.ConnectedAt).UTC()
	}
	if raw.ClosedAt > 0 {
		event.ClosedAt = time.UnixMilli(raw.ClosedAt).UTC()
	}
	return event, nil
}

func authenticatedInstance(c *gin.Context) (string, uuid.UUID, bool) {
	instanceID, ok := middleware.AuthenticatedForwarderID(c.Request.Context())
	if !ok {
		return "", uuid.Nil, false
	}
	parsed, err := uuid.Parse(instanceID)
	return instanceID, parsed, err == nil
}

func boundedText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func normalizeRuleStatuses(rules []realtime.ForwarderRuleStatus) ([]realtime.ForwarderRuleStatus, error) {
	out := make([]realtime.ForwarderRuleStatus, 0, len(rules))
	seen := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		rule.RuleID = boundedText(rule.RuleID, 128)
		rule.ListenIP = boundedText(rule.ListenIP, 64)
		rule.TargetHost = boundedText(rule.TargetHost, 255)
		rule.NodeName = boundedText(rule.NodeName, 255)
		rule.ServerName = boundedText(rule.ServerName, 255)
		rule.Inbound = boundedText(rule.Inbound, 128)
		rule.Status = boundedText(rule.Status, 32)
		rule.Error = boundedText(rule.Error, maxForwarderTextLength)
		if rule.RuleID == "" || rule.ListenPort == 0 || rule.TargetHost == "" || rule.TargetPort == 0 ||
			rule.NodeID < 0 || rule.ServerID < 0 || rule.ActiveConnections < 0 ||
			rule.UploadBps < 0 || rule.DownloadBps < 0 {
			return nil, errors.New("invalid forwarder rule status")
		}
		if rule.ListenIP != "" && net.ParseIP(rule.ListenIP) == nil {
			return nil, errors.New("invalid forwarder listen IP")
		}
		if _, ok := seen[rule.RuleID]; ok {
			return nil, errors.New("duplicate forwarder rule status")
		}
		seen[rule.RuleID] = struct{}{}
		out = append(out, rule)
	}
	return out, nil
}
