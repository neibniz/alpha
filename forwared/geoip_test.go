package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// A tiny synthetic DB3 fixture; real databases are never stored in the repository.
func geoIPTestBIN(day byte, region string) []byte {
	data := make([]byte, 512)
	data[0] = 3
	data[1] = 4
	data[2] = byte(time.Now().Year() - 2001)
	data[3] = 1
	data[4] = day
	data[29] = 1
	binary.LittleEndian.PutUint32(data[5:9], 1)
	binary.LittleEndian.PutUint32(data[9:13], 65)
	binary.LittleEndian.PutUint32(data[31:35], uint32(len(data)))
	binary.LittleEndian.PutUint32(data[68:72], 128)
	binary.LittleEndian.PutUint32(data[72:76], 160)
	binary.LittleEndian.PutUint32(data[76:80], 200)
	binary.LittleEndian.PutUint32(data[80:84], 0xffffffff)
	data[128] = 2
	copy(data[129:], "CN")
	data[131] = 5
	copy(data[132:], "China")
	data[160] = byte(len(region))
	copy(data[161:], region)
	data[200] = 4
	copy(data[201:], "Test")
	return data
}

func geoIPTestZIP(t *testing.T, name string, payload []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	f, err := writer.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func newTestGeoIP(t *testing.T, source string) *GeoIPDatabase {
	t.Helper()
	g := NewGeoIPDatabase(&Config{GeoIPFile: filepath.Join(t.TempDir(), "geoip", "db.BIN"), GeoIPDownloadURL: source, GeoIPUpdateInterval: 86400})
	t.Cleanup(g.Close)
	return g
}

func TestGeoIPPrepareDownloadsZIPAndKeepsOnlyDatabase(t *testing.T) {
	payload := geoIPTestBIN(1, "Xinjiang Uygur")
	zipped := geoIPTestZIP(t, "../../outside.BIN", payload)
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(zipped) }))
	defer source.Close()
	g := newTestGeoIP(t, source.URL)
	if err := g.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	record, err := g.Lookup("8.8.8.8")
	if err != nil || record.Region != "Xinjiang Uygur" {
		t.Fatalf("lookup failed: %v %#v", err, record)
	}
	actual, err := os.ReadFile(g.file)
	if err != nil || !bytes.Equal(actual, payload) {
		t.Fatal("downloaded database differs")
	}
	entries, err := os.ReadDir(filepath.Dir(g.file))
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files left: %#v %v", entries, err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(filepath.Dir(g.file)), "outside.BIN")); !os.IsNotExist(err) {
		t.Fatal("ZIP path escaped the database directory")
	}
}

func TestGeoIPUpdateRejectsInvalidAndOlderContentPreservingCache(t *testing.T) {
	var payload atomic.Value
	payload.Store(geoIPTestBIN(2, "Old region"))
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(payload.Load().([]byte)) }))
	defer source.Close()
	g := newTestGeoIP(t, source.URL)
	if err := g.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(g.file)
	malformedColumns := geoIPTestBIN(3, "Invalid columns")
	malformedColumns[0] = 26
	for _, data := range [][]byte{[]byte("NO PERMISSION download token private"), make([]byte, 128), geoIPTestBIN(1, "Older region"), geoIPTestBIN(3, "New region")[:70], malformedColumns} {
		payload.Store(data)
		if err := g.Update(context.Background()); err == nil {
			t.Fatal("invalid update accepted")
		}
		after, _ := os.ReadFile(g.file)
		if !bytes.Equal(before, after) {
			t.Fatal("failed update replaced cache")
		}
		record, err := g.Lookup("8.8.8.8")
		if err != nil || record.Region != "Old region" {
			t.Fatalf("lost last good reader: %#v %v", record, err)
		}
	}
	payload.Store(geoIPTestBIN(3, "New region"))
	if err := g.Update(context.Background()); err != nil {
		t.Fatal(err)
	}
	record, err := g.Lookup("8.8.8.8")
	if err != nil || record.Region != "New region" {
		t.Fatalf("new reader not applied: %#v %v", record, err)
	}
}

