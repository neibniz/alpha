package accesslog

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/neibniz/alpha/xray-agent/internal/config"
	"github.com/neibniz/alpha/xray-agent/internal/panel"
)

const (
	maxLinesPerPoll       = 5000
	maxAccessLogLineBytes = 64 << 10
	minAccessSpoolBytes   = 1 << 20
)

type EventPoster interface {
	PostAccessEvents(context.Context, []panel.AccessEvent) (panel.AccessPostResult, error)
}

type PolicyDecisionSink interface {
	ApplyIPLimitDecisions(context.Context, []panel.IPLimitDecision) error
}

type Collector struct {
	serverID int64
	logPath  string
	cfg      config.AccessReportConfig
	poster   EventPoster
	policy   PolicyDecisionSink
	spool    *spool
}

func NewCollector(serverID int64, logPath string, cfg config.AccessReportConfig, poster EventPoster, policy ...PolicyDecisionSink) (*Collector, error) {
	if cfg.MaxSpoolBytes < minAccessSpoolBytes {
		return nil, fmt.Errorf("access spool size must be at least %d bytes", minAccessSpoolBytes)
	}
	queue, err := openSpool(cfg.SpoolPath, cfg.StatePath, cfg.MaxSpoolBytes)
	if err != nil {
		return nil, err
	}
	collector := &Collector{serverID: serverID, logPath: logPath, cfg: cfg, poster: poster, spool: queue}
	if len(policy) > 0 {
		collector.policy = policy[0]
	}
	return collector, nil
}

func (c *Collector) Run(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.PollInterval)
	defer ticker.Stop()
	backoff := time.Second
	var retryAt time.Time
	for {
		if err := c.collect(); err != nil && !os.IsNotExist(err) {
			log.Printf("collect xray access log failed: %v", err)
		}
		now := time.Now()
		if c.shouldFlush(now) && !now.Before(retryAt) {
			if err := c.flush(ctx); err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("upload xray access events failed: %v", err)
				retryAt = now.Add(backoff)
				backoff *= 2
				if backoff > time.Minute {
					backoff = time.Minute
				}
			} else {
				backoff = time.Second
				retryAt = time.Time{}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *Collector) shouldFlush(now time.Time) bool {
	if c.spool.state.Pending <= 0 {
		return false
	}
	if c.spool.state.Pending >= int64(c.cfg.BatchSize) {
		return true
	}
	return c.spool.state.FirstPending > 0 && now.Sub(time.UnixMilli(c.spool.state.FirstPending)) >= c.cfg.FlushInterval
}

func (c *Collector) collect() error {
	info, err := os.Stat(c.logPath)
	if err != nil {
		return err
	}
	device, inode := fileIdentity(info)
	state := c.spool.state
	if state.FileDevice != 0 && state.FileInode != 0 &&
		(state.FileDevice != device || state.FileInode != inode) {
		rotatedPath, found, err := findFileByIdentity(c.logPath, state.FileDevice, state.FileInode)
		if err != nil {
			return err
		}
		if found {
			drained, err := c.collectFile(rotatedPath, state.FileOffset, true)
			if err != nil || !drained {
				return err
			}
		}
		_, err = c.collectFile(c.logPath, 0, false)
		return err
	}
	_, err = c.collectFile(c.logPath, state.FileOffset, false)
	return err
}

func (c *Collector) collectFile(path string, offset int64, finalizeEOF bool) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return false, err
	}
	device, inode := fileIdentity(info)
	if info.Size() < offset {
		offset = 0
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return false, err
	}
	reader := bufio.NewReader(file)
	records := make([]collectedRecord, 0, 64)
	nextOffset := offset
	discardingLine := c.spool.state.FileDevice == device &&
		c.spool.state.FileInode == inode &&
		c.spool.state.FileOffset == offset &&
		c.spool.state.DiscardingLine
	drained := false
	for lineNumber := 0; lineNumber < maxLinesPerPoll; lineNumber++ {
		lineOffset := nextOffset
		line, consumed, complete, readErr := readBoundedLine(reader, maxAccessLogLineBytes)
		if errors.Is(readErr, io.EOF) && consumed == 0 {
			drained = true
			break
		}
		oversized := errors.Is(readErr, errLineTooLong)
		if readErr != nil && !errors.Is(readErr, io.EOF) && !oversized {
			return false, readErr
		}
		if discardingLine {
			nextOffset += consumed
			if complete || (errors.Is(readErr, io.EOF) && finalizeEOF) {
				discardingLine = false
			}
			records = append(records, collectedRecord{
				EndOffset:      nextOffset,
				DiscardingLine: discardingLine,
			})
			if !complete {
				drained = errors.Is(readErr, io.EOF) && finalizeEOF
				break
			}
			continue
		}
		if oversized {
			nextOffset += consumed
			discardingLine = !complete && !(errors.Is(readErr, io.EOF) && finalizeEOF)
			records = append(records, collectedRecord{
				EndOffset:      nextOffset,
				ParseFailure:   true,
				DiscardingLine: discardingLine,
			})
			if !complete {
				drained = errors.Is(readErr, io.EOF) && finalizeEOF
				break
			}
			continue
		}
		if !complete {
			if !finalizeEOF {
				drained = false
				break
			}
			drained = true
		}
		nextOffset += consumed
		record := collectedRecord{EndOffset: nextOffset}
		event, err := Parse(strings.TrimRight(string(line), "\r\n"))
		if err != nil {
			record.ParseFailure = true
		} else {
			event.EventID = eventID(c.serverID, device, inode, lineOffset, line)
			record.Event = event
		}
		records = append(records, record)
		if drained {
			break
		}
	}
	committed, err := c.spool.appendRecords(records, device, inode, offset)
	if errors.Is(err, errSpoolFull) {
		log.Printf("xray access spool reached %d bytes; source cursor paused for retry", c.cfg.MaxSpoolBytes)
		return false, nil
	}
	return drained && committed == len(records), err
}

