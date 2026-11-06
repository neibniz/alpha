package main

import (
	"archive/zip"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	ip2location "github.com/ip2location/ip2location-go/v9"
	"github.com/neibniz/alpha/forwared/logger"
)

const maxGeoIPDownloadBytes int64 = 256 << 20
const maxGeoIPDatabaseBytes int64 = 512 << 20

// The SDK rewrites package-level IPv6 constants on OpenDB, so opening and
// closing readers must be exclusive with all lookups, not just pointer swaps.
var geoIPSDKMu sync.RWMutex

type GeoIPDatabase struct {
	file     string
	source   string
	interval time.Duration
	client   *http.Client
	updateMu sync.Mutex
	db       *ip2location.DB
	date     time.Time
	closed   bool
}

func NewGeoIPDatabase(cfg *Config) *GeoIPDatabase {
	source := strings.TrimSpace(cfg.GeoIPDownloadURL)
	if source == "" && strings.TrimSpace(cfg.GeoIPDownloadToken) != "" {
		query := url.Values{"token": {strings.TrimSpace(cfg.GeoIPDownloadToken)}, "file": {cfg.GeoIPDownloadCode}}
		source = "https://www.ip2location.com/download?" + query.Encode()
	}
	return &GeoIPDatabase{
		file: cfg.GeoIPFile, source: source, interval: time.Duration(cfg.GeoIPUpdateInterval) * time.Second,
		client: &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			req.Header.Del("Referer")
			if len(via) >= 5 || (len(via) > 0 && via[0].URL.Scheme == "https" && req.URL.Scheme != "https") {
				return errors.New("unsafe GeoIP redirect")
			}
			return nil
		}},
	}
}

func (g *GeoIPDatabase) Prepare(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(g.file), 0o700); err != nil {
		return err
	}
	if err := g.activate(g.file, false); err == nil {
		if g.source == "" {
			logger.Logger.Warn("GeoIP automatic updates require an IP2Location token or HTTPS download URL")
		}
		return nil
	}
	if g.source == "" {
		return errors.New("no usable GeoIP cache; set ALPHA_FORWARED_IP2LOCATION_TOKEN or ALPHA_FORWARED_GEOIP_URL")
	}
	return g.Update(ctx)
}

func (g *GeoIPDatabase) Lookup(ip string) (record ip2location.IP2Locationrecord, err error) {
	geoIPSDKMu.RLock()
	defer geoIPSDKMu.RUnlock()
	defer func() {
		if recover() != nil {
			record = ip2location.IP2Locationrecord{}
			err = errors.New("invalid IP2Location lookup data")
		}
	}()
	if g.db == nil {
		return ip2location.IP2Locationrecord{}, errors.New("GeoIP database is closed")
	}
	return g.db.Get_all(ip)
}

func (g *GeoIPDatabase) Close() {
	geoIPSDKMu.Lock()
	defer geoIPSDKMu.Unlock()
	g.closed = true
	if g.db != nil {
		g.db.Close()
		g.db = nil
	}
}

