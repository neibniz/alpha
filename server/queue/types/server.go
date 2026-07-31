package types

import internaltypes "github.com/neibniz/alpha/server/internal/types"

const (
	ForthwithTrafficStatistics    = "forthwith:traffic:statistics"
	ForthwithTrafficStatisticsV2  = "forthwith:traffic:statistics:v2"
	ForthwithTrafficAnalyticsSync = "forthwith:traffic:analytics-sync"
)

type UserTraffic = internaltypes.UserTraffic
type TrafficReport = internaltypes.TrafficReport

type TrafficStatistics struct {
	ServerId int64           `json:"server_id"`
	Protocol string          `json:"protocol"`
	Reports  []TrafficReport `json:"reports,omitempty"`
	Logs     []UserTraffic   `json:"logs,omitempty"`
}

type NodeStatus struct {
	OnlineUsers []OnlineUser `json:"online_users"`
	Status      ServerStatus `json:"status"`
	LastAt      int64        `json:"last_at"`
}

type OnlineUser struct {
	UID int64  `json:"uid"`
	IP  string `json:"ip"`
}

type ServerStatus struct {
	Cpu       float64 `json:"cpu"`
	Mem       float64 `json:"mem"`
	Disk      float64 `json:"disk"`
	UpdatedAt int64   `json:"updated_at"`
}

type ServerTrafficCount struct {
	ServerId  int64  `json:"server_id"`
	Name      string `json:"name"`
	Today     int64  `json:"today"`
	Yesterday int64  `json:"yesterday"`
}
