package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/neibniz/alpha/forwared/logger"
)

func NewForwarderConfig(cfg *Config, agent *Agent) *ForwarderConfig {
	ctx, cancel := context.WithCancel(context.Background())
	config := &ForwarderConfig{
		cfg:                        cfg,
		forwaredTable:              &ForwaredTable{ForwaredItems: []ForwaredItem{}},
		fetchIPInterval:            cfg.FetchIPInterval,
		fetchForwaredTableInterval: cfg.FetchForwardTableInterval,
		ctx:                        ctx,
		cancel:                     cancel,
		agent:                      agent,
		applyStatus:                "pending",
	}
	if config.fetchForwaredTableInterval == 0 {
		config.fetchForwaredTableInterval = 10
	}
	config.fm = NewForwardManager(agent.Metrics(), agent.Enqueue, agent.BootID, agent.ConfigVersion)
	return config
}

func (f *ForwarderConfig) fetchForwaredTable() {
	ctx, cancel := context.WithTimeout(f.ctx, 15*time.Second)
	defer cancel()
	table, err := f.agent.FetchConfig(ctx)
	if err != nil {
		logger.Logger.Warn("Fetch forward table failed", slog.Any("error", err))
		return
	}
	f.applyForwaredTable(table)
}

func (f *ForwarderConfig) applyForwaredTable(desired *ForwaredTable) bool {
	f.mu.Lock()
	oldTable := f.forwaredTable
	changed := f.handleForwaredTable(oldTable, desired)
	f.mu.Unlock()

	oldPorts := groupForwardItems(oldTable)
	currentPorts := groupForwardItems(oldTable)
	allSuccess := true
	var firstError error
	for port, targets := range changed {
		_ = f.fm.RemoveForward(port)
		if len(targets) == 0 {
			logger.Logger.Info("Remove forward port", slog.Int("listen_port", int(port)))
			delete(currentPorts, port)
			continue
		}
		if err := f.fm.AddForward(port, targets); err != nil {
			allSuccess = false
			if firstError == nil {
				firstError = err
			}
			logger.Logger.Error("Add forward port failed",
				slog.Int("listen_port", int(port)),
				slog.Int("targets", len(targets)),
				slog.Any("error", err))
			if oldTargets := oldPorts[port]; len(oldTargets) > 0 {
				if restoreErr := f.fm.AddForward(port, oldTargets); restoreErr != nil {
					logger.Logger.Error("Restore previous forward port failed",
						slog.Int("listen_port", int(port)),
						slog.Any("error", restoreErr))
					delete(currentPorts, port)
				} else {
					currentPorts[port] = oldTargets
				}
			}
			continue
		}
		currentPorts[port] = targets
		logger.Logger.Info("Apply forward port",
			slog.Int("listen_port", int(port)),
			slog.Int("targets", len(targets)))
	}

	actual := tableFromForwardItems(desired, currentPorts)
	applyStatus := "ok"
	if !allSuccess {
		applyStatus = "degraded"
	}
	f.mu.Lock()
	f.forwaredTable = actual
	f.desiredVersion = desired.Version
	f.desiredHash = desired.Hash
	f.appliedVersion = actual.Version
	f.appliedHash = actual.Hash
	f.applyStatus = applyStatus
	f.applyError = ""
	if !allSuccess {
		if firstError != nil {
			f.applyError = firstError.Error()
		}
	}
	f.mu.Unlock()
	f.agent.SetConfigState(desired, actual, applyStatus, firstError)
	return allSuccess
}

func (f *ForwarderConfig) handleForwaredTable(oldTable, newTable *ForwaredTable) map[uint][]ForwaredItem {
	oldPorts := groupForwardItems(oldTable)
	newPorts := groupForwardItems(newTable)
	changed := map[uint][]ForwaredItem{}
	for port, oldTargets := range oldPorts {
		if newTargets, ok := newPorts[port]; !ok || !sameForwardTargets(oldTargets, newTargets) {
			changed[port] = newPorts[port]
		}
	}
	for port, newTargets := range newPorts {
		if _, ok := oldPorts[port]; !ok {
			changed[port] = newTargets
		}
	}
	return changed
}

func groupForwardItems(table *ForwaredTable) map[uint][]ForwaredItem {
	out := map[uint][]ForwaredItem{}
	if table == nil {
		return out
	}
	seen := map[ForwaredItem]struct{}{}
	for _, item := range table.ForwaredItems {
		if item.LocalPort == 0 || item.LocalPort > 65535 || item.TargetHost == "" || item.TargetPort == 0 || item.TargetPort > 65535 {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out[item.LocalPort] = append(out[item.LocalPort], item)
	}
	for port := range out {
		sort.Slice(out[port], func(i, j int) bool {
			if out[port][i].TargetHost == out[port][j].TargetHost {
				return out[port][i].TargetPort < out[port][j].TargetPort
			}
			return out[port][i].TargetHost < out[port][j].TargetHost
		})
	}
	return out
}

func tableFromForwardItems(desired *ForwaredTable, items map[uint][]ForwaredItem) *ForwaredTable {
	out := &ForwaredTable{
		InstanceID:    desired.InstanceID,
		IP:            desired.IP,
		Version:       desired.Version,
		Hash:          desired.Hash,
		GeneratedAt:   desired.GeneratedAt,
		ForwaredItems: make([]ForwaredItem, 0),
		Warnings:      append([]string(nil), desired.Warnings...),
	}
	ports := make([]int, 0, len(items))
	for port := range items {
		ports = append(ports, int(port))
	}
	sort.Ints(ports)
	for _, port := range ports {
		out.ForwaredItems = append(out.ForwaredItems, items[uint(port)]...)
	}
	canonical, err := json.Marshal(out.ForwaredItems)
	if err == nil {
		sum := sha256.Sum256(canonical)
		out.Hash = hex.EncodeToString(sum[:])
		if out.Hash == desired.Hash {
			out.Version = desired.Version
		} else {
			out.Version = "actual-" + out.Hash[:16]
		}
	}
	return out
}

func sameForwardTargets(a, b []ForwaredItem) bool {
	if len(a) != len(b) {
		return false
	}
	counts := map[ForwaredItem]int{}
	for _, item := range a {
		counts[item]++
	}
	for _, item := range b {
		counts[item]--
		if counts[item] < 0 {
			return false
		}
	}
	return true
}

func (f *ForwarderConfig) fetch() {
	f.fetchForwaredTable()
	go func() {
		ticker := time.NewTicker(time.Duration(f.fetchForwaredTableInterval) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				f.fetchForwaredTable()
			case <-f.ctx.Done():
				return
			}
		}
	}()
}

func (f *ForwarderConfig) stop() {
	if f.cancel != nil {
		f.cancel()
	}
	f.fm.StopAll()
}

func (f *ForwarderConfig) validate() error {
	if f.agent == nil {
		return fmt.Errorf("agent is required")
	}
	return nil
}
