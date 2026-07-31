package console

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/middleware"
	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/model/membership"
	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/model/order"
	trafficModel "github.com/neibniz/alpha/server/internal/model/traffic"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/jwt"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var dashboardRealtimeUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const (
	orderTypeSubscribe    = 1
	orderTypeRenewal      = 2
	orderTypeResetTraffic = 3
	orderTypeRecharge     = 4
	orderTypeMembership   = 5

	orderStatusPaid     = 2
	orderStatusFinished = 5
)

var paidOrderStatuses = []int64{orderStatusPaid, orderStatusFinished}
var businessOrderTypes = []int64{orderTypeSubscribe, orderTypeRenewal, orderTypeResetTraffic, orderTypeMembership}

const (
	dashboardOnlineSeriesIndexKey   = "dashboard:online_users:samples:index"
	dashboardOnlineSeriesValuesKey  = "dashboard:online_users:samples:values"
	dashboardTrafficSeriesIndexKey  = "dashboard:traffic:today:samples:index"
	dashboardTrafficSeriesValuesKey = "dashboard:traffic:today:samples:values"
	dashboardNetworkSeriesIndexKey  = "dashboard:network_rate:samples:index"
	dashboardNetworkSeriesValuesKey = "dashboard:network_rate:samples:values"
	dashboardSampleTTL              = 8 * 24 * time.Hour
	dashboardNetworkSampleTTL       = 72 * time.Hour
	dashboardActivityLimit          = 50
)

type dashboardRealtimePayload struct {
	Type             string                          `json:"type"`
	Full             bool                            `json:"full"`
	Seq              uint64                          `json:"seq,omitempty"`
	Reason           string                          `json:"reason,omitempty"`
	UpdatedAt        int64                           `json:"updated_at"`
	OnlineUsers      int64                           `json:"online_users"`
	OnlineUserSeries dashboardRealtimeOnlineSeries   `json:"online_user_series"`
	PendingTickets   int64                           `json:"pending_tickets"`
	Servers          dashboardRealtimeServerSummary  `json:"servers"`
	Network          dashboardRealtimeNetwork        `json:"network"`
	NetworkSeries    []dashboardRealtimeNetworkPoint `json:"network_series"`
	Connections      dashboardRealtimeConnections    `json:"connections"`
	Resources        dashboardRealtimeResources      `json:"resources"`
	Business         dashboardRealtimeBusiness       `json:"business"`
	Traffic          dashboardRealtimeTraffic        `json:"traffic"`
	Alerts           []dashboardRealtimeAlert        `json:"alerts"`
	Activities       []dashboardRealtimeActivity     `json:"activities"`
}

type dashboardRealtimeDeltaPayload struct {
	Type           string                          `json:"type"`
	Full           bool                            `json:"full"`
	Seq            uint64                          `json:"seq,omitempty"`
	Reason         string                          `json:"reason,omitempty"`
	UpdatedAt      int64                           `json:"updated_at"`
	OnlineUsers    int64                           `json:"online_users"`
	PendingTickets int64                           `json:"pending_tickets"`
	Servers        dashboardRealtimeServerSummary  `json:"servers"`
	Network        dashboardRealtimeNetwork        `json:"network"`
	NetworkSeries  []dashboardRealtimeNetworkPoint `json:"network_series,omitempty"`
	Connections    dashboardRealtimeConnections    `json:"connections"`
	Resources      dashboardRealtimeResources      `json:"resources"`
	Business       dashboardRealtimeBusiness       `json:"business"`
	Traffic        dashboardRealtimeTrafficDelta   `json:"traffic"`
	Alerts         []dashboardRealtimeAlert        `json:"alerts"`
	Activities     []dashboardRealtimeActivity     `json:"activities,omitempty"`
}

type dashboardRealtimeServerSummary struct {
	Total         int64 `json:"total"`
	Online        int64 `json:"online"`
	Offline       int64 `json:"offline"`
	XrayRunning   int64 `json:"xray_running"`
	XrayStopped   int64 `json:"xray_stopped"`
	ConfigOK      int64 `json:"config_ok"`
	ConfigPending int64 `json:"config_pending"`
	ConfigFailed  int64 `json:"config_failed"`
	StatsError    int64 `json:"stats_error"`
	Reported      int64 `json:"reported"`
	NotReported   int64 `json:"not_reported"`
}

type dashboardRealtimeNetwork struct {
	SystemRxBps int64 `json:"system_rx_bps"`
	SystemTxBps int64 `json:"system_tx_bps"`
	XrayRxBps   int64 `json:"xray_rx_bps"`
	XrayTxBps   int64 `json:"xray_tx_bps"`
}

type dashboardRealtimeNetworkPoint struct {
	Timestamp   int64 `json:"timestamp"`
	SystemRxBps int64 `json:"system_rx_bps"`
	SystemTxBps int64 `json:"system_tx_bps"`
	XrayRxBps   int64 `json:"xray_rx_bps"`
	XrayTxBps   int64 `json:"xray_tx_bps"`
}

type dashboardRealtimeConnections struct {
	SystemInbound  int64 `json:"system_inbound"`
	SystemOutbound int64 `json:"system_outbound"`
	XrayInbound    int64 `json:"xray_inbound"`
	XrayOutbound   int64 `json:"xray_outbound"`
}

type dashboardRealtimeResources struct {
	AvgCPU  float64 `json:"avg_cpu"`
	AvgMem  float64 `json:"avg_mem"`
	AvgDisk float64 `json:"avg_disk"`
	MaxCPU  float64 `json:"max_cpu"`
	MaxMem  float64 `json:"max_mem"`
	MaxDisk float64 `json:"max_disk"`
}

type dashboardRealtimeBusiness struct {
	RevenueToday         int64 `json:"revenue_today"`
	NewOrderAmount       int64 `json:"new_order_amount"`
	RenewalOrderAmount   int64 `json:"renewal_order_amount"`
	MembershipAmount     int64 `json:"membership_amount"`
	ResetTrafficAmount   int64 `json:"reset_traffic_amount"`
	RegisterToday        int64 `json:"register_today"`
	NewOrderCount        int64 `json:"new_order_count"`
	RenewalOrderCount    int64 `json:"renewal_order_count"`
	MembershipOrderCount int64 `json:"membership_order_count"`
}

type dashboardRealtimeTraffic struct {
	TodayUpload          int64                            `json:"today_upload"`
	TodayDownload        int64                            `json:"today_download"`
	MonthlyUpload        int64                            `json:"monthly_upload"`
	MonthlyDownload      int64                            `json:"monthly_download"`
	TodaySeries          []dashboardRealtimeTrafficPoint  `json:"today_series"`
	ServerRankingToday   []dashboardRealtimeServerTraffic `json:"server_ranking_today"`
	UserRankingToday     []dashboardRealtimeUserTraffic   `json:"user_ranking_today"`
	ServerRankingMonthly []dashboardRealtimeServerTraffic `json:"server_ranking_monthly"`
	UserRankingMonthly   []dashboardRealtimeUserTraffic   `json:"user_ranking_monthly"`
}

type dashboardRealtimeTrafficDelta struct {
	TodayUpload     int64 `json:"today_upload"`
	TodayDownload   int64 `json:"today_download"`
	MonthlyUpload   int64 `json:"monthly_upload"`
	MonthlyDownload int64 `json:"monthly_download"`
}

type dashboardTrafficTotalsRow struct {
	TodayUpload     int64
	TodayDownload   int64
	MonthlyUpload   int64
	MonthlyDownload int64
}

type dashboardRealtimeOnlineSeries struct {
	LastDay   []dashboardRealtimeValuePoint `json:"last_day"`
	Last7Days []dashboardRealtimeValuePoint `json:"last_7_days"`
}

type dashboardRealtimeValuePoint struct {
	Timestamp int64 `json:"timestamp"`
	Value     int64 `json:"value"`
}

type dashboardRealtimeTrafficPoint struct {
	Timestamp int64 `json:"timestamp"`
	Upload    int64 `json:"upload"`
	Download  int64 `json:"download"`
	Total     int64 `json:"total"`
}

type dashboardRealtimeServerTraffic struct {
	ServerID int64  `json:"server_id"`
	Name     string `json:"name"`
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
}

type dashboardRealtimeUserTraffic struct {
	SID      int64 `json:"sid"`
	Upload   int64 `json:"upload"`
	Download int64 `json:"download"`
}

type dashboardRealtimeAlert struct {
	Level    string `json:"level"`
	ServerID int64  `json:"server_id,omitempty"`
	Title    string `json:"title"`
	Message  string `json:"message,omitempty"`
}

type dashboardRealtimeActivity struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Subject   string `json:"subject,omitempty"`
	Detail    string `json:"detail,omitempty"`
	IP        string `json:"ip,omitempty"`
	Location  string `json:"location,omitempty"`
	Status    string `json:"status,omitempty"`
	Amount    int64  `json:"amount,omitempty"`
	UserID    int64  `json:"user_id,omitempty"`
	CreatedAt int64  `json:"created_at"`
}

type dashboardActivityNetworkInfo struct {
	IP        string
	Location  string
	CreatedAt time.Time
}

type dashboardSystemActivityLog struct {
	Id           int64
	Type         uint8
	UserID       int64
	Method       string
	AuthMethod   string
	Identifier   string
	ClientIP     string
	CountryShort string
	Region       string
	Location     string
	Success      bool
	Timestamp    int64
	CreatedAt    time.Time
}

