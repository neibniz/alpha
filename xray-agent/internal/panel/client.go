package panel

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/neibniz/alpha/xray-agent/internal/config"
)

type Client struct {
	cfg                  config.PanelConfig
	httpClient           *http.Client
	trafficProtobuf      protocolCapability
	onlineUsersProtobuf  protocolCapability
	accessEventsProtobuf protocolCapability
}

type Response struct {
	Code uint32          `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data,omitempty"`
}

type XrayConfig struct {
	Inbounds  []interface{} `json:"inbounds"`
	Outbounds []interface{} `json:"outbounds"`
	DNS       interface{}   `json:"dns,omitempty"`
	Routing   interface{}   `json:"routing,omitempty"`
	Geodata   interface{}   `json:"geodata,omitempty"`
}

type XrayUsersResponse struct {
	Version string     `json:"version"`
	Full    bool       `json:"full"`
	Users   []XrayUser `json:"users"`
}

type XrayUser struct {
	UserSubscribeID int64    `json:"user_subscribe_id"`
	UserID          int64    `json:"user_id"`
	Email           string   `json:"email"`
	UUID            string   `json:"uuid"`
	Password        string   `json:"password"`
	Level           int      `json:"level"`
	Inbounds        []string `json:"inbounds"`
	SpeedLimit      int64    `json:"speed_limit,omitempty"`
	DeviceLimit     int64    `json:"device_limit,omitempty"`
	Enabled         bool     `json:"enabled"`
	ExpiredAt       int64    `json:"expired_at,omitempty"`
}

type UserTraffic struct {
	SID      int64 `json:"uid"`
	Upload   int64 `json:"upload"`
	Download int64 `json:"download"`
}

type TrafficReport struct {
	ReportID        string        `json:"report_id"`
	SampledAt       int64         `json:"sampled_at"`
	IntervalSeconds int64         `json:"interval_seconds"`
	Sequence        int64         `json:"sequence"`
	Traffic         []UserTraffic `json:"traffic"`
}

type OnlineUsersUpdate struct {
	Mode       string  `json:"mode"`
	InstanceID string  `json:"instance_id"`
	Seq        int64   `json:"seq"`
	Active     []int64 `json:"active,omitempty"`
	Inactive   []int64 `json:"inactive,omitempty"`
	TTLSeconds int64   `json:"ttl_seconds"`
}

type AccessEvent struct {
	EventID         string `json:"event_id"`
	Timestamp       int64  `json:"timestamp"`
	UserSubscribeID int64  `json:"user_subscribe_id,omitempty"`
	SourceIP        string `json:"source_ip,omitempty"`
	SourcePort      uint16 `json:"source_port,omitempty"`
	DestinationHost string `json:"destination_host,omitempty"`
	DestinationIP   string `json:"destination_ip,omitempty"`
	DestinationPort uint16 `json:"destination_port,omitempty"`
	Network         string `json:"network,omitempty"`
	Status          string `json:"status"`
	OutboundTag     string `json:"outbound_tag,omitempty"`
	Reason          string `json:"reason,omitempty"`
	ParserVersion   uint16 `json:"parser_version"`
}

type AccessPostResult struct {
	Accepted         int               `json:"accepted"`
	Rejected         int               `json:"rejected"`
	IPLimitDecisions []IPLimitDecision `json:"ip_limit_decisions,omitempty"`
}

type IPLimitDecision struct {
	EventID         string `json:"event_id"`
	UserSubscribeID int64  `json:"user_subscribe_id"`
	SourceNetwork   string `json:"source_network"`
	Blocked         bool   `json:"blocked"`
	ExpiresAt       int64  `json:"expires_at"`
	Active          int64  `json:"active"`
	Limit           int64  `json:"limit"`
}

func NewClient(cfg config.PanelConfig) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *Client) FetchXrayConfig(ctx context.Context) (XrayConfig, []byte, error) {
	endpoint, err := c.url(fmt.Sprintf("/v2/server/%d/xray-config", c.cfg.ServerID), nil)
	if err != nil {
		return XrayConfig{}, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return XrayConfig{}, nil, err
	}
	c.authenticate(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return XrayConfig{}, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return XrayConfig{}, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return XrayConfig{}, nil, fmt.Errorf("panel returned http %d: %s", resp.StatusCode, string(body))
	}
	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return XrayConfig{}, nil, err
	}
	if result.Code != 200 {
		return XrayConfig{}, nil, fmt.Errorf("panel returned code %d: %s", result.Code, result.Msg)
	}
	var cfg XrayConfig
	if err := json.Unmarshal(result.Data, &cfg); err != nil {
		return XrayConfig{}, nil, err
	}
	raw, err := json.Marshal(cfg)
	return cfg, raw, err
}

