package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/neibniz/alpha/forwared/logger"
	"github.com/google/uuid"
)

var Version = "dev"

type agentIdentity struct {
	InstanceID string `json:"instance_id"`
}

type registerRequest struct {
	InstanceID string `json:"instance_id"`
	BootID     string `json:"boot_id"`
	Name       string `json:"name"`
	Hostname   string `json:"hostname"`
	PublicIP   string `json:"public_ip"`
	Version    string `json:"version"`
	StartedAt  int64  `json:"started_at"`
}

type registerResponse struct {
	InstanceID     string `json:"instance_id"`
	Token          string `json:"token"`
	StatusInterval int64  `json:"status_interval"`
	StatusTTL      int64  `json:"status_ttl"`
}

type apiResponse[T any] struct {
	Code uint32 `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type statusPayload struct {
	BootID              string       `json:"boot_id"`
	Version             string       `json:"version"`
	Uptime              int64        `json:"uptime"`
	ActiveConnections   int64        `json:"active_connections"`
	AcceptedConnections uint64       `json:"accepted_connections"`
	FailedConnections   uint64       `json:"failed_connections"`
	BlockedConnections  uint64       `json:"blocked_connections"`
	UploadBps           int64        `json:"upload_bps"`
	DownloadBps         int64        `json:"download_bps"`
	UploadBytes         uint64       `json:"upload_bytes"`
	DownloadBytes       uint64       `json:"download_bytes"`
	DesiredVersion      string       `json:"desired_version"`
	DesiredHash         string       `json:"desired_hash"`
	AppliedVersion      string       `json:"applied_version"`
	AppliedHash         string       `json:"applied_hash"`
	ApplyStatus         string       `json:"apply_status"`
	ApplyError          string       `json:"apply_error"`
	Rules               []RuleStatus `json:"rules"`
}

type Agent struct {
	cfg       *Config
	client    *http.Client
	identity  agentIdentity
	bootID    string
	startedAt time.Time
	metrics   *Metrics
	uploader  *EventUploader
	ctx       context.Context
	cancel    context.CancelFunc

	mu             sync.RWMutex
	token          string
	publicIP       string
	statusInterval time.Duration
	table          *ForwaredTable
	desiredVersion string
	desiredHash    string
	applyStatus    string
	applyError     string
}

func NewAgent(cfg *Config) (*Agent, error) {
	identity, err := loadIdentity(cfg.StateFile)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	agent := &Agent{
		cfg:            cfg,
		client:         &http.Client{Timeout: 15 * time.Second},
		identity:       identity,
		bootID:         uuid.NewString(),
		startedAt:      time.Now(),
		metrics:        NewMetrics(),
		ctx:            ctx,
		cancel:         cancel,
		statusInterval: time.Duration(cfg.StatusInterval) * time.Second,
		table:          &ForwaredTable{ForwaredItems: []ForwaredItem{}},
		applyStatus:    "pending",
	}
	agent.uploader = NewEventUploader(cfg, agent.sendEvents)
	return agent, nil
}

func (a *Agent) Start() error {
	publicIP, err := discoverPublicIP(a.ctx, a.cfg, a.client)
	if err != nil {
		return err
	}
	if err := a.register(a.ctx, publicIP); err != nil {
		return err
	}
	if err := a.uploader.Start(); err != nil {
		return err
	}
	go a.statusLoop()
	go a.registrationRefreshLoop()
	return nil
}

func (a *Agent) Stop() {
	a.cancel()
	a.uploader.Stop()
}

func (a *Agent) Metrics() *Metrics {
	return a.metrics
}

func (a *Agent) Enqueue(event ConnectionEvent) {
	a.uploader.Enqueue(event)
}

func (a *Agent) InstanceID() string {
	return a.identity.InstanceID
}

func (a *Agent) BootID() string {
	return a.bootID
}

func (a *Agent) PublicIP() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.publicIP
}

func (a *Agent) ConfigVersion() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.table.Version
}

func (a *Agent) SetConfigState(desired, applied *ForwaredTable, status string, applyErr error) {
	a.mu.Lock()
	if desired != nil {
		a.desiredVersion = desired.Version
		a.desiredHash = desired.Hash
	}
	if applied != nil {
		copyTable := *applied
		copyTable.ForwaredItems = append([]ForwaredItem(nil), applied.ForwaredItems...)
		copyTable.Warnings = append([]string(nil), applied.Warnings...)
		a.table = &copyTable
	}
	a.applyStatus = status
	a.applyError = ""
	if applyErr != nil {
		a.applyError = applyErr.Error()
	}
	a.mu.Unlock()
}

func (a *Agent) FetchConfig(ctx context.Context) (*ForwaredTable, error) {
	request, err := a.authenticatedRequest(ctx, http.MethodGet, "/v2/forwarder/config", nil)
	if err != nil {
		return nil, err
	}
	response, err := a.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config request returned %s", response.Status)
	}
	var envelope apiResponse[ForwaredTable]
	if err := decodeLimitedJSON(response.Body, &envelope); err != nil {
		return nil, err
	}
	if envelope.Code != http.StatusOK {
		return nil, fmt.Errorf("config request failed: %s", envelope.Msg)
	}
	if envelope.Data.ForwaredItems == nil {
		envelope.Data.ForwaredItems = []ForwaredItem{}
	}
	return &envelope.Data, nil
}

func (a *Agent) register(ctx context.Context, publicIP string) error {
	hostname, _ := os.Hostname()
	name := strings.TrimSpace(a.cfg.Name)
	if name == "" {
		name = hostname
	}
	payload := registerRequest{
		InstanceID: a.identity.InstanceID,
		BootID:     a.bootID,
		Name:       name,
		Hostname:   hostname,
		PublicIP:   publicIP,
		Version:    Version,
		StartedAt:  a.startedAt.UnixMilli(),
	}
	request, err := a.jsonRequest(ctx, http.MethodPost, "/v2/forwarder/register", payload)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(a.cfg.SecretKey))
	response, err := a.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("register returned %s", response.Status)
	}
	var envelope apiResponse[registerResponse]
	if err := decodeLimitedJSON(response.Body, &envelope); err != nil {
		return err
	}
	if envelope.Code != http.StatusOK || strings.TrimSpace(envelope.Data.Token) == "" {
		return fmt.Errorf("register failed: %s", envelope.Msg)
	}
	a.mu.Lock()
	a.token = envelope.Data.Token
	a.publicIP = publicIP
	if envelope.Data.StatusInterval > 0 {
		a.statusInterval = time.Duration(envelope.Data.StatusInterval) * time.Second
	}
	a.mu.Unlock()
	return nil
}

func (a *Agent) statusLoop() {
	for {
		a.sendStatus()
		a.mu.RLock()
		interval := a.statusInterval
		a.mu.RUnlock()
		timer := time.NewTimer(interval)
		select {
		case <-timer.C:
		case <-a.ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (a *Agent) sendStatus() {
	a.mu.RLock()
	table := *a.table
	table.ForwaredItems = append([]ForwaredItem(nil), a.table.ForwaredItems...)
	desiredVersion := a.desiredVersion
	desiredHash := a.desiredHash
	status := a.applyStatus
	applyError := a.applyError
	a.mu.RUnlock()
	metrics := a.metrics.Snapshot(table.ForwaredItems)
	payload := statusPayload{
		BootID:              a.bootID,
		Version:             Version,
		Uptime:              int64(time.Since(a.startedAt).Seconds()),
		ActiveConnections:   metrics.ActiveConnections,
		AcceptedConnections: metrics.AcceptedConnections,
		FailedConnections:   metrics.FailedConnections,
		BlockedConnections:  metrics.BlockedConnections,
		UploadBps:           metrics.UploadBps,
		DownloadBps:         metrics.DownloadBps,
		UploadBytes:         metrics.UploadBytes,
		DownloadBytes:       metrics.DownloadBytes,
		DesiredVersion:      desiredVersion,
		DesiredHash:         desiredHash,
		AppliedVersion:      table.Version,
		AppliedHash:         table.Hash,
		ApplyStatus:         status,
		ApplyError:          applyError,
		Rules:               metrics.Rules,
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	request, err := a.authenticatedRequest(ctx, http.MethodPost, "/v2/forwarder/status", payload)
	if err != nil {
		return
	}
	response, err := a.client.Do(request)
	if err != nil {
		logger.Logger.Warn("Report forwarder status failed", "error", err)
		return
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		logger.Logger.Warn("Report forwarder status returned non-success", "status", response.Status)
	}
}

func (a *Agent) registrationRefreshLoop() {
	interval := time.Duration(a.cfg.FetchIPInterval) * time.Second
	if interval <= 0 {
		interval = 2 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
			publicIP, err := discoverPublicIP(ctx, a.cfg, a.client)
			if err == nil && publicIP != a.PublicIP() {
				err = a.register(ctx, publicIP)
			}
			cancel()
			if err != nil {
				logger.Logger.Warn("Refresh forwarder registration failed", "error", err)
			}
		case <-a.ctx.Done():
			return
		}
	}
}

func (a *Agent) sendEvents(ctx context.Context, events []ConnectionEvent) error {
	payload, err := json.Marshal(struct {
		Events []ConnectionEvent `json:"events"`
	}{Events: events})
	if err != nil {
		return err
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(payload); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	request, err := a.authenticatedRequestBody(
		ctx,
		http.MethodPost,
		"/v2/forwarder/events",
		bytes.NewReader(compressed.Bytes()),
	)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Encoding", "gzip")
	response, err := a.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("event upload returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (a *Agent) authenticatedRequest(ctx context.Context, method, path string, payload any) (*http.Request, error) {
	request, err := a.jsonRequest(ctx, method, path, payload)
	if err != nil {
		return nil, err
	}
	a.setAuthenticationHeaders(request)
	return request, nil
}

func (a *Agent) authenticatedRequestBody(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	endpoint := strings.TrimRight(a.cfg.BaseURL, "/") + path
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	a.setAuthenticationHeaders(request)
	return request, nil
}

func (a *Agent) setAuthenticationHeaders(request *http.Request) {
	a.mu.RLock()
	token := a.token
	a.mu.RUnlock()
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-Alpha-Forwarder-ID", a.identity.InstanceID)
}

func (a *Agent) jsonRequest(ctx context.Context, method, path string, payload any) (*http.Request, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}
	endpoint := strings.TrimRight(a.cfg.BaseURL, "/") + path
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	return request, nil
}

func loadIdentity(path string) (agentIdentity, error) {
	payload, err := os.ReadFile(path)
	if err == nil {
		var identity agentIdentity
		if json.Unmarshal(payload, &identity) == nil {
			if _, parseErr := uuid.Parse(identity.InstanceID); parseErr == nil {
				return identity, nil
			}
		}
	}
	if err != nil && !os.IsNotExist(err) {
		return agentIdentity{}, err
	}
	identity := agentIdentity{InstanceID: uuid.NewString()}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return agentIdentity{}, err
	}
	payload, err = json.Marshal(identity)
	if err != nil {
		return agentIdentity{}, err
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, payload, 0o600); err != nil {
		return agentIdentity{}, err
	}
	if err := os.Rename(temp, path); err != nil {
		return agentIdentity{}, err
	}
	return identity, nil
}

func discoverPublicIP(ctx context.Context, cfg *Config, client *http.Client) (string, error) {
	if value := strings.TrimSpace(cfg.PublicIP); value != "" {
		if net.ParseIP(value) == nil {
			return "", fmt.Errorf("configured public_ip is invalid")
		}
		return value, nil
	}
	endpoint := strings.TrimSpace(cfg.PublicIPURL)
	if endpoint == "" {
		endpoint = "https://checkip.amazonaws.com"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("public IP endpoint returned %s", response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 128))
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(body))
	if net.ParseIP(value) == nil {
		return "", fmt.Errorf("public IP endpoint returned invalid address")
	}
	return value, nil
}

func decodeLimitedJSON(reader io.Reader, target any) error {
	decoder := json.NewDecoder(io.LimitReader(reader, 4<<20))
	return decoder.Decode(target)
}