type dashboardOrderActivity struct {
	Id           int64
	UserId       int64
	Type         uint8
	Status       uint8
	Amount       int64
	SubscribeId  int64
	ProductName  string
	ClientIP     string
	CountryShort string
	Region       string
	Location     string
	CreatedAt    time.Time
}

var dashboardSystemActivitySelectColumns = []string{
	"id",
	"type",
	"object_id AS user_id",
	"country_short",
	"region",
	"location",
	"method",
	"method AS auth_method",
	"identifier",
	"ip AS client_ip",
	"success",
	"event_at AS timestamp",
	"created_at",
}

var dashboardOrderActivitySelectColumns = []string{
	`"order".id`,
	`"order".user_id`,
	`"order".type`,
	`"order".status`,
	`"order".amount`,
	`"order".subscribe_id`,
	`COALESCE(NULLIF("order".product_name, ''), subscribe.name, '') AS product_name`,
	`"order".client_ip`,
	`"order".country_short`,
	`"order".region`,
	`"order".location`,
	`"order".created_at`,
}

var dashboardSubscribeActivitySelectColumns = []string{
	"id",
	"user_id",
	"user_subscribe_id",
	"client_ip",
	"country_short",
	"region",
	"location",
	"success",
	"created_at",
}

func DashboardRealtimeHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		if !validateDashboardRealtimeToken(c, svcCtx) {
			c.String(http.StatusForbidden, "Forbidden")
			return
		}
		upgrader := dashboardRealtimeUpgrader
		upgrader.CheckOrigin = func(request *http.Request) bool {
			return middleware.OriginAllowed(request, svcCtx.Config.HTTP.AllowedOrigins)
		}
		upgrader.Subprotocols = []string{"alpha-auth"}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.WithContext(c.Request.Context()).Errorf("[DashboardRealtimeHandler] upgrade error: %v", err)
			return
		}
		defer conn.Close()

		var events <-chan realtime.DashboardMessage
		if svcCtx.DashboardRealtime != nil {
			sub, unsubscribe := svcCtx.DashboardRealtime.Subscribe()
			defer unsubscribe()
			events = sub.C
		}
		lastSeq := uint64(0)
		if svcCtx.DashboardRealtime != nil {
			lastSeq = svcCtx.DashboardRealtime.CurrentSeq()
		}

		writeFullPayload := func(seq uint64) bool {
			payload := buildDashboardRealtimePayload(c, svcCtx, seq)
			if err := conn.WriteJSON(payload); err != nil {
				return false
			}
			return true
		}
		writeDeltaPayload := func(event realtime.DashboardMessage) bool {
			payload := buildDashboardRealtimeDeltaPayload(c, svcCtx, event)
			if err := conn.WriteJSON(payload); err != nil {
				return false
			}
			return true
		}
		if !writeFullPayload(lastSeq) {
			return
		}

		conn.SetReadLimit(16 * 1024)
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
			return nil
		})

		pingTicker := time.NewTicker(30 * time.Second)
		defer pingTicker.Stop()
		readDone := make(chan struct{})
		go func() {
			defer close(readDone)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()

		for {
			select {
			case event := <-events:
				if event.Seq <= lastSeq {
					continue
				}
				if !writeDeltaPayload(event) {
					return
				}
				lastSeq = event.Seq
			case <-pingTicker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
					return
				}
			case <-readDone:
				return
			}
		}
	}
}

func buildDashboardRealtimePayload(c *gin.Context, svcCtx *svc.ServiceContext, seq uint64) dashboardRealtimePayload {
	payload := dashboardRealtimePayload{
		Type:      "dashboard_realtime",
		Full:      true,
		Seq:       seq,
		UpdatedAt: time.Now().UnixMilli(),
	}
	payload.Servers, payload.Network, payload.Connections, payload.Resources, payload.Alerts = queryDashboardRealtimeSnapshot(c, svcCtx)
	if onlineUsers, err := svcCtx.NodeModel.OnlineUserSubscribeGlobal(c.Request.Context()); err == nil {
		payload.OnlineUsers = onlineUsers
	} else {
		logger.WithContext(c.Request.Context()).Errorf("[DashboardRealtime] query online users error: %v", err)
	}
	payload.Business = queryDashboardBusiness(c, svcCtx)
	payload.Traffic = queryDashboardTraffic(c, svcCtx)
	payload.PendingTickets = queryDashboardPendingTickets(c, svcCtx)
	recordDashboardRealtimeSamples(c, svcCtx, payload.OnlineUsers, payload.Traffic.TodayUpload, payload.Traffic.TodayDownload, payload.Network)
	payload.OnlineUserSeries = queryDashboardOnlineUserSeries(c, svcCtx)
	payload.Traffic.TodaySeries = queryDashboardTrafficTodaySeries(c, svcCtx)
	payload.NetworkSeries = queryDashboardNetworkRateSeries(c, svcCtx)
	payload.Activities = queryDashboardActivities(c, svcCtx)
	return payload
}

func buildDashboardRealtimeDeltaPayload(c *gin.Context, svcCtx *svc.ServiceContext, event realtime.DashboardMessage) dashboardRealtimeDeltaPayload {
	traffic := queryDashboardTrafficTotals(c, svcCtx)
	payload := dashboardRealtimeDeltaPayload{
		Type:           "dashboard_realtime",
		Full:           false,
		Seq:            event.Seq,
		Reason:         string(event.Reason),
		UpdatedAt:      firstPositive(event.UpdatedAt, time.Now().UnixMilli()),
		Business:       queryDashboardBusiness(c, svcCtx),
		PendingTickets: queryDashboardPendingTickets(c, svcCtx),
		Traffic: dashboardRealtimeTrafficDelta{
			TodayUpload:     traffic.TodayUpload,
			TodayDownload:   traffic.TodayDownload,
			MonthlyUpload:   traffic.MonthlyUpload,
			MonthlyDownload: traffic.MonthlyDownload,
		},
	}
	payload.Servers, payload.Network, payload.Connections, payload.Resources, payload.Alerts = queryDashboardRealtimeSnapshot(c, svcCtx)
	if onlineUsers, err := svcCtx.NodeModel.OnlineUserSubscribeGlobal(c.Request.Context()); err == nil {
		payload.OnlineUsers = onlineUsers
	} else {
		logger.WithContext(c.Request.Context()).Errorf("[DashboardRealtime] query online users error: %v", err)
	}
	recordDashboardRealtimeSamples(c, svcCtx, payload.OnlineUsers, payload.Traffic.TodayUpload, payload.Traffic.TodayDownload, payload.Network)
	payload.NetworkSeries = []dashboardRealtimeNetworkPoint{networkRatePoint(payload.UpdatedAt, payload.Network)}
	if dashboardEventIncludesActivities(event.Reason) {
		payload.Activities = queryDashboardActivities(c, svcCtx)
	}
	return payload
}

func dashboardEventIncludesActivities(reason realtime.DashboardEvent) bool {
	switch reason {
	case realtime.DashboardEventActivity, realtime.DashboardEventBusiness, realtime.DashboardEventGeneral:
		return true
	default:
		return false
	}
}

