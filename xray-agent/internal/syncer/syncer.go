package syncer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/neibniz/alpha/xray-agent/internal/accesslog"
	"github.com/neibniz/alpha/xray-agent/internal/config"
	"github.com/neibniz/alpha/xray-agent/internal/panel"
	"github.com/neibniz/alpha/xray-agent/internal/status"
	"github.com/neibniz/alpha/xray-agent/internal/supervisor"
	"github.com/neibniz/alpha/xray-agent/internal/trafficreport"
	"github.com/neibniz/alpha/xray-agent/internal/xray"
	"github.com/gorilla/websocket"
)

type Runner struct {
	cfg         config.Config
	panel       *panel.Client
	supervisor  *supervisor.Supervisor
	lastHash    string
	userVersion string
	userCache   []panel.XrayUser
	hotUsers    *xray.UserHotReloader
	ipPolicy    *xray.IPLimitPolicy
	runtime     *status.RuntimeState
}

func New(cfg config.Config, panelClient *panel.Client, xraySupervisor *supervisor.Supervisor) *Runner {
	runner := &Runner{
		cfg:        cfg,
		panel:      panelClient,
		supervisor: xraySupervisor,
		hotUsers:   xray.NewUserHotReloader(cfg.Xray.APIAddress),
		runtime:    status.NewRuntimeState(),
	}
	if cfg.Xray.InjectAPI {
		runner.ipPolicy = xray.NewIPLimitPolicy(cfg.Xray.APIAddress)
	}
	return runner
}

func (r *Runner) Run(ctx context.Context) error {
	defer r.supervisor.Stop()

	collector := status.NewCollector(r.cfg, r.supervisor, r.runtime)
	go r.reportStatus(ctx, collector)
	statsCollector := xray.NewGRPCStatsCollector(r.cfg.Xray.APIAddress, r.cfg.Xray.StatsCursorFile)
	defer func() {
		if err := statsCollector.Close(); err != nil {
			log.Printf("close xray stats collector failed: %v", err)
		}
	}()
	trafficSpool, err := trafficreport.Open(
		r.cfg.Report.Traffic.SpoolPath,
		r.cfg.Report.Traffic.StatePath,
		r.cfg.Report.Traffic.MaxSpoolBytes,
	)
	if err != nil {
		return fmt.Errorf("open traffic report spool: %w", err)
	}
	if err := statsCollector.RestoreCursor(trafficSpool.Checkpoint()); err != nil {
		return fmt.Errorf("restore traffic stats cursor: %w", err)
	}
	onlineTracker := xray.NewActiveUserTracker(3 * time.Minute)
	go r.reportTraffic(ctx, statsCollector, onlineTracker, trafficSpool)
	go r.uploadTrafficReports(ctx, trafficSpool)
	go r.sampleXrayTraffic(ctx, statsCollector)
	if r.ipPolicy != nil {
		go r.ipPolicy.Run(ctx)
	}

	if err := r.syncConfig(ctx); err != nil {
		log.Printf("initial config sync failed: %v", err)
	}
	if r.cfg.Report.Access.Enabled && r.cfg.Log.XrayAccessPath != "" {
		var accessCollector *accesslog.Collector
		var err error
		if r.ipPolicy != nil {
			accessCollector, err = accesslog.NewCollector(r.cfg.Server.ServerID, r.cfg.Log.XrayAccessPath, r.cfg.Report.Access, r.panel, r.ipPolicy)
		} else {
			accessCollector, err = accesslog.NewCollector(r.cfg.Server.ServerID, r.cfg.Log.XrayAccessPath, r.cfg.Report.Access, r.panel)
		}
		if err != nil {
			log.Printf("initialize xray access collector failed: %v", err)
		} else {
			go accessCollector.Run(ctx)
		}
	}

	ticker := time.NewTicker(r.cfg.Agent.PullInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.syncConfig(ctx); err != nil {
				log.Printf("config sync failed: %v", err)
			}
		}
	}
}