func (g *GeoIPDatabase) Run(ctx context.Context) {
	if g.source == "" {
		return
	}
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		info, err := os.Stat(g.file)
		if err != nil || time.Since(info.ModTime()) >= g.interval {
			if err := g.Update(ctx); err != nil && ctx.Err() == nil {
				logger.Logger.Warn("GeoIP update failed; retaining the previous database", "error", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (g *GeoIPDatabase) Update(ctx context.Context) error {
	g.updateMu.Lock()
	defer g.updateMu.Unlock()
	if g.source == "" {
		return errors.New("GeoIP download source is not configured")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, g.source, nil)
	if err != nil {
		return errors.New("invalid GeoIP download URL")
	}
	if info, err := os.Stat(g.file); err == nil {
		geoIPSDKMu.RLock()
		available := g.db != nil
		geoIPSDKMu.RUnlock()
		if available {
			request.Header.Set("If-Modified-Since", info.ModTime().UTC().Format(http.TimeFormat))
		}
	}
	response, err := g.client.Do(request)
	if err != nil {
		// HTTP errors and redirect locations can contain the download token.
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("GeoIP download request failed")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotModified {
		geoIPSDKMu.RLock()
		available := g.db != nil
		geoIPSDKMu.RUnlock()
		if !available {
			return errors.New("GeoIP source returned 304 without a usable cache")
		}
		now := time.Now()
		return os.Chtimes(g.file, now, now)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GeoIP download returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength > maxGeoIPDownloadBytes {
		return errors.New("GeoIP download is too large")
	}
	archive, err := os.CreateTemp(filepath.Dir(g.file), ".geoip-download-*")
	if err != nil {
		return err
	}
	defer os.Remove(archive.Name())
	defer archive.Close()
	if _, err := copyGeoIPBytes(archive, response.Body, maxGeoIPDownloadBytes); err != nil {
		return err
	}
	if err := archive.Sync(); err != nil {
		return err
	}
	if err := archive.Close(); err != nil {
		return err
	}
	candidate, cleanup, err := geoIPCandidate(archive.Name(), filepath.Dir(g.file))
	if err != nil {
		return err
	}
	defer cleanup()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := g.activate(candidate, true); err != nil {
		return err
	}
	logger.Logger.Info("IP2Location database updated", "attribution", "IP geolocation by https://www.ip2location.com")
	return nil
}

func copyGeoIPBytes(dst io.Writer, src io.Reader, limit int64) (int64, error) {
	n, err := io.Copy(dst, io.LimitReader(src, limit+1))
	if err != nil {
		return n, errors.New("failed to read GeoIP database content")
	}
	if n > limit {
		return n, errors.New("GeoIP database content exceeds size limit")
	}
	return n, nil
}

func geoIPCandidate(source, dir string) (string, func(), error) {
	f, err := os.Open(source)
	if err != nil {
		return "", nil, err
	}
	var magic [4]byte
	_, readErr := io.ReadFull(f, magic[:])
	f.Close()
	if readErr != nil {
		return "", nil, errors.New("GeoIP download is truncated")
	}
	if string(magic[:]) != "PK\x03\x04" {
		return source, func() {}, nil
	}
	archive, err := zip.OpenReader(source)
	if err != nil {
		return "", nil, errors.New("invalid GeoIP ZIP archive")
	}
	defer archive.Close()
	var entry *zip.File
	for _, item := range archive.File {
		if item.Mode().IsRegular() && strings.EqualFold(filepath.Ext(item.Name), ".bin") {
			if entry != nil {
				return "", nil, errors.New("GeoIP ZIP must contain exactly one BIN database")
			}
			entry = item
		}
	}
	if entry == nil || entry.UncompressedSize64 > uint64(maxGeoIPDatabaseBytes) {
		return "", nil, errors.New("GeoIP ZIP has no valid sized BIN database")
	}
	reader, err := entry.Open()
	if err != nil {
		return "", nil, errors.New("failed to open GeoIP ZIP entry")
	}
	defer reader.Close()
	tmp, err := os.CreateTemp(dir, ".geoip-candidate-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.Remove(tmp.Name()) }
	_, err = copyGeoIPBytes(tmp, reader, maxGeoIPDatabaseBytes)
	if err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err != nil || closeErr != nil {
		cleanup()
		return "", nil, errors.New("failed to extract GeoIP database")
	}
	return tmp.Name(), cleanup, nil
}

func (g *GeoIPDatabase) activate(candidate string, replace bool) error {
	geoIPSDKMu.Lock()
	defer geoIPSDKMu.Unlock()
	if g.closed {
		return errors.New("GeoIP database is closed")
	}
	f, err := os.Open(candidate)
	if err != nil {
		return err
	}
	var header [64]byte
	_, err = io.ReadFull(f, header[:])
	info, statErr := f.Stat()
	if err != nil || statErr != nil || info.Size() > maxGeoIPDatabaseBytes || header[0] < 3 || header[0] > 26 || header[1] < 4 || header[1] > 32 || (header[2] >= 21 && header[29] != 1) {
		f.Close()
		return errors.New("invalid IP2Location BIN header (country and region are required)")
	}
	date := time.Date(2000+int(header[2]), time.Month(header[3]), int(header[4]), 0, 0, 0, 0, time.UTC)
	if header[3] < 1 || header[3] > 12 || header[4] < 1 || date.Day() != int(header[4]) || date.After(time.Now().Add(24*time.Hour)) || (!g.date.IsZero() && date.Before(g.date)) {
		f.Close()
		return errors.New("invalid or older IP2Location database date")
	}
	for _, table := range []struct{ count, address, width uint32 }{
		{binary.LittleEndian.Uint32(header[5:9]), binary.LittleEndian.Uint32(header[9:13]), uint32(header[1]) * 4},
		{binary.LittleEndian.Uint32(header[13:17]), binary.LittleEndian.Uint32(header[17:21]), 16 + (uint32(header[1])-1)*4},
	} {
		if table.count > 0 && (table.address < 65 || uint64(table.address)-1+(uint64(table.count)+1)*uint64(table.width) > uint64(info.Size())) {
			f.Close()
			return errors.New("truncated IP2Location database table")
		}
	}
	if binary.LittleEndian.Uint32(header[5:9]) == 0 {
		f.Close()
		return errors.New("IP2Location database has no IPv4 ranges")
	}
	db, err := validateGeoIPReader(f)
	if err != nil {
		return err
	}
	if replace {
		if err := os.Rename(candidate, g.file); err != nil {
			db.Close()
			return err
		}
	}
	old := g.db
	g.db = db
	g.date = date
	if old != nil {
		old.Close()
	}
	return nil
}

func validateGeoIPReader(f *os.File) (db *ip2location.DB, err error) {
	// The upstream SDK may panic on malformed pointer/column metadata. Reject a
	// candidate without crashing the running forwarding service or losing its cache.
	defer func() {
		if recover() != nil {
			if db != nil {
				db.Close()
			} else {
				f.Close()
			}
			db = nil
			err = errors.New("invalid IP2Location database metadata")
		}
	}()
	db, err = ip2location.OpenDBWithReader(f)
	if err != nil {
		return nil, errors.New("failed to open IP2Location database")
	}
	record, lookupErr := db.Get_all("8.8.8.8")
	if lookupErr != nil || record.Country_short == "" || record.Region == "" {
		db.Close()
		return nil, errors.New("IP2Location database lookup validation failed")
	}
	return db, nil
}