func firstPositive(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func queryDashboardRealtimeSnapshot(c *gin.Context, svcCtx *svc.ServiceContext) (
	dashboardRealtimeServerSummary,
	dashboardRealtimeNetwork,
	dashboardRealtimeConnections,
	dashboardRealtimeResources,
	[]dashboardRealtimeAlert,
) {
	serverIDs := queryServerIDs(c, svcCtx)
	states := svcCtx.NodeRealtime.Snapshot(serverIDs)
	total := int64(len(serverIDs))
	servers := dashboardRealtimeServerSummary{
		Total:       total,
		NotReported: total - int64(len(states)),
		Reported:    int64(len(states)),
	}
	if servers.NotReported < 0 {
		servers.NotReported = 0
	}
	var network dashboardRealtimeNetwork
	var connections dashboardRealtimeConnections
	var resources dashboardRealtimeResources
	alerts := make([]dashboardRealtimeAlert, 0)
	var cpuTotal, memTotal, diskTotal float64
	var onlineStateCount int64
	for _, state := range states {
		if state.Online {
			servers.Online++
		} else {
			servers.Offline++
		}
		if state.XrayRunning {
			servers.XrayRunning++
		} else {
			servers.XrayStopped++
		}
		switch state.ConfigApplyStatus {
		case "ok":
			servers.ConfigOK++
		case "pending":
			servers.ConfigPending++
		case "":
		default:
			servers.ConfigFailed++
		}
		if state.XrayStatsError != "" {
			servers.StatsError++
		}
		if state.Online {
			onlineStateCount++
			network.SystemRxBps += state.NetRxBps
			network.SystemTxBps += state.NetTxBps
			network.XrayRxBps += state.XrayRxBps
			network.XrayTxBps += state.XrayTxBps
			connections.SystemInbound += state.SystemInboundConnections
			connections.SystemOutbound += state.SystemOutboundConnections
			connections.XrayInbound += state.XrayInboundConnections
			connections.XrayOutbound += state.XrayOutboundConnections
			cpuTotal += state.CPU
			memTotal += state.Mem
			diskTotal += state.Disk
			if state.CPU > resources.MaxCPU {
				resources.MaxCPU = state.CPU
			}
			if state.Mem > resources.MaxMem {
				resources.MaxMem = state.Mem
			}
			if state.Disk > resources.MaxDisk {
				resources.MaxDisk = state.Disk
			}
		}
		appendDashboardAlerts(&alerts, state)
	}
	if onlineStateCount > 0 {
		count := float64(onlineStateCount)
		resources.AvgCPU = cpuTotal / count
		resources.AvgMem = memTotal / count
		resources.AvgDisk = diskTotal / count
	}
	if total > servers.Online+servers.Offline {
		servers.Offline = total - servers.Online
	}
	return servers, network, connections, resources, alerts
}

func appendDashboardAlerts(alerts *[]dashboardRealtimeAlert, state realtime.NodeStatus) {
	if !state.Online {
		*alerts = append(*alerts, dashboardRealtimeAlert{
			Level:    "critical",
			ServerID: state.ServerID,
			Title:    "Server offline",
		})
	}
	if state.Online && !state.XrayRunning {
		*alerts = append(*alerts, dashboardRealtimeAlert{
			Level:    "critical",
			ServerID: state.ServerID,
			Title:    "Xray stopped",
		})
	}
	if state.ConfigApplyStatus != "" && state.ConfigApplyStatus != "ok" && state.ConfigApplyStatus != "pending" {
		*alerts = append(*alerts, dashboardRealtimeAlert{
			Level:    "warning",
			ServerID: state.ServerID,
			Title:    "Config apply failed",
			Message:  state.LastApplyError,
		})
	}
	if state.XrayStatsError != "" {
		*alerts = append(*alerts, dashboardRealtimeAlert{
			Level:    "warning",
			ServerID: state.ServerID,
			Title:    "Xray stats error",
			Message:  state.XrayStatsError,
		})
	}
	if state.CPU >= 90 || state.Mem >= 90 || state.Disk >= 90 {
		*alerts = append(*alerts, dashboardRealtimeAlert{
			Level:    "warning",
			ServerID: state.ServerID,
			Title:    "High resource usage",
			Message:  fmt.Sprintf("CPU %.1f%%, MEM %.1f%%, DISK %.1f%%", state.CPU, state.Mem, state.Disk),
		})
	}
}

func queryServerIDs(c *gin.Context, svcCtx *svc.ServiceContext) []int64 {
	var ids []int64
	if err := svcCtx.DB.WithContext(c.Request.Context()).Model(&node.Server{}).Pluck("id", &ids).Error; err != nil {
		logger.WithContext(c.Request.Context()).Errorf("[DashboardRealtimeHandler] query servers error: %v", err)
	}
	return ids
}

func queryDashboardBusiness(c *gin.Context, svcCtx *svc.ServiceContext) dashboardRealtimeBusiness {
	ctx := c.Request.Context()
	now := time.Now()
	todayStart := startOfDay(now)
	var revenue struct {
		AmountTotal        int64
		NewOrderAmount     int64
		RenewalOrderAmount int64
		MembershipAmount   int64
		ResetTrafficAmount int64
		NewOrderCount      int64
		RenewalOrderCount  int64
		MembershipCount    int64
	}
	if err := svcCtx.DB.WithContext(ctx).
		Model(&struct{}{}).
		Table(order.Order{}.TableName()).
		Where("status IN ? AND type IN ? AND created_at >= ? AND created_at < ?", paidOrderStatuses, businessOrderTypes, todayStart, todayStart.AddDate(0, 0, 1)).
		Select(`
			COALESCE(SUM(amount), 0) AS amount_total,
			COALESCE(SUM(CASE WHEN type = ? THEN amount ELSE 0 END), 0) AS new_order_amount,
			COALESCE(SUM(CASE WHEN type = ? THEN amount ELSE 0 END), 0) AS renewal_order_amount,
			COALESCE(SUM(CASE WHEN type = ? THEN amount ELSE 0 END), 0) AS membership_amount,
			COALESCE(SUM(CASE WHEN type = ? THEN amount ELSE 0 END), 0) AS reset_traffic_amount,
			COALESCE(SUM(CASE WHEN type = ? THEN 1 ELSE 0 END), 0) AS new_order_count,
			COALESCE(SUM(CASE WHEN type = ? THEN 1 ELSE 0 END), 0) AS renewal_order_count,
			COALESCE(SUM(CASE WHEN type = ? THEN 1 ELSE 0 END), 0) AS membership_count
		`, orderTypeSubscribe, orderTypeRenewal, orderTypeMembership, orderTypeResetTraffic, orderTypeSubscribe, orderTypeRenewal, orderTypeMembership).
		Scan(&revenue).Error; err != nil {
		logger.WithContext(ctx).Errorf("[DashboardRealtimeHandler] query business revenue error: %v", err)
	}
	var registerToday int64
	if err := svcCtx.DB.WithContext(ctx).
		Model(&user.User{}).
		Where("created_at >= ? AND created_at < ?", todayStart, todayStart.AddDate(0, 0, 1)).
		Count(&registerToday).Error; err != nil {
		logger.WithContext(ctx).Errorf("[DashboardRealtimeHandler] query business register error: %v", err)
	}
	return dashboardRealtimeBusiness{
		RevenueToday:         revenue.AmountTotal,
		NewOrderAmount:       revenue.NewOrderAmount,
		RenewalOrderAmount:   revenue.RenewalOrderAmount,
		MembershipAmount:     revenue.MembershipAmount,
		ResetTrafficAmount:   revenue.ResetTrafficAmount,
		RegisterToday:        registerToday,
		NewOrderCount:        revenue.NewOrderCount,
		RenewalOrderCount:    revenue.RenewalOrderCount,
		MembershipOrderCount: revenue.MembershipCount,
	}
}

func queryDashboardTraffic(c *gin.Context, svcCtx *svc.ServiceContext) dashboardRealtimeTraffic {
	result := queryDashboardTrafficTotals(c, svcCtx)
	result.TodaySeries = []dashboardRealtimeTrafficPoint{}
	result.ServerRankingToday = []dashboardRealtimeServerTraffic{}
	result.UserRankingToday = []dashboardRealtimeUserTraffic{}
	result.ServerRankingMonthly = []dashboardRealtimeServerTraffic{}
	result.UserRankingMonthly = []dashboardRealtimeUserTraffic{}
	today := time.Now().Format(time.DateOnly)
	monthStart := startOfMonth(time.Now()).Format(time.DateOnly)
	if servers, err := svcCtx.TrafficLogModel.TopDailyServerTraffic(c.Request.Context(), today, 10); err == nil {
		result.ServerRankingToday = buildServerTrafficRanking(c, svcCtx, servers)
	}
	if users, err := svcCtx.TrafficLogModel.TopDailyUserTraffic(c.Request.Context(), today, 10); err == nil {
		result.UserRankingToday = buildUserTrafficRanking(users)
	}
	result.ServerRankingMonthly = queryMonthlyServerTrafficRanking(c, svcCtx, monthStart, today, 10)
	result.UserRankingMonthly = queryMonthlyUserTrafficRanking(c, svcCtx, monthStart, today, 10)
	return result
}

func queryDashboardTrafficTotals(c *gin.Context, svcCtx *svc.ServiceContext) dashboardRealtimeTraffic {
	now := time.Now()
	today := now.Format(time.DateOnly)
	monthStart := startOfMonth(now).Format(time.DateOnly)
	return queryDashboardTrafficTotalsForDates(c, svcCtx, today, monthStart)
}

func queryDashboardTrafficTotalsForDates(c *gin.Context, svcCtx *svc.ServiceContext, today, monthStart string) dashboardRealtimeTraffic {
	var row dashboardTrafficTotalsRow
	if err := svcCtx.DB.WithContext(c.Request.Context()).
		Raw(dashboardTrafficTotalsSQL,
			today, today,
			monthStart, today,
			monthStart, today,
			monthStart, today,
		).
		Scan(&row).Error; err != nil {
		logger.WithContext(c.Request.Context()).Errorf("[DashboardRealtimeHandler] query traffic totals error: %v", err)
	}
	return row.toDashboardTraffic()
}

func (r dashboardTrafficTotalsRow) toDashboardTraffic() dashboardRealtimeTraffic {
	return dashboardRealtimeTraffic{
		TodayUpload:     r.TodayUpload,
		TodayDownload:   r.TodayDownload,
		MonthlyUpload:   r.MonthlyUpload,
		MonthlyDownload: r.MonthlyDownload,
	}
}

const dashboardTrafficTotalsSQL = `
SELECT
	COALESCE(SUM(CASE WHEN date = ? THEN upload ELSE 0 END), 0) AS today_upload,
	COALESCE(SUM(CASE WHEN date = ? THEN download ELSE 0 END), 0) AS today_download,
	COALESCE(SUM(CASE WHEN date >= ? AND date <= ? THEN upload ELSE 0 END), 0) AS monthly_upload,
	COALESCE(SUM(CASE WHEN date >= ? AND date <= ? THEN download ELSE 0 END), 0) AS monthly_download
FROM "daily_traffic_stat"
WHERE date >= ? AND date <= ?
`

func queryDashboardPendingTickets(c *gin.Context, svcCtx *svc.ServiceContext) int64 {
	count, err := svcCtx.TicketModel.QueryWaitReplyTotal(c.Request.Context())
	if err != nil {
		logger.WithContext(c.Request.Context()).Errorf("[DashboardRealtimeHandler] query pending tickets error: %v", err)
		return 0
	}
	return count
}

func recordDashboardRealtimeSamples(c *gin.Context, svcCtx *svc.ServiceContext, onlineUsers, todayUpload, todayDownload int64, network dashboardRealtimeNetwork) {
	ctx := c.Request.Context()
	now := time.Now()
	bucket := now.Truncate(time.Minute).UnixMilli()
	member := strconv.FormatInt(bucket, 10)
	networkBucket := now.Truncate(5 * time.Second).UnixMilli()
	networkMember := strconv.FormatInt(networkBucket, 10)
	dedupeKey := fmt.Sprintf("dashboard:realtime:samples:dedupe:%d", networkBucket)
	shouldRecord, err := svcCtx.Redis.SetNX(ctx, dedupeKey, "1", 5*time.Second).Result()
	if err != nil {
		logger.WithContext(ctx).Debugf("[DashboardRealtimeHandler] sample dedupe error: %v", err)
	} else if !shouldRecord {
		return
	}
	pipe := svcCtx.Redis.Pipeline()
	pipe.ZAdd(ctx, dashboardOnlineSeriesIndexKey, redis.Z{Score: float64(bucket), Member: member})
	pipe.HSet(ctx, dashboardOnlineSeriesValuesKey, member, strconv.FormatInt(onlineUsers, 10))
	pipe.Expire(ctx, dashboardOnlineSeriesIndexKey, dashboardSampleTTL)
	pipe.Expire(ctx, dashboardOnlineSeriesValuesKey, dashboardSampleTTL)
	trafficValue, _ := json.Marshal(dashboardRealtimeTrafficPoint{
		Timestamp: bucket,
		Upload:    todayUpload,
		Download:  todayDownload,
		Total:     todayUpload + todayDownload,
	})
	pipe.ZAdd(ctx, dashboardTrafficSeriesIndexKey, redis.Z{Score: float64(bucket), Member: member})
	pipe.HSet(ctx, dashboardTrafficSeriesValuesKey, member, string(trafficValue))
	pipe.Expire(ctx, dashboardTrafficSeriesIndexKey, dashboardSampleTTL)
	pipe.Expire(ctx, dashboardTrafficSeriesValuesKey, dashboardSampleTTL)
	networkValue, _ := json.Marshal(networkRatePoint(networkBucket, network))
	pipe.ZAdd(ctx, dashboardNetworkSeriesIndexKey, redis.Z{Score: float64(networkBucket), Member: networkMember})
	pipe.HSet(ctx, dashboardNetworkSeriesValuesKey, networkMember, string(networkValue))
	pipe.Expire(ctx, dashboardNetworkSeriesIndexKey, dashboardNetworkSampleTTL)
	pipe.Expire(ctx, dashboardNetworkSeriesValuesKey, dashboardNetworkSampleTTL)
	cutoff := strconv.FormatInt(now.Add(-7*24*time.Hour).UnixMilli(), 10)
	if oldOnlineMembers, err := svcCtx.Redis.ZRangeByScore(ctx, dashboardOnlineSeriesIndexKey, &redis.ZRangeBy{Min: "-inf", Max: cutoff}).Result(); err == nil && len(oldOnlineMembers) > 0 {
		pipe.HDel(ctx, dashboardOnlineSeriesValuesKey, oldOnlineMembers...)
	}
	if oldTrafficMembers, err := svcCtx.Redis.ZRangeByScore(ctx, dashboardTrafficSeriesIndexKey, &redis.ZRangeBy{Min: "-inf", Max: cutoff}).Result(); err == nil && len(oldTrafficMembers) > 0 {
		pipe.HDel(ctx, dashboardTrafficSeriesValuesKey, oldTrafficMembers...)
	}
	networkCutoff := strconv.FormatInt(now.Add(-dashboardNetworkSampleTTL).UnixMilli(), 10)
	if oldNetworkMembers, err := svcCtx.Redis.ZRangeByScore(ctx, dashboardNetworkSeriesIndexKey, &redis.ZRangeBy{Min: "-inf", Max: networkCutoff}).Result(); err == nil && len(oldNetworkMembers) > 0 {
		pipe.HDel(ctx, dashboardNetworkSeriesValuesKey, oldNetworkMembers...)
	}
	pipe.ZRemRangeByScore(ctx, dashboardOnlineSeriesIndexKey, "-inf", cutoff)
	pipe.ZRemRangeByScore(ctx, dashboardTrafficSeriesIndexKey, "-inf", cutoff)
	pipe.ZRemRangeByScore(ctx, dashboardNetworkSeriesIndexKey, "-inf", networkCutoff)
	if _, err := pipe.Exec(ctx); err != nil {
		logger.WithContext(ctx).Debugf("[DashboardRealtimeHandler] record samples error: %v", err)
	}
}

func networkRatePoint(timestamp int64, network dashboardRealtimeNetwork) dashboardRealtimeNetworkPoint {
	return dashboardRealtimeNetworkPoint{
		Timestamp:   timestamp,
		SystemRxBps: network.SystemRxBps,
		SystemTxBps: network.SystemTxBps,
		XrayRxBps:   network.XrayRxBps,
		XrayTxBps:   network.XrayTxBps,
	}
}

func queryDashboardOnlineUserSeries(c *gin.Context, svcCtx *svc.ServiceContext) dashboardRealtimeOnlineSeries {
	now := time.Now()
	points := queryDashboardValueSamples(c, svcCtx, dashboardOnlineSeriesIndexKey, dashboardOnlineSeriesValuesKey, now.Add(-7*24*time.Hour), now)
	return buildDashboardOnlineUserSeries(points, now)
}

func buildDashboardOnlineUserSeries(points []dashboardRealtimeValuePoint, now time.Time) dashboardRealtimeOnlineSeries {
	if len(points) == 0 {
		return dashboardRealtimeOnlineSeries{
			LastDay:   []dashboardRealtimeValuePoint{},
			Last7Days: []dashboardRealtimeValuePoint{},
		}
	}
	lastDay := compactValuePointsCopy(filterValuePointsSince(points, now.Add(-24*time.Hour)), 5*time.Minute)
	return dashboardRealtimeOnlineSeries{
		LastDay:   lastDay,
		Last7Days: compactValuePoints(points, time.Hour),
	}
}

func queryDashboardTrafficTodaySeries(c *gin.Context, svcCtx *svc.ServiceContext) []dashboardRealtimeTrafficPoint {
	now := time.Now()
	points := queryDashboardTrafficSamples(c, svcCtx, dashboardTrafficSeriesIndexKey, dashboardTrafficSeriesValuesKey, startOfDay(now), now)
	return compactTrafficPoints(points, 5*time.Minute)
}

func queryDashboardNetworkRateSeries(c *gin.Context, svcCtx *svc.ServiceContext) []dashboardRealtimeNetworkPoint {
	now := time.Now()
	points := queryDashboardNetworkSamples(c, svcCtx, now.Add(-dashboardNetworkSampleTTL), now)
	return compactNetworkPoints(points, time.Minute)
}

func queryDashboardValueSamples(c *gin.Context, svcCtx *svc.ServiceContext, indexKey, valuesKey string, start, end time.Time) []dashboardRealtimeValuePoint {
	ctx := c.Request.Context()
	members, err := svcCtx.Redis.ZRangeByScore(ctx, indexKey, &redis.ZRangeBy{
		Min: strconv.FormatInt(start.UnixMilli(), 10),
		Max: strconv.FormatInt(end.UnixMilli(), 10),
	}).Result()
	if err != nil || len(members) == 0 {
		if err != nil {
			logger.WithContext(ctx).Debugf("[DashboardRealtimeHandler] query value sample index error: %v", err)
		}
		return []dashboardRealtimeValuePoint{}
	}
	values, err := svcCtx.Redis.HMGet(ctx, valuesKey, members...).Result()
	if err != nil {
		logger.WithContext(ctx).Debugf("[DashboardRealtimeHandler] query value samples error: %v", err)
		return []dashboardRealtimeValuePoint{}
	}
	points := make([]dashboardRealtimeValuePoint, 0, len(members))
	for index, member := range members {
		timestamp, err := strconv.ParseInt(member, 10, 64)
		if err != nil || values[index] == nil {
			continue
		}
		value, ok := dashboardRedisInt64(values[index])
		if !ok {
			continue
		}
		points = append(points, dashboardRealtimeValuePoint{Timestamp: timestamp, Value: value})
	}
	return points
}

func queryDashboardTrafficSamples(c *gin.Context, svcCtx *svc.ServiceContext, indexKey, valuesKey string, start, end time.Time) []dashboardRealtimeTrafficPoint {
	ctx := c.Request.Context()
	members, err := svcCtx.Redis.ZRangeByScore(ctx, indexKey, &redis.ZRangeBy{
		Min: strconv.FormatInt(start.UnixMilli(), 10),
		Max: strconv.FormatInt(end.UnixMilli(), 10),
	}).Result()
	if err != nil || len(members) == 0 {
		if err != nil {
			logger.WithContext(ctx).Debugf("[DashboardRealtimeHandler] query traffic sample index error: %v", err)
		}
		return []dashboardRealtimeTrafficPoint{}
	}
	values, err := svcCtx.Redis.HMGet(ctx, valuesKey, members...).Result()
	if err != nil {
		logger.WithContext(ctx).Debugf("[DashboardRealtimeHandler] query traffic samples error: %v", err)
		return []dashboardRealtimeTrafficPoint{}
	}
	points := make([]dashboardRealtimeTrafficPoint, 0, len(members))
	for index, member := range members {
		timestamp, err := strconv.ParseInt(member, 10, 64)
		if err != nil || values[index] == nil {
			continue
		}
		var point dashboardRealtimeTrafficPoint
		if !dashboardRedisJSON(values[index], &point) {
			continue
		}
		point.Timestamp = timestamp
		points = append(points, point)
	}
	return points
}

func queryDashboardNetworkSamples(c *gin.Context, svcCtx *svc.ServiceContext, start, end time.Time) []dashboardRealtimeNetworkPoint {
	ctx := c.Request.Context()
	members, err := svcCtx.Redis.ZRangeByScore(ctx, dashboardNetworkSeriesIndexKey, &redis.ZRangeBy{
		Min: strconv.FormatInt(start.UnixMilli(), 10),
		Max: strconv.FormatInt(end.UnixMilli(), 10),
	}).Result()
	if err != nil || len(members) == 0 {
		if err != nil {
			logger.WithContext(ctx).Debugf("[DashboardRealtimeHandler] query network sample index error: %v", err)
		}
		return []dashboardRealtimeNetworkPoint{}
	}
	values, err := svcCtx.Redis.HMGet(ctx, dashboardNetworkSeriesValuesKey, members...).Result()
	if err != nil {
		logger.WithContext(ctx).Debugf("[DashboardRealtimeHandler] query network samples error: %v", err)
		return []dashboardRealtimeNetworkPoint{}
	}
	points := make([]dashboardRealtimeNetworkPoint, 0, len(members))
	for index, member := range members {
		timestamp, err := strconv.ParseInt(member, 10, 64)
		if err != nil || values[index] == nil {
			continue
		}
		var point dashboardRealtimeNetworkPoint
		if !dashboardRedisJSON(values[index], &point) {
			continue
		}
		point.Timestamp = timestamp
		points = append(points, point)
	}
	return points
}

func dashboardRedisInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		if uint64(v) > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		if v > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(v), true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return parsed, err == nil
	case []byte:
		return dashboardRedisInt64Bytes(v)
	default:
		return 0, false
	}
}