func (c *Client) FetchXrayUsers(ctx context.Context, version string) (XrayUsersResponse, error) {
	endpoint, err := c.url(fmt.Sprintf("/v2/server/%d/xray-users", c.cfg.ServerID), nil)
	if err != nil {
		return XrayUsersResponse{}, err
	}
	q := endpoint.Query()
	if version != "" {
		q.Set("version", version)
	}
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return XrayUsersResponse{}, err
	}
	c.authenticate(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return XrayUsersResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return XrayUsersResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return XrayUsersResponse{}, fmt.Errorf("panel returned http %d: %s", resp.StatusCode, string(body))
	}
	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return XrayUsersResponse{}, err
	}
	if result.Code != 200 {
		return XrayUsersResponse{}, fmt.Errorf("panel returned code %d: %s", result.Code, result.Msg)
	}
	var users XrayUsersResponse
	if err := json.Unmarshal(result.Data, &users); err != nil {
		return XrayUsersResponse{}, err
	}
	return users, nil
}

func (c *Client) PostStatus(ctx context.Context, payload interface{}) error {
	endpoint, err := c.url("/v1/server/status", nil)
	if err != nil {
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	c.authenticate(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("panel returned http %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) PostTraffic(ctx context.Context, protocol string, traffic []UserTraffic) error {
	if len(traffic) == 0 {
		return nil
	}
	endpoint, err := c.url("/v1/server/push", nil)
	if err != nil {
		return err
	}
	q := endpoint.Query()
	if protocol != "" {
		q.Set("protocol", protocol)
	}
	endpoint.RawQuery = q.Encode()

	data, err := json.Marshal(map[string]interface{}{
		"traffic": traffic,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	c.authenticate(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("panel returned http %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) PostTrafficReports(ctx context.Context, protocol string, reports []TrafficReport) error {
	if len(reports) == 0 {
		return nil
	}
	if handled, err := c.postTrafficReportsProtobuf(ctx, protocol, reports); handled {
		return err
	}
	return c.postTrafficReportsJSON(ctx, protocol, reports)
}

func (c *Client) postTrafficReportsJSON(ctx context.Context, protocol string, reports []TrafficReport) error {
	if len(reports) == 0 {
		return nil
	}
	endpoint, err := c.url("/v1/server/push", nil)
	if err != nil {
		return err
	}
	query := endpoint.Query()
	if protocol != "" {
		query.Set("protocol", protocol)
	}
	endpoint.RawQuery = query.Encode()

	data, err := json.Marshal(struct {
		Reports []TrafficReport `json:"reports"`
	}{Reports: reports})
	if err != nil {
		return err
	}
	requestBody := data
	compressed := false
	if len(data) >= 4<<10 {
		var buffer bytes.Buffer
		writer := gzip.NewWriter(&buffer)
		if _, err := writer.Write(data); err != nil {
			return err
		}
		if err := writer.Close(); err != nil {
			return err
		}
		requestBody = buffer.Bytes()
		compressed = true
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(requestBody))
	if err != nil {
		return err
	}
	c.authenticate(req)
	req.Header.Set("Content-Type", "application/json")
	if compressed {
		req.Header.Set("Content-Encoding", "gzip")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return fmt.Errorf("panel returned http %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) PostOnlineUsers(ctx context.Context, update OnlineUsersUpdate) error {
	if handled, err := c.postOnlineUsersProtobuf(ctx, update); handled {
		return err
	}
	return c.postOnlineUsersJSON(ctx, update)
}

func (c *Client) postOnlineUsersJSON(ctx context.Context, update OnlineUsersUpdate) error {
	endpoint, err := c.url("/v1/server/online-users", nil)
	if err != nil {
		return err
	}
	data, err := json.Marshal(update)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	c.authenticate(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("panel returned http %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) PostAccessEvents(ctx context.Context, events []AccessEvent) (AccessPostResult, error) {
	if len(events) == 0 {
		return AccessPostResult{}, nil
	}
	if result, handled, err := c.postAccessEventsProtobuf(ctx, events); handled {
		return result, err
	}
	return c.postAccessEventsJSON(ctx, events)
}

func (c *Client) postAccessEventsJSON(ctx context.Context, events []AccessEvent) (AccessPostResult, error) {
	if len(events) == 0 {
		return AccessPostResult{}, nil
	}
	endpoint, err := c.url("/v1/server/access-logs", nil)
	if err != nil {
		return AccessPostResult{}, err
	}
	q := endpoint.Query()
	q.Set("protocol", "xray")
	endpoint.RawQuery = q.Encode()

	data, err := json.Marshal(struct {
		Events []AccessEvent `json:"events"`
	}{Events: events})
	if err != nil {
		return AccessPostResult{}, err
	}
	requestBody := data
	compressed := false
	if len(data) >= 4<<10 {
		var buffer bytes.Buffer
		writer := gzip.NewWriter(&buffer)
		if _, err := writer.Write(data); err != nil {
			return AccessPostResult{}, err
		}
		if err := writer.Close(); err != nil {
			return AccessPostResult{}, err
		}
		requestBody = buffer.Bytes()
		compressed = true
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(requestBody))
	if err != nil {
		return AccessPostResult{}, err
	}
	c.authenticate(req)
	req.Header.Set("Content-Type", "application/json")
	if compressed {
		req.Header.Set("Content-Encoding", "gzip")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return AccessPostResult{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPanelResponseBytes+1))
	if err != nil {
		return AccessPostResult{}, err
	}
	if len(body) > maxPanelResponseBytes {
		return AccessPostResult{}, fmt.Errorf("panel response exceeds %d bytes", maxPanelResponseBytes)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return AccessPostResult{}, fmt.Errorf("panel returned http %d: %s", resp.StatusCode, string(body))
	}
	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return AccessPostResult{}, err
	}
	if result.Code != 200 {
		return AccessPostResult{}, fmt.Errorf("panel returned code %d: %s", result.Code, result.Msg)
	}
	if len(bytes.TrimSpace(result.Data)) == 0 || bytes.Equal(bytes.TrimSpace(result.Data), []byte("null")) {
		return AccessPostResult{}, fmt.Errorf("panel returned no access event acknowledgement")
	}
	var report AccessPostResult
	if err := json.Unmarshal(result.Data, &report); err != nil {
		return AccessPostResult{}, err
	}
	if err := validateAccessAcknowledgement(report, len(events)); err != nil {
		return AccessPostResult{}, err
	}
	return report, nil
}

func validateAccessAcknowledgement(result AccessPostResult, expected int) error {
	if result.Accepted < 0 || result.Rejected < 0 {
		return fmt.Errorf("panel returned a negative access event acknowledgement")
	}
	if result.Accepted+result.Rejected != expected {
		return fmt.Errorf("panel acknowledged %d of %d access events", result.Accepted+result.Rejected, expected)
	}
	return nil
}

func (c *Client) DialStatusWS(ctx context.Context) (*websocket.Conn, *http.Response, error) {
	endpoint, err := c.url("/v1/server/status/ws", nil)
	if err != nil {
		return nil, nil, err
	}
	if endpoint.Scheme == "https" {
		endpoint.Scheme = "wss"
	} else {
		endpoint.Scheme = "ws"
	}
	headers := http.Header{}
	token := c.credential()
	headers.Set("Authorization", "Bearer "+token)
	headers.Set("X-Alpha-Server-ID", fmt.Sprintf("%d", c.cfg.ServerID))
	return websocket.DefaultDialer.DialContext(ctx, endpoint.String(), headers)
}

func (c *Client) authenticate(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.credential())
	req.Header.Set("X-Alpha-Server-ID", fmt.Sprintf("%d", c.cfg.ServerID))
}

func (c *Client) credential() string {
	if c.cfg.Token != "" {
		return c.cfg.Token
	}
	return c.cfg.SecretKey
}

func (c *Client) url(path string, values url.Values) (*url.URL, error) {
	base, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	base.Path = strings.TrimRight(base.Path, "/") + path
	if values != nil {
		base.RawQuery = values.Encode()
	}
	return base, nil
}
