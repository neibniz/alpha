package trafficreport

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/neibniz/alpha/xray-agent/internal/panel"
)

var ErrSpoolFull = errors.New("traffic report spool is full")

type spoolState struct {
	AckOffset    int64            `json:"ack_offset"`
	WriteOffset  int64            `json:"write_offset"`
	Pending      int64            `json:"pending"`
	FirstPending int64            `json:"first_pending_at"`
	Sequence     int64            `json:"sequence"`
	Cursor       map[string]int64 `json:"stats_cursor,omitempty"`
}

type Spool struct {
	mu        sync.Mutex
	path      string
	statePath string
	maxBytes  int64
	state     spoolState
}

func Open(path, statePath string, maxBytes int64) (*Spool, error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("traffic report spool size must be positive")
	}
	for _, target := range []string{path, statePath} {
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
	}
	queue := &Spool{path: path, statePath: statePath, maxBytes: maxBytes}
	if data, err := os.ReadFile(statePath); err == nil {
		if err := json.Unmarshal(data, &queue.state); err != nil {
			return nil, fmt.Errorf("decode traffic spool state: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	if err := queue.reconcileLocked(); err != nil {
		return nil, err
	}
	return queue, nil
}

func (s *Spool) Append(report panel.TrafficReport, cursor map[string]int64) (panel.TrafficReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.compactLocked(); err != nil {
		return panel.TrafficReport{}, err
	}
	if report.ReportID == "" {
		id, err := newReportID()
		if err != nil {
			return panel.TrafficReport{}, err
		}
		report.ReportID = id
	}
	report.Sequence = s.state.Sequence + 1
	payload, err := json.Marshal(report)
	if err != nil {
		return panel.TrafficReport{}, err
	}
	payload = append(payload, '\n')
	pendingBytes := s.state.WriteOffset - s.state.AckOffset
	if pendingBytes+int64(len(payload)) > s.maxBytes {
		return panel.TrafficReport{}, ErrSpoolFull
	}

	file, err := os.OpenFile(s.path, os.O_WRONLY, 0o600)
	if err != nil {
		return panel.TrafficReport{}, err
	}
	if _, err := file.Seek(s.state.WriteOffset, io.SeekStart); err != nil {
		_ = file.Close()
		return panel.TrafficReport{}, err
	}
	if _, err := file.Write(payload); err != nil {
		_ = file.Close()
		_ = os.Truncate(s.path, s.state.WriteOffset)
		return panel.TrafficReport{}, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Truncate(s.path, s.state.WriteOffset)
		return panel.TrafficReport{}, err
	}
	if err := file.Close(); err != nil {
		_ = os.Truncate(s.path, s.state.WriteOffset)
		return panel.TrafficReport{}, err
	}

	next := s.state
	next.WriteOffset += int64(len(payload))
	next.Pending++
	next.Sequence = report.Sequence
	next.Cursor = cloneCursor(cursor)
	if next.FirstPending == 0 {
		next.FirstPending = time.Now().UnixMilli()
	}
	if err := s.saveStateLocked(next); err != nil {
		_ = os.Truncate(s.path, s.state.WriteOffset)
		return panel.TrafficReport{}, err
	}
	s.state = next
	return report, nil
}

func (s *Spool) ReadBatch(maxRows int) ([]panel.TrafficReport, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if maxRows <= 0 || s.state.Pending == 0 {
		return nil, s.state.AckOffset, nil
	}
	file, err := os.Open(s.path)
	if err != nil {
		return nil, s.state.AckOffset, err
	}
	defer file.Close()
	if _, err := file.Seek(s.state.AckOffset, io.SeekStart); err != nil {
		return nil, s.state.AckOffset, err
	}
	reader := bufio.NewReader(io.LimitReader(file, s.state.WriteOffset-s.state.AckOffset))
	reports := make([]panel.TrafficReport, 0, maxRows)
	nextOffset := s.state.AckOffset
	for len(reports) < maxRows {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			nextOffset += int64(len(line))
			var report panel.TrafficReport
			if err := json.Unmarshal(line, &report); err != nil {
				return nil, s.state.AckOffset, fmt.Errorf("decode traffic spool record: %w", err)
			}
			reports = append(reports, report)
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return nil, s.state.AckOffset, readErr
		}
	}
	return reports, nextOffset, nil
}