func dashboardRedisInt64Bytes(raw []byte) (int64, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return 0, false
	}
	negative := false
	switch raw[0] {
	case '-':
		negative = true
		raw = raw[1:]
	case '+':
		raw = raw[1:]
	}
	if len(raw) == 0 {
		return 0, false
	}

	const maxInt64 = uint64(^uint64(0) >> 1)
	limit := maxInt64
	if negative {
		limit++
	}
	var value uint64
	for _, c := range raw {
		if c < '0' || c > '9' {
			return 0, false
		}
		digit := uint64(c - '0')
		if value > (limit-digit)/10 {
			return 0, false
		}
		value = value*10 + digit
	}
	if negative {
		if value == limit {
			return -1 << 63, true
		}
		return -int64(value), true
	}
	return int64(value), true
}

func dashboardRedisJSON(value any, target any) bool {
	var raw []byte
	switch v := value.(type) {
	case string:
		raw = []byte(v)
	case []byte:
		raw = v
	default:
		return false
	}
	return json.Unmarshal(raw, target) == nil
}

func filterValuePointsSince(points []dashboardRealtimeValuePoint, since time.Time) []dashboardRealtimeValuePoint {
	cutoff := since.UnixMilli()
	start := sort.Search(len(points), func(index int) bool {
		return points[index].Timestamp >= cutoff
	})
	return points[start:]
}

