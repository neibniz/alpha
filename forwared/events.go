package main

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