func (r *Runner) reportTraffic(
	ctx context.Context,
	collector xray.StatsCollector,
	onlineTracker *xray.ActiveUserTracker,
	trafficSpool *trafficreport.Spool,
) {
	ticker := time.NewTicker(r.cfg.Report.TrafficInterval)
	defer ticker.Stop()
	fullTicker := time.NewTicker(5 * time.Minute)
	defer fullTicker.Stop()
	now := time.Now()
	r.runtime.SetOnlineUsers(onlineTracker.Count(now))
	r.postOnlineUsers(ctx, onlineTracker, onlineTracker.Full(now))
	trafficFailures := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-fullTicker.C:
			now := time.Now()
			r.runtime.SetOnlineUsers(onlineTracker.Count(now))
			r.postOnlineUsers(ctx, onlineTracker, onlineTracker.Full(now))
		case <-ticker.C:
			now := time.Now()
			if !r.supervisor.Running() {
				r.runtime.SetOnlineUsers(onlineTracker.Count(now))
				continue
			}
			deltas, err := collector.Collect(ctx)
			if err != nil {
				trafficFailures++
				if trafficFailures == 1 || trafficFailures%5 == 0 {
					log.Printf("collect traffic failed (%d consecutive): %v", trafficFailures, err)
				}
				continue
			}
			trafficFailures = 0
			traffic := make([]panel.UserTraffic, 0, len(deltas))
			activeUsers := make([]int64, 0, len(deltas))
			for _, delta := range deltas {
				if delta.UserSubscribeID <= 0 || (delta.Upload == 0 && delta.Download == 0) {
					continue
				}
				activeUsers = append(activeUsers, delta.UserSubscribeID)
				traffic = append(traffic, panel.UserTraffic{
					SID:      delta.UserSubscribeID,
					Upload:   delta.Upload,
					Download: delta.Download,
				})
			}
			onlineTracker.Observe(activeUsers, now)
			r.runtime.SetOnlineUsers(onlineTracker.Count(now))
			if len(traffic) > 0 {
				_, err := trafficSpool.Append(panel.TrafficReport{
					SampledAt:       now.UnixMilli(),
					IntervalSeconds: int64(r.cfg.Report.TrafficInterval / time.Second),
					Traffic:         traffic,
				}, collector.PendingCursor())
				if err != nil {
					log.Printf("persist traffic report failed: %v", err)
					continue
				}
			}
			if err := collector.Commit(); err != nil {
				log.Printf("commit traffic cursor failed: %v", err)
			}
			if update, ok := onlineTracker.Delta(now); ok {
				r.postOnlineUsers(ctx, onlineTracker, update)
			}
		}
	}
}

func (r *Runner) uploadTrafficReports(ctx context.Context, spool *trafficreport.Spool) {
	pollInterval := time.Second
	if r.cfg.Report.Traffic.FlushInterval < pollInterval {
		pollInterval = r.cfg.Report.Traffic.FlushInterval
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			for spool.ShouldFlush(now, r.cfg.Report.Traffic.BatchSize, r.cfg.Report.Traffic.FlushInterval) {
				reports, nextOffset, err := spool.ReadBatch(r.cfg.Report.Traffic.BatchSize)
				if err != nil {
					log.Printf("read traffic report spool failed: %v", err)
					break
				}
				if len(reports) == 0 {
					break
				}
				if err := r.panel.PostTrafficReports(ctx, "xray", reports); err != nil {
					log.Printf("upload traffic report batch failed: %v", err)
					break
				}
				if err := spool.Acknowledge(nextOffset, len(reports)); err != nil {
					log.Printf("acknowledge traffic report batch failed: %v", err)
					break
				}
				now = time.Now()
			}
		}
	}
}

func (r *Runner) postOnlineUsers(ctx context.Context, tracker *xray.ActiveUserTracker, update xray.OnlineUsersUpdate) {
	if update.Mode == "" {
		return
	}
	err := r.panel.PostOnlineUsers(ctx, panel.OnlineUsersUpdate{
		Mode:       update.Mode,
		InstanceID: update.InstanceID,
		Seq:        update.Seq,
		Active:     update.Active,
		Inactive:   update.Inactive,
		TTLSeconds: update.TTLSeconds,
	})
	if err != nil {
		log.Printf("report online users failed: %v", err)
		return
	}
	tracker.MarkSent(update)
}

