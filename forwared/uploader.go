package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/neibniz/alpha/forwared/logger"
)

type EventUploader struct {
	path          string
	maxSpoolBytes int64
	batchSize     int
	flushInterval time.Duration
	retryDelay    time.Duration
	nextAttempt   time.Time
	send          func(context.Context, []ConnectionEvent) error
	queue         chan ConnectionEvent
	stop          chan struct{}
	done          chan struct{}
	stopOnce      sync.Once
}

func NewEventUploader(cfg *Config, send func(context.Context, []ConnectionEvent) error) *EventUploader {
	return &EventUploader{
		path:          cfg.SpoolFile,
		maxSpoolBytes: cfg.MaxSpoolBytes,
		batchSize:     cfg.EventBatchSize,
		flushInterval: time.Duration(cfg.EventFlushInterval) * time.Second,
		send:          send,
		queue:         make(chan ConnectionEvent, 8192),
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}
}

func (u *EventUploader) Start() error {
	if err := os.MkdirAll(filepath.Dir(u.path), 0o700); err != nil {
		return err
	}
	go u.run()
	return nil
}

func (u *EventUploader) Enqueue(event ConnectionEvent) {
	select {
	case u.queue <- event:
	case <-u.stop:
		logger.Logger.Warn("Connection event rejected during shutdown", "event_type", event.EventType)
	}
}

func (u *EventUploader) Stop() {
	u.stopOnce.Do(func() { close(u.stop) })
	<-u.done
}

func (u *EventUploader) run() {
	defer close(u.done)
	pending, err := u.readSpool()
	if err != nil {
		logger.Logger.Error("Read connection event spool failed", "error", err)
	}
	ticker := time.NewTicker(u.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case event := <-u.queue:
			if err := u.appendSpool(event); err != nil {
				logger.Logger.Error("Append connection event spool failed", "error", err)
				continue
			}
			pending = append(pending, event)
			if len(pending) >= u.batchSize {
				pending = u.flush(pending, false)
			}
		case <-ticker.C:
			pending = u.flush(pending, false)
		case <-u.stop:
			for {
				select {
				case event := <-u.queue:
					if err := u.appendSpool(event); err == nil {
						pending = append(pending, event)
					}
				default:
					u.flush(pending, true)
					return
				}
			}
		}
	}
}

func (u *EventUploader) flush(pending []ConnectionEvent, force bool) []ConnectionEvent {
	if len(pending) == 0 {
		return pending
	}
	now := time.Now()
	if !force && now.Before(u.nextAttempt) {
		return pending
	}
	count := min(len(pending), u.batchSize)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	err := u.send(ctx, pending[:count])
	cancel()
	if err != nil {
		if u.retryDelay == 0 {
			u.retryDelay = time.Second
		} else {
			u.retryDelay = min(u.retryDelay*2, time.Minute)
		}
		jitterWindow := u.retryDelay / 5
		jitter := time.Duration(now.UnixNano() % int64(jitterWindow+1))
		u.nextAttempt = now.Add(u.retryDelay + jitter)
		logger.Logger.Warn(
			"Upload connection events failed",
			"events", count,
			"retry_after", time.Until(u.nextAttempt).Round(time.Millisecond),
			"error", err,
		)
		return pending
	}
	u.retryDelay = 0
	u.nextAttempt = time.Time{}
	pending = append([]ConnectionEvent(nil), pending[count:]...)
	if err := u.rewriteSpool(pending); err != nil {
		logger.Logger.Error("Rewrite connection event spool failed", "error", err)
	}
	return pending
}

func (u *EventUploader) appendSpool(event ConnectionEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if info, err := os.Stat(u.path); err == nil && info.Size()+int64(len(payload)+1) > u.maxSpoolBytes {
		return fmt.Errorf("spool exceeds %d bytes", u.maxSpoolBytes)
	}
	file, err := os.OpenFile(u.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(payload); err != nil {
		return err
	}
	_, err = file.Write([]byte{'\n'})
	return err
}

func (u *EventUploader) readSpool() ([]ConnectionEvent, error) {
	file, err := os.Open(u.path)
	if errors.Is(err, os.ErrNotExist) {
		return []ConnectionEvent{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	result := make([]ConnectionEvent, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		var event ConnectionEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			logger.Logger.Warn("Skip invalid connection event spool row", "error", err)
			continue
		}
		result = append(result, event)
	}
	return result, scanner.Err()
}

func (u *EventUploader) rewriteSpool(events []ConnectionEvent) error {
	temp := u.path + ".tmp"
	file, err := os.OpenFile(temp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			file.Close()
			return err
		}
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(temp, u.path)
}
