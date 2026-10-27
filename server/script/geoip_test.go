package script_test

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallGeoIP(t *testing.T) {
	for _, tool := range []string{"sh", "gzip", "sha256sum"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is required for the shell installer tests", tool)
		}
	}
	script, err := os.ReadFile("install-geoip-database.sh")
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name          string
		cached        bool
		corruptCache  bool
		downloadFails bool
		overrideURL   bool
		wrongChecksum bool
		invalidGzip   bool
		small         bool
		wantError     string
	}{
		{name: "downloads verified archive"},
		{name: "uses valid cache without downloading", cached: true, downloadFails: true},
		{name: "repairs corrupt cache", cached: true, corruptCache: true},
		{name: "supports checksum-pinned mirror", overrideURL: true},
		{name: "failed download preserves database", downloadFails: true, wantError: "GeoIP download failed"},
		{name: "checksum mismatch", wrongChecksum: true, wantError: "checksum mismatch"},
		{name: "invalid gzip", invalidGzip: true, wantError: "gzip"},
		{name: "undersized database", small: true, wantError: "unexpected GeoIP database size"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			scriptDir := filepath.Join(root, "script")
			assetDir := filepath.Join(root, "assets", "geoip")
			binDir := filepath.Join(root, "bin")
			cacheDir := filepath.Join(root, "cache with spaces")
			outputDir := filepath.Join(root, "output with spaces")
			for _, dir := range []string{scriptDir, assetDir, binDir, cacheDir, outputDir} {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			writeFile(t, filepath.Join(scriptDir, "install-geoip-database.sh"), script, 0o644)
			writeFile(t, filepath.Join(binDir, "curl"), []byte(fakeCurl), 0o755)
			payload := bytes.Repeat([]byte("x"), 1048576)
			if test.small {
				payload = payload[:1048575]
			}
			var compressed bytes.Buffer
			writer := gzip.NewWriter(&compressed)
			if _, err := writer.Write(payload); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			archive := compressed.Bytes()
			if test.invalidGzip {
				archive = []byte("not gzip")
			}
			checksum := fmt.Sprintf("%x", sha256.Sum256(archive))
			if test.wrongChecksum {
				checksum = strings.Repeat("0", 64)
			}
			manifest := fmt.Sprintf("ALPHA_GEOIP_DATABASE_VERSION=test\nALPHA_GEOIP_DATABASE_URL=https://example.invalid/unavailable.gz\nALPHA_GEOIP_DATABASE_SHA256=%s\n", checksum)
			writeFile(t, filepath.Join(assetDir, "dbip-city-lite.env"), []byte(manifest), 0o644)
			cacheFile := filepath.Join(cacheDir, "dbip-city-lite-test-"+checksum+".mmdb.gz")
			if test.cached {
				cached := archive
				if test.corruptCache {
					cached = []byte("corrupt cached data")
				}
				writeFile(t, cacheFile, cached, 0o644)
			}
			source := filepath.Join(root, "download.gz")
			writeFile(t, source, archive, 0o644)
			log := filepath.Join(root, "download.log")
			destination := filepath.Join(outputDir, "dbip-city-lite.mmdb")
			writeFile(t, destination, []byte("existing database"), 0o600)
			cmd := exec.Command("sh", filepath.Join(scriptDir, "install-geoip-database.sh"), destination)
			sourceURL := ""
			if test.overrideURL {
				sourceURL = "https://mirror.example.com/database.gz"
			}
			cmd.Env = append(os.Environ(),
				"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
				"ALPHA_GEOIP_CACHE_DIR="+cacheDir,
				"ALPHA_GEOIP_SOURCE_URL="+sourceURL,
				"ALPHA_TEST_DOWNLOAD_SOURCE="+source,
				"ALPHA_TEST_DOWNLOAD_LOG="+log,
				fmt.Sprintf("ALPHA_TEST_DOWNLOAD_FAILURE=%t", test.downloadFails),
			)
			output, err := cmd.CombinedOutput()
			downloadLog, _ := os.ReadFile(log)
			if test.cached && !test.corruptCache {
				if len(downloadLog) != 0 {
					t.Fatalf("valid cache triggered a download: %s", downloadLog)
				}
			} else {
				expectedURL := "https://example.invalid/unavailable.gz"
				if test.overrideURL {
					expectedURL = sourceURL
				}
				if strings.TrimSpace(string(downloadLog)) != expectedURL {
					t.Fatalf("unexpected download source: %q", downloadLog)
				}
			}
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("installer failed: %v\n%s", err, output)
				}
				actual, err := os.ReadFile(destination)
				if err != nil || !bytes.Equal(actual, payload) {
					t.Fatalf("installed database differs: %v", err)
				}
				info, err := os.Stat(destination)
				if err != nil || info.Mode().Perm() != 0o644 {
					t.Fatalf("unexpected database permissions: %v", err)
				}
				cached, err := os.ReadFile(cacheFile)
				if err != nil || !bytes.Equal(cached, archive) {
					t.Fatalf("verified download was not cached: %v", err)
				}
			} else {
				if err == nil || !strings.Contains(string(output), test.wantError) {
					t.Fatalf("expected %q failure, got %v\n%s", test.wantError, err, output)
				}
				actual, readErr := os.ReadFile(destination)
				if readErr != nil || string(actual) != "existing database" {
					t.Fatalf("failed installation replaced the existing database: %v", readErr)
				}
				if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
					t.Fatalf("failed download or installation populated the cache: %v", err)
				}
			}
			for _, dir := range []string{outputDir, cacheDir} {
				leftovers, err := filepath.Glob(filepath.Join(dir, ".alpha-geoip*"))
				if err != nil || len(leftovers) != 0 {
					t.Fatalf("temporary files were not cleaned up: %v, %v", leftovers, err)
				}
			}
		})
	}
}

const fakeCurl = `#!/bin/sh
set -eu
output=''
url=''
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output) output=$2; shift 2 ;;
    *) url=$1; shift ;;
  esac
done
printf '%s\n' "$url" >> "$ALPHA_TEST_DOWNLOAD_LOG"
if [ "$ALPHA_TEST_DOWNLOAD_FAILURE" = true ]; then exit 22; fi
cp "$ALPHA_TEST_DOWNLOAD_SOURCE" "$output"
`

func writeFile(t *testing.T, path string, content []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatal(err)
	}
}