func compactValuePoints(points []dashboardRealtimeValuePoint, interval time.Duration) []dashboardRealtimeValuePoint {
	if len(points) <= 1 {
		return points
	}
	intervalMs := interval.Milliseconds()
	if intervalMs <= 0 {
		return points
	}
	result := points[:0]
	currentBucket := points[0].Timestamp / intervalMs * intervalMs
	currentPoint := points[0]
	for _, point := range points[1:] {
		key := point.Timestamp / intervalMs * intervalMs
		if key == currentBucket {
			currentPoint = point
			continue
		}
		result = append(result, currentPoint)
		currentBucket = key
		currentPoint = point
	}
	result = append(result, currentPoint)
	return result
}

func compactValuePointsCopy(points []dashboardRealtimeValuePoint, interval time.Duration) []dashboardRealtimeValuePoint {
	if len(points) == 0 {
		return points
	}
	intervalMs := interval.Milliseconds()
	if len(points) <= 1 || intervalMs <= 0 {
		result := make([]dashboardRealtimeValuePoint, len(points))
		copy(result, points)
		return result
	}
	result := make([]dashboardRealtimeValuePoint, 0, compactValuePointsCapacity(points, intervalMs))
	currentBucket := points[0].Timestamp / intervalMs * intervalMs
	currentPoint := points[0]
	for _, point := range points[1:] {
		key := point.Timestamp / intervalMs * intervalMs
		if key == currentBucket {
			currentPoint = point
			continue
		}
		result = append(result, currentPoint)
		currentBucket = key
		currentPoint = point
	}
	result = append(result, currentPoint)
	return result
}

func compactValuePointsCapacity(points []dashboardRealtimeValuePoint, intervalMs int64) int {
	if len(points) == 0 || intervalMs <= 0 {
		return len(points)
	}
	firstBucket := points[0].Timestamp / intervalMs
	lastBucket := points[len(points)-1].Timestamp / intervalMs
	if lastBucket < firstBucket {
		return len(points)
	}
	capacity := int(lastBucket-firstBucket) + 1
	if capacity > len(points) {
		return len(points)
	}
	return capacity
}

func compactTrafficPoints(points []dashboardRealtimeTrafficPoint, interval time.Duration) []dashboardRealtimeTrafficPoint {
	if len(points) <= 1 {
		return points
	}
	intervalMs := interval.Milliseconds()
	if intervalMs <= 0 {
		return points
	}
	result := points[:0]
	currentBucket := points[0].Timestamp / intervalMs * intervalMs
	currentPoint := points[0]
	for _, point := range points[1:] {
		key := point.Timestamp / intervalMs * intervalMs
		if key == currentBucket {
			currentPoint = point
			continue
		}
		result = append(result, currentPoint)
		currentBucket = key
		currentPoint = point
	}
	result = append(result, currentPoint)
	return result
}

func compactNetworkPoints(points []dashboardRealtimeNetworkPoint, interval time.Duration) []dashboardRealtimeNetworkPoint {
	if len(points) <= 1 {
		return points
	}
	intervalMs := interval.Milliseconds()
	if intervalMs <= 0 {
		return points
	}
	result := points[:0]
	currentBucket := points[0].Timestamp / intervalMs * intervalMs
	currentPoint := points[0]
	for _, point := range points[1:] {
		key := point.Timestamp / intervalMs * intervalMs
		if key == currentBucket {
			currentPoint = point
			continue
		}
		result = append(result, currentPoint)
		currentBucket = key
		currentPoint = point
	}
	result = append(result, currentPoint)
	return result
}

func buildServerTrafficRanking(c *gin.Context, svcCtx *svc.ServiceContext, rows []trafficModel.DailyServerTraffic) []dashboardRealtimeServerTraffic {
	serverIDs := make([]int64, 0, len(rows))
	for _, item := range rows {
		serverIDs = append(serverIDs, item.ServerId)
	}
	names := queryServerNames(c, svcCtx, serverIDs)
	result := make([]dashboardRealtimeServerTraffic, 0, len(rows))
	for _, item := range rows {
		result = append(result, dashboardRealtimeServerTraffic{
			ServerID: item.ServerId,
			Name:     displayServer(item.ServerId, names[item.ServerId]),
			Upload:   item.Upload,
			Download: item.Download,
		})
	}
	return result
}

func buildUserTrafficRanking(rows []trafficModel.DailyUserTraffic) []dashboardRealtimeUserTraffic {
	result := make([]dashboardRealtimeUserTraffic, 0, len(rows))
	for _, item := range rows {
		result = append(result, dashboardRealtimeUserTraffic{
			SID:      item.SubscribeId,
			Upload:   item.Upload,
			Download: item.Download,
		})
	}
	return result
}

func queryMonthlyServerTrafficRanking(c *gin.Context, svcCtx *svc.ServiceContext, start, end string, limit int) []dashboardRealtimeServerTraffic {
	type row struct {
		ServerId int64
		Upload   int64
		Download int64
		Total    int64
	}
	var rows []row
	if err := svcCtx.DB.WithContext(c.Request.Context()).
		Model(&trafficModel.DailyServerTraffic{}).
		Where("date >= ? AND date <= ?", start, end).
		Select("server_id, COALESCE(SUM(upload), 0) AS upload, COALESCE(SUM(download), 0) AS download, COALESCE(SUM(total), 0) AS total").
		Group("server_id").
		Order("total DESC, server_id DESC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		logger.WithContext(c.Request.Context()).Errorf("[DashboardRealtimeHandler] query monthly server traffic ranking error: %v", err)
		return []dashboardRealtimeServerTraffic{}
	}
	serverIDs := make([]int64, 0, len(rows))
	for _, item := range rows {
		serverIDs = append(serverIDs, item.ServerId)
	}
	names := queryServerNames(c, svcCtx, serverIDs)
	result := make([]dashboardRealtimeServerTraffic, 0, len(rows))
	for _, item := range rows {
		result = append(result, dashboardRealtimeServerTraffic{
			ServerID: item.ServerId,
			Name:     displayServer(item.ServerId, names[item.ServerId]),
			Upload:   item.Upload,
			Download: item.Download,
		})
	}
	return result
}