func (r *Runner) syncConfig(ctx context.Context) error {
	_, raw, err := r.panel.FetchXrayConfig(ctx)
	if err != nil {
		r.runtime.SetSyncFailed("", err)
		return err
	}
	rawHash := hash(raw)
	r.runtime.SetSyncOK(rawHash)
	r.runtime.SetApplyPending(rawHash)
	structureConfig, err := r.buildRuntimeConfig(raw)
	if err != nil {
		r.runtime.SetApplyFailed("validate_failed", rawHash, err)
		return err
	}
	if err := xray.ValidateRuntimeConfig(structureConfig); err != nil {
		r.runtime.SetApplyFailed("validate_failed", rawHash, err)
		return err
	}
	structureHash := hash(structureConfig)

	users, err := r.panel.FetchXrayUsers(ctx, r.userVersion)
	if err != nil {
		r.runtime.SetApplyFailed("sync_failed", rawHash, err)
		return err
	}
	effectiveUsers, err := r.effectiveUsers(users)
	if err != nil {
		r.runtime.SetApplyFailed("validate_failed", rawHash, err)
		return err
	}
	desiredUsers, err := xray.BuildInboundUsers(raw, effectiveUsers)
	if err != nil {
		r.runtime.SetApplyFailed("validate_failed", rawHash, err)
		return err
	}
	rendered, err := xray.InjectUsers(raw, effectiveUsers)
	if err != nil {
		r.runtime.SetApplyFailed("validate_failed", rawHash, err)
		return err
	}
	runtimeConfig, err := r.buildRuntimeConfig(rendered)
	if err != nil {
		r.runtime.SetApplyFailed("validate_failed", rawHash, err)
		return err
	}
	if err := xray.ValidateRuntimeConfig(runtimeConfig); err != nil {
		r.runtime.SetApplyFailed("validate_failed", rawHash, err)
		return err
	}
	runtimeHash := hash(runtimeConfig)
	if r.ipPolicy != nil {
		if err := r.ipPolicy.SetBaseConfig(runtimeConfig); err != nil {
			r.runtime.SetApplyFailed("validate_failed", rawHash, err)
			return err
		}
		r.ipPolicy.SetUsers(effectiveUsers)
	}

	if structureHash == r.lastHash && r.supervisor.Running() {
		if users.Version == "" || users.Version == r.userVersion {
			if r.ipPolicy != nil {
				if err := r.ipPolicy.Reapply(ctx, false); err != nil {
					r.runtime.SetApplyFailed("apply_failed", rawHash, err)
					return err
				}
			}
			r.runtime.SetApplyOK(rawHash)
			return nil
		}
		if err := r.supervisor.SaveConfig(ctx, runtimeConfig); err != nil {
			r.runtime.SetApplyFailed("apply_failed", rawHash, err)
			return err
		}
		restarted := false
		if err := r.hotUsers.Sync(ctx, desiredUsers); err != nil {
			log.Printf("hot user reload failed, falling back to full restart: %v", err)
			if err := r.supervisor.Apply(ctx, runtimeConfig); err != nil {
				r.runtime.SetApplyFailed("apply_failed", rawHash, err)
				return err
			}
			r.hotUsers.SetSnapshot(desiredUsers)
			restarted = true
		}
		if r.ipPolicy != nil {
			if err := r.ipPolicy.Reapply(ctx, restarted); err != nil {
				r.runtime.SetApplyFailed("apply_failed", rawHash, err)
				return err
			}
		}
		r.userVersion = users.Version
		r.supervisor.SetConfigVersion(runtimeHash)
		r.runtime.SetApplyOK(rawHash)
		log.Printf("applied xray users version=%s by hot reload", users.Version)
		return nil
	}

	if err := r.supervisor.Apply(ctx, runtimeConfig); err != nil {
		r.runtime.SetApplyFailed("apply_failed", rawHash, err)
		return err
	}
	if r.ipPolicy != nil {
		if err := r.ipPolicy.Reapply(ctx, true); err != nil {
			r.runtime.SetApplyFailed("apply_failed", rawHash, err)
			return err
		}
	}
	r.lastHash = structureHash
	r.userVersion = users.Version
	r.hotUsers.SetSnapshot(desiredUsers)
	r.supervisor.SetConfigVersion(runtimeHash)
	r.runtime.SetApplyOK(rawHash)
	log.Printf("applied xray config hash=%s users_version=%s", structureHash, users.Version)
	return nil
}

