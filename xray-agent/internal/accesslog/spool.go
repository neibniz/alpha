package accesslog

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/neibniz/alpha/xray-agent/internal/panel"
)

var errSpoolFull = errors.New("access event spool is full")

type spoolState struct {
	FileDevice    uint64 `json:"file_device"`
	FileInode     uint64 `json:"file_inode"`
	FileOffset    int64  `json:"file_offset"`
	AckOffset     int64  `json:"ack_offset"`
	Pending       int64  `json:"pending"`
	FirstPending  int64  `json:"first_pending_at"`
	Dropped       int64  `json:"dropped"`
	ParseFailures int64  `json:"parse_failures"`
}

type spool struct {
	path      string
	statePath string
	maxBytes  int64
	state     spoolState
}

func openSpool(path, statePath string, maxBytes int64) (*spool, error) {
	s := &spool{path: path, statePath: statePath, maxBytes: maxBytes}
	if data, err := os.ReadFile(statePath); err == nil {
		_ = json.Unmarshal(data, &s.state)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	_ = file.Close()
	if err := s.reconcile(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *spool) appendAndAdvance(events []panel.AccessEvent, device, inode uint64, nextOffset int64, parseFailures int64) error {
	if len(events) == 0 {
		s.state.FileDevice = device
		s.state.FileInode = inode
		s.state.FileOffset = nextOffset
		s.state.ParseFailures += parseFailures
		return s.saveState()
	}
	if err := s.compactIfNeeded(); err != nil {
		return err
	}
	file, err := os.OpenFile(s.path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}
	startSize := info.Size()
	size := startSize
	accepted := int64(0)
	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			continue
		}
		payload = append(payload, '\n')
		if size+int64(len(payload)) > s.maxBytes {
			s.state.Dropped += int64(len(events)) - accepted
			break
		}
		if _, err := file.Write(payload); err != nil {
			_ = file.Close()
			_ = os.Truncate(s.path, startSize)
			return err
		}
		size += int64(len(payload))
		accepted++
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Truncate(s.path, startSize)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Truncate(s.path, startSize)
		return err
	}
	s.state.FileDevice = device
	s.state.FileInode = inode
	s.state.FileOffset = nextOffset
	s.state.ParseFailures += parseFailures
	if accepted > 0 && s.state.Pending == 0 {
		s.state.FirstPending = time.Now().UnixMilli()
	}
	s.state.Pending += accepted
	if err := s.saveState(); err != nil {
		return err
	}
	if accepted < int64(len(events)) {
		return errSpoolFull
	}
	return nil
}

func (s *spool) readBatch(maxRows int, maxBytes int64) ([]panel.AccessEvent, int64, error) {
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
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			nextOffset += int64(len(line))
			bytesRead += int64(len(line))
			var event panel.AccessEvent
			if json.Unmarshal(line, &event) == nil {
				events = append(events, event)
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, s.state.AckOffset, err
		}
	}
	return events, nextOffset, nil
}

func (s *spool) acknowledge(nextOffset int64, sent int) error {
	s.state.AckOffset = nextOffset
	s.state.Pending -= int64(sent)
	if s.state.Pending < 0 {
		s.state.Pending = 0
	}
	info, err := os.Stat(s.path)
	if err != nil {
		return err
	}
	if s.state.AckOffset >= info.Size() {
		if err := os.Truncate(s.path, 0); err != nil {
			return err
		}
		s.state.AckOffset = 0
		s.state.Pending = 0
		s.state.FirstPending = 0
	}
	return s.saveState()
}

func (s *spool) reconcile() error {
	info, err := os.Stat(s.path)
	if err != nil {
		return err
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
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	pending := int64(0)
	for scanner.Scan() {
		var event panel.AccessEvent
		if json.Unmarshal(scanner.Bytes(), &event) == nil {
			pending++
		}
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
	return s.saveState()
}

func (s *spool) compactIfNeeded() error {
	info, err := os.Stat(s.path)
	if err != nil {
		return err
	}
	if s.state.AckOffset < 8<<20 || s.state.AckOffset < info.Size()/2 {
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
	s.state.AckOffset = 0
	return s.saveState()
}

func (s *spool) saveState() error {
	payload, err := json.Marshal(s.state)
	if err != nil {
		return err
	}
	tempPath := s.statePath + ".tmp"
	if err := os.WriteFile(tempPath, payload, 0o600); err != nil {
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