func queryMonthlyUserTrafficRanking(c *gin.Context, svcCtx *svc.ServiceContext, start, end string, limit int) []dashboardRealtimeUserTraffic {
	type row struct {
		SubscribeId int64
		Upload      int64
		Download    int64
		Total       int64
	}
	var rows []row
	if err := svcCtx.DB.WithContext(c.Request.Context()).
		Model(&trafficModel.DailyUserTraffic{}).
		Where("date >= ? AND date <= ?", start, end).
		Select("subscribe_id, COALESCE(SUM(upload), 0) AS upload, COALESCE(SUM(download), 0) AS download, COALESCE(SUM(total), 0) AS total").
		Group("subscribe_id").
		Order("total DESC, subscribe_id DESC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		logger.WithContext(c.Request.Context()).Errorf("[DashboardRealtimeHandler] query monthly user traffic ranking error: %v", err)
		return []dashboardRealtimeUserTraffic{}
	}
	result := make([]dashboardRealtimeUserTraffic, 0, len(rows))
	for _, item := range rows {
		result = append(result, dashboardRealtimeUserTraffic{
			SID:      item.SubscribeId,
			Upload:   item.Upload,
			Download: item.Download,
		})
	}
	return result
}

func queryServerNames(c *gin.Context, svcCtx *svc.ServiceContext, ids []int64) map[int64]string {
	if len(ids) == 0 {
		return nil
	}
	ids = uniqueDashboardInt64s(ids)
	if len(ids) == 0 {
		return nil
	}
	var servers []node.Server
	if err := svcCtx.DB.WithContext(c.Request.Context()).
		Model(&node.Server{}).
		Select("id", "name").
		Where("id IN ?", ids).
		Find(&servers).Error; err != nil {
		logger.WithContext(c.Request.Context()).Errorf("[DashboardRealtimeHandler] query traffic server names error: %v", err)
		return nil
	}
	result := make(map[int64]string, len(ids))
	for _, item := range servers {
		result[item.Id] = item.Name
	}
	return result
}

func queryDashboardActivities(c *gin.Context, svcCtx *svc.ServiceContext) []dashboardRealtimeActivity {
	ctx := c.Request.Context()
	activities := make([]dashboardRealtimeActivity, 0, dashboardActivityLimit*3)
	userLabelCache := make(map[int64]string, dashboardActivityLimit*3)

	var systemLogs []dashboardSystemActivityLog
	if err := svcCtx.DB.WithContext(ctx).
		Model(&log.SystemLog{}).
		Select(dashboardSystemActivitySelectColumns).
		Where("type IN ?", []int64{int64(log.TypeLogin.Uint8()), int64(log.TypeRegister.Uint8()), int64(log.TypeLogout.Uint8())}).
		Order("id DESC").
		Limit(dashboardActivityLimit).
		Scan(&systemLogs).Error; err == nil {
		userIDs := dashboardSystemActivityUserIDs(systemLogs)
		userNames := queryUserLabelsCached(c, svcCtx, userIDs, userLabelCache)
		for _, item := range systemLogs {
			switch item.Type {
			case log.TypeLogin.Uint8():
				status := "success"
				if !item.Success {
					status = "failed"
				}
				activities = append(activities, dashboardRealtimeActivity{
					ID:        dashboardPrefixedInt("log-", item.Id),
					Type:      "login",
					Title:     "User login",
					Subject:   displayUser(item.UserID, userNames[item.UserID]),
					Detail:    activityDetail(item.Method),
					IP:        dashboardDisplayIP(item.ClientIP),
					Location:  dashboardActivityLocation(svcCtx, item.ClientIP, item.Location, item.CountryShort, item.Region),
					Status:    status,
					UserID:    item.UserID,
					CreatedAt: activityTime(item.CreatedAt, item.Timestamp),
				})
			case log.TypeRegister.Uint8():
				activities = append(activities, dashboardRealtimeActivity{
					ID:        dashboardPrefixedInt("log-", item.Id),
					Type:      "register",
					Title:     "New registration",
					Subject:   displayUser(item.UserID, firstNonEmpty(dashboardDisplayIdentifier(item.Identifier), userNames[item.UserID])),
					Detail:    activityDetail(item.AuthMethod),
					IP:        dashboardDisplayIP(item.ClientIP),
					Location:  dashboardActivityLocation(svcCtx, item.ClientIP, item.Location, item.CountryShort, item.Region),
					Status:    "success",
					UserID:    item.UserID,
					CreatedAt: activityTime(item.CreatedAt, item.Timestamp),
				})
			case log.TypeLogout.Uint8():
				activities = append(activities, dashboardRealtimeActivity{
					ID:        dashboardPrefixedInt("log-", item.Id),
					Type:      "logout",
					Title:     "User logout",
					Subject:   displayUser(item.UserID, userNames[item.UserID]),
					IP:        dashboardDisplayIP(item.ClientIP),
					Location:  dashboardActivityLocation(svcCtx, item.ClientIP, item.Location, item.CountryShort, item.Region),
					Status:    "success",
					UserID:    item.UserID,
					CreatedAt: activityTime(item.CreatedAt, item.Timestamp),
				})
			}
		}
	} else {
		logger.WithContext(ctx).Errorf("[DashboardRealtimeHandler] query activity logs error: %v", err)
	}

	var orders []dashboardOrderActivity
	if err := svcCtx.DB.WithContext(ctx).
		Model(&order.Order{}).
		Select(dashboardOrderActivitySelectColumns).
		Joins(`LEFT JOIN subscribe ON subscribe.id = "order".subscribe_id`).
		Order(`"order".id DESC`).
		Limit(dashboardActivityLimit).
		Scan(&orders).Error; err == nil {
		userNames := queryActivityUsers(c, svcCtx, orders, userLabelCache)
		planNames := queryActivityMembershipPlans(c, svcCtx, orders)
		orderNetworks := queryActivityOrderNetworks(c, svcCtx, orders)
		for _, item := range orders {
			activityType, title := orderActivityTypeTitle(item.Type)
			product := orderActivityProduct(item, planNames)
			network := orderNetworks[item.Id]
			activities = append(activities, dashboardRealtimeActivity{
				ID:        dashboardPrefixedInt("order-", item.Id),
				Type:      activityType,
				Title:     title,
				Subject:   displayUser(item.UserId, userNames[item.UserId]),
				Detail:    product,
				IP:        dashboardDisplayIP(network.IP),
				Location:  network.Location,
				Status:    orderStatusText(item.Status),
				Amount:    item.Amount,
				UserID:    item.UserId,
				CreatedAt: item.CreatedAt.UnixMilli(),
			})
		}
	} else {
		logger.WithContext(ctx).Errorf("[DashboardRealtimeHandler] query activity orders error: %v", err)
	}

	var subscribeLogs []log.SubscribeAccessLog
	if err := svcCtx.DB.WithContext(ctx).
		Model(&log.SubscribeAccessLog{}).
		Select(dashboardSubscribeActivitySelectColumns).
		Order("created_at DESC, id DESC").
		Limit(dashboardActivityLimit).
		Find(&subscribeLogs).Error; err == nil {
		userIDs, subscribeIDs := dashboardSubscribeActivityTargets(subscribeLogs)
		userNames := queryUserLabelsCached(c, svcCtx, userIDs, userLabelCache)
		subscribeNames := queryUserSubscribeNames(c, svcCtx, subscribeIDs)
		for _, item := range subscribeLogs {
			status := "success"
			if !item.Success {
				status = "failed"
			}
			activities = append(activities, dashboardRealtimeActivity{
				ID:        dashboardPrefixedInt("subscribe-", item.Id),
				Type:      "subscribe_pull",
				Title:     "Subscription pulled",
				Subject:   displayUser(item.UserId, userNames[item.UserId]),
				Detail:    displaySubscription(item.UserSubscribeId, subscribeNames[item.UserSubscribeId]),
				IP:        dashboardDisplayIP(item.ClientIP),
				Location:  dashboardActivityLocation(svcCtx, item.ClientIP, item.Location, item.CountryShort, item.Region),
				Status:    status,
				UserID:    item.UserId,
				CreatedAt: item.CreatedAt.UnixMilli(),
			})
		}
	} else {
		logger.WithContext(ctx).Errorf("[DashboardRealtimeHandler] query subscribe activities error: %v", err)
	}

	sort.SliceStable(activities, func(i, j int) bool {
		return activities[i].CreatedAt > activities[j].CreatedAt
	})
	if len(activities) > dashboardActivityLimit {
		activities = activities[:dashboardActivityLimit]
	}
	return activities
}

func dashboardSystemActivityUserIDs(rows []dashboardSystemActivityLog) []int64 {
	userIDs := viewutil.NewRowsCollector[int64](len(rows))
	for _, item := range rows {
		if item.UserID > 0 {
			userIDs.Add(item.UserID)
		}
	}
	return userIDs.Rows()
}

func dashboardSubscribeActivityTargets(rows []log.SubscribeAccessLog) ([]int64, []int64) {
	userIDs := viewutil.NewRowsCollector[int64](len(rows))
	subscribeIDs := viewutil.NewRowsCollector[int64](len(rows))
	for _, item := range rows {
		if item.UserId > 0 {
			userIDs.Add(item.UserId)
		}
		if item.UserSubscribeId > 0 {
			subscribeIDs.Add(item.UserSubscribeId)
		}
	}
	return userIDs.Rows(), subscribeIDs.Rows()
}

type dashboardActivityNetworkCandidate struct {
	UserID    int64
	IP        string
	Location  string
	CreatedAt time.Time
}

type dashboardActivityNetworkCandidateRow struct {
	UserID       int64
	IP           string
	CountryShort string
	Region       string
	Location     string
	CreatedAt    time.Time
}