func (s *Spool) Acknowledge(nextOffset int64, sent int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if nextOffset < s.state.AckOffset || nextOffset > s.state.WriteOffset {
		return fmt.Errorf("invalid traffic spool acknowledgement offset %d", nextOffset)
	}
	next := s.state
	next.AckOffset = nextOffset
	next.Pending -= int64(sent)
	if next.Pending < 0 {
		next.Pending = 0
	}
	if next.AckOffset == next.WriteOffset {
		next.AckOffset = 0
		next.WriteOffset = 0
		next.Pending = 0
		next.FirstPending = 0
	}
	if err := s.saveStateLocked(next); err != nil {
		return err
	}
	s.state = next
	if next.WriteOffset == 0 {
		return os.Truncate(s.path, 0)
	}
	return nil
}

func (s *Spool) ShouldFlush(now time.Time, batchSize int, flushInterval time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Pending <= 0 {
		return false
	}
	if batchSize > 0 && s.state.Pending >= int64(batchSize) {
		return true
	}
	return s.state.FirstPending > 0 && now.Sub(time.UnixMilli(s.state.FirstPending)) >= flushInterval
}

func (s *Spool) Checkpoint() map[string]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneCursor(s.state.Cursor)
}

func (s *Spool) Pending() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.Pending
}

func (s *Spool) reconcileLocked() error {
	info, err := os.Stat(s.path)
	if err != nil {
		return err
	}
	size := info.Size()
	switch {
	case size > s.state.WriteOffset:
		if err := os.Truncate(s.path, s.state.WriteOffset); err != nil {
			return err
		}
		size = s.state.WriteOffset
	case size < s.state.WriteOffset:
		// A compaction rename completed before its state file was persisted.
		s.state.AckOffset = 0
		s.state.WriteOffset = size
	}
	if s.state.AckOffset < 0 || s.state.AckOffset > s.state.WriteOffset {
		s.state.AckOffset = 0
	}
	file, err := os.Open(s.path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Seek(s.state.AckOffset, io.SeekStart); err != nil {
		return err
	}
	scanner := bufio.NewScanner(io.LimitReader(file, s.state.WriteOffset-s.state.AckOffset))
	scanner.Buffer(make([]byte, 4096), 4<<20)
	pending := int64(0)
	for scanner.Scan() {
		var report panel.TrafficReport
		if err := json.Unmarshal(scanner.Bytes(), &report); err != nil {
			return fmt.Errorf("decode traffic spool record: %w", err)
		}
		pending++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	s.state.Pending = pending
	if pending == 0 {
		s.state.FirstPending = 0
	} else if s.state.FirstPending == 0 {
		s.state.FirstPending = time.Now().UnixMilli()
	}
	return s.saveStateLocked(s.state)
}

func (s *Spool) compactLocked() error {
	if s.state.AckOffset < 8<<20 || s.state.AckOffset < s.state.WriteOffset/2 {
		return nil
	}
	source, err := os.Open(s.path)
	if err != nil {
		return err
	}
	if _, err := source.Seek(s.state.AckOffset, io.SeekStart); err != nil {
		_ = source.Close()
		return err
	}
	tempPath := s.path + ".compact"
	target, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		_ = source.Close()
		return err
	}
	written, copyErr := io.CopyN(target, source, s.state.WriteOffset-s.state.AckOffset)
	if errors.Is(copyErr, io.EOF) {
		copyErr = nil
	}
	syncErr := target.Sync()
	closeTargetErr := target.Close()
	closeSourceErr := source.Close()
	if copyErr != nil {
		return copyErr
	}
	if syncErr != nil {
		return syncErr
	}
	if closeTargetErr != nil {
		return closeTargetErr
	}
	if closeSourceErr != nil {
		return closeSourceErr
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		return err
	}
	next := s.state
	next.AckOffset = 0
	next.WriteOffset = written
	if err := s.saveStateLocked(next); err != nil {
		return err
	}
	s.state = next
	return nil
}

func (s *Spool) saveStateLocked(state spoolState) error {
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	tempPath := s.statePath + ".tmp"
	if err := os.WriteFile(tempPath, payload, 0o600); err != nil {
		return err
	}
	return os.Rename(tempPath, s.statePath)
}

func cloneCursor(cursor map[string]int64) map[string]int64 {
	if len(cursor) == 0 {
		return nil
	}
	result := make(map[string]int64, len(cursor))
	for name, value := range cursor {
		result[name] = value
	}
	return result
}

func newReportID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	encoded := make([]byte, 36)
	hex.Encode(encoded[0:8], value[0:4])
	encoded[8] = '-'
	hex.Encode(encoded[9:13], value[4:6])
	encoded[13] = '-'
	hex.Encode(encoded[14:18], value[6:8])
	encoded[18] = '-'
	hex.Encode(encoded[19:23], value[8:10])
	encoded[23] = '-'
	hex.Encode(encoded[24:36], value[10:16])
	return string(encoded), nil
}