func TestGeoIPCachedStartupDoesNotRequireNetworkAnd304RefreshesCheck(t *testing.T) {
	var requests atomic.Int32
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("If-Modified-Since") == "" {
			t.Error("missing conditional request")
		}
		w.WriteHeader(http.StatusNotModified)
	}))
	defer source.Close()
	g := newTestGeoIP(t, source.URL)
	os.MkdirAll(filepath.Dir(g.file), 0700)
	if err := os.WriteFile(g.file, geoIPTestBIN(1, "Cache"), 0600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	os.Chtimes(g.file, old, old)
	if err := g.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 0 {
		t.Fatal("cached startup waited for download")
	}
	if err := g.Update(context.Background()); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(g.file)
	if time.Since(info.ModTime()) > time.Minute {
		t.Fatal("304 did not refresh update check")
	}
}

func TestGeoIPRequiresUsableCacheAndRedactsDownloadCredentials(t *testing.T) {
	g := newTestGeoIP(t, "")
	if err := g.Prepare(context.Background()); err == nil {
		t.Fatal("started without geographic policy data")
	}
	g.source = "http://127.0.0.1:1/database?token=secret-download-token"
	if err := g.Update(context.Background()); err == nil || strings.Contains(err.Error(), "secret-download-token") {
		t.Fatalf("unsafe download error: %v", err)
	}
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, "token=secret-download-token")
	}))
	defer source.Close()
	g.source = source.URL + "?token=secret-download-token"
	if err := g.Update(context.Background()); err == nil || strings.Contains(err.Error(), "secret-download-token") {
		t.Fatalf("unsafe response error: %v", err)
	}
}

func TestGeoIPHotReloadIsSafeDuringLookups(t *testing.T) {
	payload := geoIPTestZIP(t, "database.BIN", geoIPTestBIN(2, "New region"))
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(payload) }))
	defer source.Close()
	g := newTestGeoIP(t, source.URL)
	if err := g.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if _, err := g.Lookup("8.8.8.8"); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	for i := 0; i < 3; i++ {
		if err := g.Update(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
}

func TestGeoIPContentLimitsAndArchiveValidation(t *testing.T) {
	if _, err := copyGeoIPBytes(io.Discard, strings.NewReader("12345"), 4); err == nil {
		t.Fatal("accepted oversized content")
	}
	g := newTestGeoIP(t, "")
	os.MkdirAll(filepath.Dir(g.file), 0700)
	for _, data := range [][]byte{geoIPTestZIP(t, "README.txt", []byte("not a BIN")), []byte("PK\x03\x04broken")} {
		path := filepath.Join(filepath.Dir(g.file), "archive.zip")
		os.WriteFile(path, data, 0600)
		if _, cleanup, err := geoIPCandidate(path, filepath.Dir(g.file)); err == nil {
			cleanup()
			t.Fatal("accepted invalid ZIP")
		}
	}
}

func TestGeoIPRunUpdatesStaleCacheInBackground(t *testing.T) {
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(geoIPTestBIN(2, "Background update")) }))
	defer source.Close()
	g := newTestGeoIP(t, source.URL)
	os.MkdirAll(filepath.Dir(g.file), 0700)
	if err := os.WriteFile(g.file, geoIPTestBIN(1, "Cached region"), 0600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	os.Chtimes(g.file, old, old)
	if err := g.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); g.Run(ctx) }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		record, err := g.Lookup("8.8.8.8")
		if err != nil {
			t.Fatal(err)
		}
		if record.Region == "Background update" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("background update was not installed")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("updater did not stop")
	}
}

func TestGeoIPHTTPSRedirectDoesNotForwardTokenInReferer(t *testing.T) {
	payload := geoIPTestZIP(t, "db.BIN", geoIPTestBIN(1, "Region"))
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Referer") != "" || r.URL.Query().Get("token") != "" {
			t.Error("download token leaked to redirected storage")
		}
		w.Write(payload)
	}))
	defer target.Close()
	source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer source.Close()
	g := newTestGeoIP(t, source.URL+"?token=private-download-token")
	roots := x509.NewCertPool()
	roots.AddCert(source.Certificate())
	roots.AddCert(target.Certificate())
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}
	defer transport.CloseIdleConnections()
	g.client.Transport = transport
	if err := g.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestGeoIPRejectsHTTPSDowngrade(t *testing.T) {
	var reached atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached.Add(1) }))
	defer target.Close()
	source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer source.Close()
	g := newTestGeoIP(t, source.URL)
	transport := source.Client().Transport.(*http.Transport)
	defer transport.CloseIdleConnections()
	g.client.Transport = transport
	if err := g.Prepare(context.Background()); err == nil {
		t.Fatal("accepted HTTPS downgrade")
	}
	if reached.Load() != 0 {
		t.Fatal("followed unencrypted redirect")
	}
}