func queryActivityOrderNetworks(c *gin.Context, svcCtx *svc.ServiceContext, orders []dashboardOrderActivity) map[int64]dashboardActivityNetworkInfo {
	if len(orders) == 0 {
		return nil
	}
	var result map[int64]dashboardActivityNetworkInfo
	var userIDs []int64
	var minCreated time.Time
	var maxCreated time.Time
	for _, item := range orders {
		if item.ClientIP != "" {
			if result == nil {
				result = make(map[int64]dashboardActivityNetworkInfo, len(orders))
			}
			result[item.Id] = dashboardActivityNetworkInfo{
				IP:        item.ClientIP,
				Location:  dashboardActivityLocation(svcCtx, item.ClientIP, item.Location, item.CountryShort, item.Region),
				CreatedAt: item.CreatedAt,
			}
			continue
		}
		if item.UserId <= 0 {
			continue
		}
		userIDs = append(userIDs, item.UserId)
		if minCreated.IsZero() || item.CreatedAt.Before(minCreated) {
			minCreated = item.CreatedAt
		}
		if maxCreated.IsZero() || item.CreatedAt.After(maxCreated) {
			maxCreated = item.CreatedAt
		}
	}
	userIDs = uniqueDashboardInt64s(userIDs)
	if len(userIDs) == 0 || minCreated.IsZero() || maxCreated.IsZero() {
		return result
	}

	candidates := queryActivityNetworkCandidates(c, svcCtx, userIDs, minCreated.AddDate(0, -1, 0), maxCreated.Add(time.Hour))
	if len(candidates) == 0 {
		return result
	}
	byUser := newDashboardActivityNetworkCandidateIndex(userIDs)
	for _, item := range candidates {
		byUser.Add(item)
	}
	for _, item := range orders {
		if _, ok := result[item.Id]; ok || item.UserId <= 0 {
			continue
		}
		cutoff := item.CreatedAt.Add(time.Hour)
		for _, candidate := range byUser.Get(item.UserId) {
			if candidate.IP == "" || candidate.CreatedAt.After(cutoff) {
				continue
			}
			if result == nil {
				result = make(map[int64]dashboardActivityNetworkInfo, len(orders))
			}
			result[item.Id] = dashboardActivityNetworkInfo{
				IP:        candidate.IP,
				Location:  candidate.Location,
				CreatedAt: candidate.CreatedAt,
			}
			break
		}
	}
	return result
}

type dashboardActivityNetworkCandidateBucket struct {
	userID     int64
	candidates []dashboardActivityNetworkCandidate
}

type dashboardActivityNetworkCandidateIndex struct {
	buckets []dashboardActivityNetworkCandidateBucket
	byUser  map[int64]int
	capHint int
}

func newDashboardActivityNetworkCandidateIndex(userIDs []int64) dashboardActivityNetworkCandidateIndex {
	index := dashboardActivityNetworkCandidateIndex{capHint: len(userIDs)}
	for _, userID := range userIDs {
		index.addUser(userID)
	}
	return index
}

func (i *dashboardActivityNetworkCandidateIndex) addUser(userID int64) {
	if userID <= 0 {
		return
	}
	if i.byUser != nil {
		if _, ok := i.byUser[userID]; ok {
			return
		}
		i.byUser[userID] = len(i.buckets)
		i.buckets = append(i.buckets, dashboardActivityNetworkCandidateBucket{userID: userID})
		return
	}
	for _, bucket := range i.buckets {
		if bucket.userID == userID {
			return
		}
	}
	i.ensureBuckets()
	i.buckets = append(i.buckets, dashboardActivityNetworkCandidateBucket{userID: userID})
	if len(i.buckets) > dashboardSmallInt64SetLimit {
		i.upgradeLookup()
	}
}

func (i *dashboardActivityNetworkCandidateIndex) ensureBuckets() {
	if i.buckets != nil {
		return
	}
	i.buckets = make([]dashboardActivityNetworkCandidateBucket, 0, dashboardInt64InitialCap(i.capHint))
}

func (i *dashboardActivityNetworkCandidateIndex) upgradeLookup() {
	if i.byUser != nil {
		return
	}
	nextCap := dashboardInt64LookupCap(len(i.buckets), i.capHint)
	if nextCap > cap(i.buckets) {
		buckets := make([]dashboardActivityNetworkCandidateBucket, len(i.buckets), nextCap)
		copy(buckets, i.buckets)
		i.buckets = buckets
	}
	i.byUser = make(map[int64]int, nextCap)
	for row, bucket := range i.buckets {
		i.byUser[bucket.userID] = row
	}
}

func (i *dashboardActivityNetworkCandidateIndex) Add(candidate dashboardActivityNetworkCandidate) {
	if candidate.UserID <= 0 {
		return
	}
	if i.byUser != nil {
		row, ok := i.byUser[candidate.UserID]
		if !ok {
			return
		}
		i.buckets[row].candidates = append(i.buckets[row].candidates, candidate)
		return
	}
	for row := range i.buckets {
		if i.buckets[row].userID == candidate.UserID {
			i.buckets[row].candidates = append(i.buckets[row].candidates, candidate)
			return
		}
	}
}

func (i dashboardActivityNetworkCandidateIndex) Get(userID int64) []dashboardActivityNetworkCandidate {
	if i.byUser != nil {
		row, ok := i.byUser[userID]
		if !ok {
			return nil
		}
		if i.buckets[row].candidates == nil {
			return []dashboardActivityNetworkCandidate{}
		}
		return i.buckets[row].candidates
	}
	for row := range i.buckets {
		if i.buckets[row].userID == userID {
			if i.buckets[row].candidates == nil {
				return []dashboardActivityNetworkCandidate{}
			}
			return i.buckets[row].candidates
		}
	}
	return nil
}

const dashboardActivityNetworkCandidatesSQL = `
SELECT
	user_id,
	ip,
	country_short,
	region,
	location,
	created_at
FROM (
	SELECT
		object_id AS user_id,
		ip,
		country_short,
		region,
		location,
		created_at,
		id
	FROM system_logs
	WHERE object_id IN ? AND ip <> '' AND created_at BETWEEN ? AND ?
	UNION ALL
	SELECT
		user_id,
		client_ip AS ip,
		country_short,
		region,
		location,
		created_at,
		id
	FROM subscribe_access_logs
	WHERE user_id IN ? AND client_ip <> '' AND created_at BETWEEN ? AND ?
) AS activity_network
ORDER BY created_at DESC, id DESC
LIMIT ?
`

func queryActivityNetworkCandidates(c *gin.Context, svcCtx *svc.ServiceContext, userIDs []int64, start, end time.Time) []dashboardActivityNetworkCandidate {
	ctx := c.Request.Context()
	limit := dashboardActivityLimit * 8
	if userLimit := len(userIDs) * 8; userLimit > limit {
		limit = userLimit
	}
	rows := make([]dashboardActivityNetworkCandidateRow, 0)
	if err := svcCtx.DB.WithContext(ctx).
		Raw(dashboardActivityNetworkCandidatesSQL, userIDs, start, end, userIDs, start, end, limit).
		Scan(&rows).Error; err != nil {
		logger.WithContext(ctx).Errorf("[DashboardRealtimeHandler] query order activity network candidates error: %v", err)
		return []dashboardActivityNetworkCandidate{}
	}
	candidates := make([]dashboardActivityNetworkCandidate, 0, len(rows))
	for _, item := range rows {
		candidates = append(candidates, dashboardActivityNetworkCandidate{
			UserID:    item.UserID,
			IP:        item.IP,
			Location:  dashboardActivityLocation(svcCtx, item.IP, item.Location, item.CountryShort, item.Region),
			CreatedAt: item.CreatedAt,
		})
	}
	return candidates
}

func uniqueDashboardInt64s(values []int64) []int64 {
	if len(values) == 0 {
		return values
	}
	result := values[:0]
	var seen map[int64]struct{}
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if seen != nil {
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			result = append(result, value)
			continue
		}
		duplicate := false
		for _, existing := range result {
			if existing == value {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		if len(result) == dashboardSmallInt64SetLimit {
			seen = make(map[int64]struct{}, dashboardInt64InitialCap(len(values)))
			for _, existing := range result {
				seen[existing] = struct{}{}
			}
			seen[value] = struct{}{}
		}
		result = append(result, value)
	}
	return result
}

const dashboardSmallInt64SetLimit = 8

func dashboardInt64InitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > dashboardSmallInt64SetLimit+1 {
		return dashboardSmallInt64SetLimit + 1
	}
	return capHint
}

func dashboardInt64LookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func queryActivityUsers(c *gin.Context, svcCtx *svc.ServiceContext, orders []dashboardOrderActivity, cache map[int64]string) map[int64]string {
	var ids []int64
	for _, item := range orders {
		if item.UserId > 0 {
			ids = append(ids, item.UserId)
		}
	}
	return queryUserLabelsCached(c, svcCtx, ids, cache)
}

