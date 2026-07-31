package accesslog

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/neibniz/alpha/xray-agent/internal/panel"
)

var errSpoolFull = errors.New("access event spool is full")

const maxSpoolRecordBytes = 1 << 20

type collectedRecord struct {
	Event          panel.AccessEvent
	EndOffset      int64
	ParseFailure   bool
	DiscardingLine bool
}

type spoolState struct {
	FileDevice     uint64 `json:"file_device"`
	FileInode      uint64 `json:"file_inode"`
	FileOffset     int64  `json:"file_offset"`
	AckOffset      int64  `json:"ack_offset"`
	Pending        int64  `json:"pending"`
	FirstPending   int64  `json:"first_pending_at"`
	Dropped        int64  `json:"dropped"`
	ParseFailures  int64  `json:"parse_failures"`
	DiscardingLine bool   `json:"discarding_line,omitempty"`
}

type spool struct {
	path      string
	statePath string
	maxBytes  int64
	state     spoolState
}

func openSpool(path, statePath string, maxBytes int64) (*spool, error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("access event spool size must be positive")
	}
	for _, target := range []string{path, statePath} {
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
	}
	s := &spool{path: path, statePath: statePath, maxBytes: maxBytes}
	if data, err := os.ReadFile(statePath); err == nil {
		if err := json.Unmarshal(data, &s.state); err != nil {
			return nil, fmt.Errorf("decode access spool state: %w", err)
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
	if err := s.reconcile(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *spool) appendRecords(records []collectedRecord, device, inode uint64, startOffset int64) (int, error) {
	if startOffset < 0 {
		return 0, fmt.Errorf("invalid access source offset %d", startOffset)
	}
	payloads := make([][]byte, len(records))
	var requiredBytes int64
	previousOffset := startOffset
	for index, record := range records {
		if record.EndOffset <= previousOffset {
			return 0, fmt.Errorf("invalid access source record offset %d", record.EndOffset)
		}
		previousOffset = record.EndOffset
		if record.ParseFailure || record.Event.EventID == "" {
			continue
		}
		payload, err := json.Marshal(record.Event)
		if err != nil {
			return 0, err
		}
		payloads[index] = append(payload, '\n')
		requiredBytes += int64(len(payloads[index]))
	}
	if err := s.compactIfNeeded(requiredBytes); err != nil {
		return 0, err
	}
	file, err := os.OpenFile(s.path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return 0, err
	}
	startSize := info.Size()
	size := startSize
	committed := 0
	accepted := int64(0)
	next := s.state
	sameSource := next.FileDevice == device && next.FileInode == inode && next.FileOffset == startOffset
	next.FileDevice = device
	next.FileInode = inode
	next.FileOffset = startOffset
	if !sameSource {
		next.DiscardingLine = false
	}
	for index, record := range records {
		payload := payloads[index]
		if len(payload) > 0 && size+int64(len(payload)) > s.maxBytes {
			break
		}
		if len(payload) > 0 {
			if _, err := file.Write(payload); err != nil {
				_ = file.Close()
				_ = os.Truncate(s.path, startSize)
				return 0, err
			}
			size += int64(len(payload))
			accepted++
		} else if record.ParseFailure {
			next.ParseFailures++
		}
		next.FileOffset = record.EndOffset
		next.DiscardingLine = record.DiscardingLine
		committed++
	}
	if accepted > 0 {
		if err := file.Sync(); err != nil {
			_ = file.Close()
			_ = os.Truncate(s.path, startSize)
			return 0, err
		}
	}
	if err := file.Close(); err != nil {
		_ = os.Truncate(s.path, startSize)
		return 0, err
	}
	if accepted > 0 && next.Pending == 0 {
		next.FirstPending = time.Now().UnixMilli()
	}
	next.Pending += accepted
	if err := s.saveState(next); err != nil {
		_ = os.Truncate(s.path, startSize)
		return 0, err
	}
	s.state = next
	if committed < len(records) {
		return committed, errSpoolFull
	}
	return committed, nil
}

func (s *spool) readBatch(maxRows int, maxBytes int64) ([]panel.AccessEvent, int64, error) {
	if maxRows <= 0 {
		return nil, s.state.AckOffset, nil
	}
	if maxBytes <= 0 {
		return nil, s.state.AckOffset, fmt.Errorf("access event batch size must be positive")
	}
	file, err := os.Open(s.path)
	if err != nil {
		return nil, s.state.AckOffset, err
	}
	defer file.Close()
	if _, err := file.Seek(s.state.AckOffset, io.SeekStart); err != nil {
		return nil, s.state.AckOffset, err
	}
	reader := bufio.NewReader(file)
	events := make([]panel.AccessEvent, 0, maxRows)
	nextOffset := s.state.AckOffset
	var bytesRead int64
	for len(events) < maxRows && bytesRead < maxBytes {
		line, consumed, complete, readErr := readBoundedLine(reader, maxSpoolRecordBytes)
		if errors.Is(readErr, io.EOF) && consumed == 0 {
			break
		}
		if errors.Is(readErr, errLineTooLong) {
			return nil, s.state.AckOffset, fmt.Errorf("decode access spool record at byte %d: %w", nextOffset, readErr)
		}
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return nil, s.state.AckOffset, readErr
		}
		if !complete {
			return nil, s.state.AckOffset, fmt.Errorf("decode access spool record at byte %d: truncated record", nextOffset)
		}
		var event panel.AccessEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return nil, s.state.AckOffset, fmt.Errorf("decode access spool record at byte %d: %w", nextOffset, err)
		}
		nextOffset += consumed
		bytesRead += consumed
		events = append(events, event)
	}
	return events, nextOffset, nil
}