func (r *Runner) buildRuntimeConfig(raw []byte) ([]byte, error) {
	return xray.BuildRuntimeConfigWithLog(raw, r.cfg.Xray.APIPort, r.cfg.Xray.InjectAPI, xray.RuntimeLogConfig{
		AccessPath: r.cfg.Log.XrayAccessPath,
		ErrorPath:  r.cfg.Log.XrayErrorPath,
		Level:      r.cfg.Log.XrayLevel,
	})
}

func (r *Runner) sampleXrayTraffic(ctx context.Context, collector *xray.GRPCStatsCollector) {
	ticker := time.NewTicker(r.cfg.Report.StatusInterval)
	defer ticker.Stop()
	var previous xray.TrafficTotals
	var previousAt time.Time
	statsFailures := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			if !r.supervisor.Running() {
				statsFailures = 0
				r.runtime.SetXrayTraffic(0, 0, previous.Download, previous.Upload, "")
				continue
			}
			totals, err := collector.CollectTotals(ctx)
			if err != nil {
				statsFailures++
				errText := ""
				if statsFailures >= 3 {
					errText = err.Error()
				}
				r.runtime.SetXrayTraffic(0, 0, previous.Download, previous.Upload, errText)
				continue
			}
			statsFailures = 0
			var rxBps, txBps int64
			if !previousAt.IsZero() {
				elapsed := now.Sub(previousAt).Seconds()
				if elapsed > 0 && totals.Download >= previous.Download && totals.Upload >= previous.Upload {
					rxBps = int64(float64(totals.Download-previous.Download) * 8 / elapsed)
					txBps = int64(float64(totals.Upload-previous.Upload) * 8 / elapsed)
				}
			}
			previous = totals
			previousAt = now
			r.runtime.SetXrayTraffic(rxBps, txBps, totals.Download, totals.Upload, "")
		}
	}
}

func (r *Runner) effectiveUsers(resp panel.XrayUsersResponse) ([]panel.XrayUser, error) {
	if !resp.Full && resp.Version == r.userVersion {
		if r.userVersion == "" {
			return nil, nil
		}
		if r.userCache == nil {
			return nil, fmt.Errorf("xray users version %s unchanged but local cache is empty", resp.Version)
		}
		return copyUsers(r.userCache), nil
	}
	r.userCache = copyUsers(resp.Users)
	return copyUsers(resp.Users), nil
}

func copyUsers(users []panel.XrayUser) []panel.XrayUser {
	if users == nil {
		return nil
	}
	copied := make([]panel.XrayUser, len(users))
	copy(copied, users)
	for i := range copied {
		copied[i].Inbounds = append([]string(nil), users[i].Inbounds...)
	}
	return copied
}

func (r *Runner) reportStatus(ctx context.Context, collector *status.Collector) {
	for {
		if err := r.reportStatusWS(ctx, collector); err != nil && ctx.Err() == nil {
			log.Printf("status websocket disconnected: %v", err)
			time.Sleep(3 * time.Second)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func (r *Runner) reportStatusWS(ctx context.Context, collector *status.Collector) error {
	conn, _, err := r.panel.DialStatusWS(ctx)
	if err != nil {
		payload := collector.Collect()
		if postErr := r.panel.PostStatus(ctx, payload); postErr != nil {
			return err
		}
		return err
	}
	defer conn.Close()

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(r.cfg.Report.StatusInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
			return ctx.Err()
		case <-readDone:
			return nil
		case <-ticker.C:
			if err := conn.WriteJSON(collector.Collect()); err != nil {
				return err
			}
		}
	}
}

func hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