func queryActivityMembershipPlans(c *gin.Context, svcCtx *svc.ServiceContext, orders []dashboardOrderActivity) map[int64]string {
	var ids []int64
	for _, item := range orders {
		if item.Type == orderTypeMembership && item.SubscribeId > 0 {
			ids = append(ids, item.SubscribeId)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	ids = uniqueDashboardInt64s(ids)
	if len(ids) == 0 {
		return nil
	}
	var plans []membership.Plan
	if err := svcCtx.DB.WithContext(c.Request.Context()).
		Model(&membership.Plan{}).
		Select("id", "name").
		Where("id IN ?", ids).
		Find(&plans).Error; err != nil {
		logger.WithContext(c.Request.Context()).Errorf("[DashboardRealtimeHandler] query membership plan names error: %v", err)
		return nil
	}
	result := make(map[int64]string, len(ids))
	for _, item := range plans {
		result[item.Id] = item.Name
	}
	return result
}

func queryUserSubscribeNames(c *gin.Context, svcCtx *svc.ServiceContext, ids []int64) map[int64]string {
	if len(ids) == 0 {
		return nil
	}
	ids = uniqueDashboardInt64s(ids)
	if len(ids) == 0 {
		return nil
	}
	type row struct {
		ID   int64
		Name string
	}
	var rows []row
	if err := svcCtx.DB.WithContext(c.Request.Context()).
		Table("user_subscribe").
		Select("user_subscribe.id, subscribe.name").
		Joins("LEFT JOIN subscribe ON subscribe.id = user_subscribe.subscribe_id").
		Where("user_subscribe.id IN ?", ids).
		Scan(&rows).Error; err != nil {
		logger.WithContext(c.Request.Context()).Errorf("[DashboardRealtimeHandler] query user subscribe names error: %v", err)
		return nil
	}
	var result map[int64]string
	for _, item := range rows {
		if item.Name != "" {
			if result == nil {
				result = make(map[int64]string, len(ids))
			}
			result[item.ID] = item.Name
		}
	}
	return result
}

func queryUserLabels(c *gin.Context, svcCtx *svc.ServiceContext, ids []int64) map[int64]string {
	return queryUserLabelsCached(c, svcCtx, ids, nil)
}

func queryUserLabelsCached(c *gin.Context, svcCtx *svc.ServiceContext, ids []int64, cache map[int64]string) map[int64]string {
	if len(ids) == 0 {
		return nil
	}
	ids = uniqueDashboardInt64s(ids)
	if len(ids) == 0 {
		return nil
	}
	if cache == nil {
		cache = make(map[int64]string, len(ids))
	}
	var result map[int64]string
	var missingIDs []int64
	for _, id := range ids {
		if label, ok := cache[id]; ok {
			if label != "" {
				if result == nil {
					result = make(map[int64]string, len(ids))
				}
				result[id] = label
			}
			continue
		}
		cache[id] = ""
		missingIDs = append(missingIDs, id)
	}
	if len(missingIDs) == 0 {
		return result
	}
	var authMethods []*user.AuthMethods
	if svcCtx.UserModel != nil {
		var err error
		authMethods, err = svcCtx.UserModel.FindUserAuthMethodsByType(c.Request.Context(), "email", missingIDs, false)
		if err != nil {
			logger.WithContext(c.Request.Context()).Errorf("[DashboardRealtimeHandler] query activity users error: %v", err)
			return result
		}
	} else {
		var rows []user.AuthMethods
		if err := svcCtx.DB.WithContext(c.Request.Context()).
			Model(&user.AuthMethods{}).
			Select("user_id", "auth_identifier").
			Where("user_id IN ? AND auth_type = ?", missingIDs, "email").
			Find(&rows).Error; err != nil {
			logger.WithContext(c.Request.Context()).Errorf("[DashboardRealtimeHandler] query activity users error: %v", err)
			return result
		}
		authMethods = make([]*user.AuthMethods, 0, len(rows))
		for index := range rows {
			authMethods = append(authMethods, &rows[index])
		}
	}
	for _, item := range authMethods {
		if item == nil {
			continue
		}
		if _, ok := cache[item.UserId]; !ok {
			continue
		}
		if cache[item.UserId] == "" {
			cache[item.UserId] = dashboardDisplayIdentifier(item.AuthIdentifier)
		}
		if cache[item.UserId] != "" {
			if result == nil {
				result = make(map[int64]string, len(ids))
			}
			result[item.UserId] = cache[item.UserId]
		}
	}
	return result
}

func activityTime(fallback time.Time, timestamp int64) int64 {
	if timestamp > 0 {
		return timestamp
	}
	return fallback.UnixMilli()
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func displayUser(userID int64, label string) string {
	if label != "" {
		return label
	}
	if userID > 0 {
		return dashboardPrefixedInt("User #", userID)
	}
	return "Unknown user"
}

func dashboardDisplayIP(ip string) string {
	return privacy.MaskIP(ip)
}

func dashboardDisplayIdentifier(identifier string) string {
	return privacy.MaskIdentifier(identifier)
}

func displayServer(serverID int64, label string) string {
	if strings.TrimSpace(label) != "" {
		return label
	}
	if serverID > 0 {
		return dashboardPrefixedInt("Server #", serverID)
	}
	return "Unknown server"
}

func displaySubscription(subscriptionID int64, label string) string {
	label = strings.TrimSpace(label)
	if label != "" {
		return label
	}
	return dashboardPrefixedInt("Subscription #", subscriptionID)
}

func dashboardPrefixedInt(prefix string, id int64) string {
	value := make([]byte, 0, len(prefix)+20)
	value = append(value, prefix...)
	value = strconv.AppendInt(value, id, 10)
	return string(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func activityDetail(values ...string) string {
	var first string
	var builder strings.Builder
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			if first == "" && builder.Len() == 0 {
				first = value
				continue
			}
			if builder.Len() == 0 {
				builder.Grow(len(first) + len(value) + len(" · "))
				builder.WriteString(first)
			}
			builder.WriteString(" · ")
			builder.WriteString(value)
		}
	}
	if builder.Len() > 0 {
		return builder.String()
	}
	return first
}

func dashboardActivityLocation(svcCtx *svc.ServiceContext, ip, location, countryShort, region string) string {
	location = strings.TrimSpace(location)
	if location != "" {
		return location
	}
	return dashboardIPLocation(svcCtx, ip, countryShort, region)
}

func dashboardIPLocation(svcCtx *svc.ServiceContext, ip, countryShort, region string) string {
	countryShort = strings.TrimSpace(countryShort)
	region = strings.TrimSpace(region)
	if countryShort == "" && region == "" && svcCtx != nil && svcCtx.GeoIP != nil {
		location := svcCtx.GeoIP.Lookup(ip)
		countryShort = location.CountryShort
		region = location.Region
	}
	return activityDetail(countryShort, region)
}

func orderStatusText(status uint8) string {
	switch status {
	case 1:
		return "pending"
	case 2:
		return "paid"
	case 3:
		return "closed"
	case 4:
		return "failed"
	case 5:
		return "finished"
	default:
		return "unknown"
	}
}

func orderActivityTypeTitle(orderType uint8) (string, string) {
	switch orderType {
	case orderTypeSubscribe:
		return "purchase", "Subscription purchased"
	case orderTypeRenewal:
		return "renewal", "Subscription renewed"
	case orderTypeResetTraffic:
		return "reset_traffic", "Traffic reset"
	case orderTypeRecharge:
		return "recharge", "Balance recharged"
	case orderTypeMembership:
		return "membership", "Membership opened"
	default:
		return "order", "Order updated"
	}
}

func orderActivityProduct(item dashboardOrderActivity, planNames map[int64]string) string {
	if item.ProductName != "" {
		return item.ProductName
	}
	switch item.Type {
	case orderTypeMembership:
		return firstNonEmpty(planNames[item.SubscribeId], "Membership card")
	case orderTypeRecharge:
		return "Balance recharge"
	default:
		if item.SubscribeId > 0 {
			return dashboardPrefixedInt("Subscribe #", item.SubscribeId)
		}
		return "Order"
	}
}

func validateDashboardRealtimeToken(c *gin.Context, svcCtx *svc.ServiceContext) bool {
	token := realtimeWebSocketToken(c)
	if token == "" {
		return false
	}
	claims, err := jwt.ParseJwtToken(token, svcCtx.Config.JwtAuth.AccessSecret)
	if err != nil {
		logger.WithContext(c.Request.Context()).Debug("[DashboardRealtimeHandler] ParseJwtToken", logger.Field("error", err.Error()))
		return false
	}
	userIDFloat, ok := claims["UserId"].(float64)
	if !ok {
		return false
	}
	sessionID, ok := claims["SessionId"].(string)
	if !ok || sessionID == "" {
		return false
	}
	key := config.SessionCacheKey(sessionID)
	value, err := svcCtx.Redis.Get(c.Request.Context(), key).Result()
	if err != nil || value != config.SessionUserValue(int64(userIDFloat)) {
		return false
	}
	userInfo, err := svcCtx.UserModel.FindOneForAdminCheck(c.Request.Context(), int64(userIDFloat))
	if err != nil || userInfo.DeletedAt.Valid || userInfo.IsAdmin == nil || !*userInfo.IsAdmin || (userInfo.Enable != nil && !*userInfo.Enable) {
		return false
	}
	return true
}

func realtimeWebSocketToken(c *gin.Context) string {
	if authorization := strings.TrimSpace(c.GetHeader("Authorization")); authorization != "" {
		if scheme, token, ok := strings.Cut(authorization, " "); ok && strings.EqualFold(scheme, "Bearer") {
			return strings.TrimSpace(token)
		}
	}
	protocols := websocket.Subprotocols(c.Request)
	for index := 0; index+1 < len(protocols); index++ {
		if protocols[index] == "alpha-auth" {
			return strings.TrimSpace(protocols[index+1])
		}
	}
	return ""
}
