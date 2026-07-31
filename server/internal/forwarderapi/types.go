package forwarderapi

import "github.com/neibniz/alpha/server/internal/realtime"

type RegisterRequest struct {
	InstanceID string `json:"instance_id"`
	BootID     string `json:"boot_id"`
	Name       string `json:"name"`
	Hostname   string `json:"hostname"`
	PublicIP   string `json:"public_ip"`
	Version    string `json:"version"`
	StartedAt  int64  `json:"started_at"`
}

type RegisterResponse struct {
	InstanceID     string `json:"instance_id"`
	Token          string `json:"token"`
	StatusInterval int64  `json:"status_interval"`
	StatusTTL      int64  `json:"status_ttl"`
}

type Rule struct {
	RuleID     string `json:"rule_id"`
	ListenIP   string `json:"listen_ip,omitempty"`
	ListenPort uint16 `json:"listen_port"`
	TargetHost string `json:"target_host"`
	TargetPort uint16 `json:"target_port"`
	NodeID     int64  `json:"node_id,omitempty"`
	NodeName   string `json:"node_name,omitempty"`
	ServerID   int64  `json:"server_id,omitempty"`
	ServerName string `json:"server_name,omitempty"`
	Inbound    string `json:"inbound,omitempty"`
}

type ConfigResponse struct {
	InstanceID  string   `json:"instance_id"`
	PublicIP    string   `json:"public_ip"`
	Version     string   `json:"version"`
	Hash        string   `json:"hash"`
	GeneratedAt int64    `json:"generated_at"`
	Rules       []Rule   `json:"rules"`
	Warnings    []string `json:"warnings"`
}

type StatusRequest struct {
	BootID              string                         `json:"boot_id"`
	Version             string                         `json:"version"`
	Uptime              int64                          `json:"uptime"`
	ActiveConnections   int64                          `json:"active_connections"`
	AcceptedConnections uint64                         `json:"accepted_connections"`
	FailedConnections   uint64                         `json:"failed_connections"`
	BlockedConnections  uint64                         `json:"blocked_connections"`
	UploadBps           int64                          `json:"upload_bps"`
	DownloadBps         int64                          `json:"download_bps"`
	UploadBytes         uint64                         `json:"upload_bytes"`
	DownloadBytes       uint64                         `json:"download_bytes"`
	DesiredVersion      string                         `json:"desired_version"`
	DesiredHash         string                         `json:"desired_hash"`
	AppliedVersion      string                         `json:"applied_version"`
	AppliedHash         string                         `json:"applied_hash"`
	ApplyStatus         string                         `json:"apply_status"`
	ApplyError          string                         `json:"apply_error"`
	Rules               []realtime.ForwarderRuleStatus `json:"rules"`
}

type ConnectionEvent struct {
	EventID               string `json:"event_id"`
	ConnectionID          string `json:"connection_id"`
	BootID                string `json:"boot_id"`
	EventTime             int64  `json:"event_time"`
	EventType             string `json:"event_type"`
	SourceIP              string `json:"source_ip"`
	SourcePort            uint16 `json:"source_port"`
	ListenIP              string `json:"listen_ip"`
	ListenPort            uint16 `json:"listen_port"`
	ConfiguredTargetHost  string `json:"configured_target_host"`
	ConfiguredTargetPort  uint16 `json:"configured_target_port"`
	ResolvedTargetIP      string `json:"resolved_target_ip"`
	ResolvedTargetPort    uint16 `json:"resolved_target_port"`
	ForwarderOutboundIP   string `json:"forwarder_outbound_ip"`
	ForwarderOutboundPort uint16 `json:"forwarder_outbound_port"`
	RuleID                string `json:"rule_id"`
	ServerID              int64  `json:"server_id"`
	NodeID                int64  `json:"node_id"`
	Inbound               string `json:"inbound"`
	ConfigVersion         string `json:"config_version"`
	ConnectedAt           int64  `json:"connected_at"`
	ClosedAt              int64  `json:"closed_at"`
	DurationMS            int64  `json:"duration_ms"`
	UploadBytes           uint64 `json:"upload_bytes"`
	DownloadBytes         uint64 `json:"download_bytes"`
	CloseInitiator        string `json:"close_initiator"`
	CloseReason           string `json:"close_reason"`
	Error                 string `json:"error"`
}

type ConnectionEventsRequest struct {
	Events []ConnectionEvent `json:"events"`
}

type InstanceView struct {
	ID             int64                    `json:"id"`
	InstanceID     string                   `json:"instance_id"`
	BootID         string                   `json:"boot_id"`
	Name           string                   `json:"name"`
	Hostname       string                   `json:"hostname"`
	PublicIP       string                   `json:"public_ip"`
	Version        string                   `json:"version"`
	StartedAt      int64                    `json:"started_at"`
	FirstSeenAt    int64                    `json:"first_seen_at"`
	LastSeenAt     int64                    `json:"last_seen_at"`
	DesiredVersion string                   `json:"desired_version"`
	DesiredHash    string                   `json:"desired_hash"`
	AppliedVersion string                   `json:"applied_version"`
	AppliedHash    string                   `json:"applied_hash"`
	ApplyStatus    string                   `json:"apply_status"`
	ApplyError     string                   `json:"apply_error"`
	Status         realtime.ForwarderStatus `json:"status"`
}

type ConnectionEventView struct {
	EventID               string `json:"event_id"`
	ConnectionID          string `json:"connection_id"`
	InstanceID            string `json:"instance_id"`
	BootID                string `json:"boot_id"`
	EventTime             int64  `json:"event_time"`
	EventType             string `json:"event_type"`
	SourceIP              string `json:"source_ip"`
	SourcePort            uint16 `json:"source_port"`
	ListenIP              string `json:"listen_ip"`
	ListenPort            uint16 `json:"listen_port"`
	ConfiguredTargetHost  string `json:"configured_target_host"`
	ConfiguredTargetPort  uint16 `json:"configured_target_port"`
	ResolvedTargetIP      string `json:"resolved_target_ip"`
	ResolvedTargetPort    uint16 `json:"resolved_target_port"`
	ForwarderOutboundIP   string `json:"forwarder_outbound_ip"`
	ForwarderOutboundPort uint16 `json:"forwarder_outbound_port"`
	RuleID                string `json:"rule_id"`
	ServerID              int64  `json:"server_id"`
	NodeID                int64  `json:"node_id"`
	Inbound               string `json:"inbound"`
	ConfigVersion         string `json:"config_version"`
	ConnectedAt           int64  `json:"connected_at"`
	ClosedAt              int64  `json:"closed_at"`
	DurationMS            int64  `json:"duration_ms"`
	UploadBytes           uint64 `json:"upload_bytes"`
	DownloadBytes         uint64 `json:"download_bytes"`
	CloseInitiator        string `json:"close_initiator"`
	CloseReason           string `json:"close_reason"`
	Error                 string `json:"error"`
}

type ConnectionPage struct {
	List       []ConnectionEventView `json:"list"`
	NextCursor string                `json:"next_cursor"`
	HasMore    bool                  `json:"has_more"`
}