func findFileByIdentity(activePath string, device, inode uint64) (string, bool, error) {
	entries, err := os.ReadDir(filepath.Dir(activePath))
	if err != nil {
		return "", false, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(filepath.Dir(activePath), entry.Name())
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", false, err
		}
		candidateDevice, candidateInode := fileIdentity(info)
		if candidateDevice == device && candidateInode == inode {
			return path, true, nil
		}
	}
	return "", false, nil
}

func (c *Collector) flush(ctx context.Context) error {
	events, nextOffset, err := c.spool.readBatch(c.cfg.BatchSize, c.cfg.MaxBatchBytes)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		if nextOffset != c.spool.state.AckOffset {
			return c.spool.acknowledge(nextOffset, 0)
		}
		return nil
	}
	result, err := c.poster.PostAccessEvents(ctx, events)
	if err != nil {
		return err
	}
	if result.Accepted < 0 || result.Rejected < 0 || result.Accepted+result.Rejected != len(events) {
		return fmt.Errorf("panel acknowledged %d accepted and %d rejected of %d access events", result.Accepted, result.Rejected, len(events))
	}
	if c.policy != nil && len(result.IPLimitDecisions) > 0 {
		if err := c.policy.ApplyIPLimitDecisions(ctx, result.IPLimitDecisions); err != nil {
			log.Printf("apply xray IP limit decisions failed: %v", err)
		}
	}
	return c.spool.acknowledge(nextOffset, len(events))
}

func fileIdentity(info os.FileInfo) (uint64, uint64) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, uint64(info.ModTime().UnixNano())
	}
	return uint64(stat.Dev), uint64(stat.Ino)
}

func eventID(serverID int64, device, inode uint64, offset int64, line []byte) string {
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "%d:%d:%d:%d:", serverID, device, inode, offset)
	_, _ = hash.Write(line)
	sum := hash.Sum(nil)[:16]
	sum[6] = (sum[6] & 0x0f) | 0x50
	sum[8] = (sum[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(sum)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}