func (s *spool) acknowledge(nextOffset int64, sent int) error {
	info, err := os.Stat(s.path)
	if err != nil {
		return err
	}
	if nextOffset < s.state.AckOffset || nextOffset > info.Size() {
		return fmt.Errorf("invalid access spool acknowledgement offset %d", nextOffset)
	}
	if sent < 0 || int64(sent) > s.state.Pending || (sent == 0 && nextOffset != s.state.AckOffset) {
		return fmt.Errorf("invalid access spool acknowledgement count %d", sent)
	}
	next := s.state
	next.AckOffset = nextOffset
	next.Pending -= int64(sent)
	if next.AckOffset == info.Size() {
		if err := os.Truncate(s.path, 0); err != nil {
			return err
		}
		next.AckOffset = 0
		next.Pending = 0
		next.FirstPending = 0
	}
	if err := s.saveState(next); err != nil {
		return err
	}
	s.state = next
	return nil
}

func (s *spool) reconcile() error {
	info, err := os.Stat(s.path)
	if err != nil {
		return err
	}
	if s.state.FileOffset < 0 || s.state.Pending < 0 || s.state.FirstPending < 0 || s.state.ParseFailures < 0 || s.state.Dropped < 0 {
		return fmt.Errorf("invalid access spool state")
	}
	if s.state.AckOffset < 0 || s.state.AckOffset > info.Size() {
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
	reader := bufio.NewReader(file)
	pending := int64(0)
	offset := s.state.AckOffset
	for {
		line, consumed, complete, readErr := readBoundedLine(reader, maxSpoolRecordBytes)
		if errors.Is(readErr, io.EOF) && consumed == 0 {
			break
		}
		if errors.Is(readErr, errLineTooLong) {
			return fmt.Errorf("decode access spool record at byte %d: %w", offset, readErr)
		}
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return readErr
		}
		if !complete {
			return fmt.Errorf("decode access spool record at byte %d: truncated record", offset)
		}
		var event panel.AccessEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return fmt.Errorf("decode access spool record at byte %d: %w", offset, err)
		}
		pending++
		offset += consumed
	}
	next := s.state
	next.Pending = pending
	if pending == 0 {
		next.FirstPending = 0
	} else if next.FirstPending == 0 {
		next.FirstPending = time.Now().UnixMilli()
	}
	if err := s.saveState(next); err != nil {
		return err
	}
	s.state = next
	return nil
}

func (s *spool) compactIfNeeded(requiredBytes int64) error {
	info, err := os.Stat(s.path)
	if err != nil {
		return err
	}
	pressure := s.state.AckOffset > 0 && info.Size()+requiredBytes > s.maxBytes
	periodic := s.state.AckOffset >= 8<<20 && s.state.AckOffset >= info.Size()/2
	if !pressure && !periodic {
		return nil
	}
	source, err := os.Open(s.path)
	if err != nil {
		return err
	}
	defer source.Close()
	if _, err := source.Seek(s.state.AckOffset, io.SeekStart); err != nil {
		return err
	}
	tempPath := s.path + ".compact"
	target, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(target, source)
	syncErr := target.Sync()
	closeErr := target.Close()
	if copyErr != nil {
		return copyErr
	}
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		return err
	}
	next := s.state
	next.AckOffset = 0
	if err := s.saveState(next); err != nil {
		return err
	}
	s.state = next
	return nil
}

func (s *spool) saveState(state spoolState) error {
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	tempPath := s.statePath + ".tmp"
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(payload); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, s.statePath); err != nil {
		return err
	}
	if dir, err := os.Open(filepath.Dir(s.statePath)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}
