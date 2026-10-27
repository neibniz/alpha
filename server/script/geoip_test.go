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

func TestInstallBundledGeoIP(t *testing.T) {
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
		missing       bool
		wrongChecksum bool
		invalidGzip   bool
		small         bool
		wantError     string
	}{
		{name: "offline installation"},
		{name: "missing archive", missing: true, wantError: "archive is missing"},
		{name: "checksum mismatch", wrongChecksum: true, wantError: "checksum mismatch"},
		{name: "invalid gzip", invalidGzip: true, wantError: "gzip"},
		{name: "undersized database", small: true, wantError: "unexpected GeoIP database size"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			scriptDir := filepath.Join(root, "script")
			assetDir := filepath.Join(root, "assets", "geoip")
			binDir := filepath.Join(root, "bin")
			outputDir := filepath.Join(root, "output with spaces")
			for _, dir := range []string{scriptDir, assetDir, binDir, outputDir} {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			writeFile(t, filepath.Join(scriptDir, "install-geoip-database.sh"), script, 0o644)
			writeFile(t, filepath.Join(binDir, "curl"), []byte("#!/bin/sh\necho unexpected-network-access >&2\nexit 99\n"), 0o755)
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
			if !test.missing {
				writeFile(t, filepath.Join(assetDir, "dbip-city-lite.mmdb.gz"), archive, 0o644)
			}
			destination := filepath.Join(outputDir, "dbip-city-lite.mmdb")
			writeFile(t, destination, []byte("existing database"), 0o600)
			cmd := exec.Command("sh", filepath.Join(scriptDir, "install-geoip-database.sh"), destination)
			cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			output, err := cmd.CombinedOutput()
			if strings.Contains(string(output), "unexpected-network-access") {
				t.Fatalf("installer attempted a network download: %s", output)
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
			} else {
				if err == nil || !strings.Contains(string(output), test.wantError) {
					t.Fatalf("expected %q failure, got %v\n%s", test.wantError, err, output)
				}
				actual, readErr := os.ReadFile(destination)
				if readErr != nil || string(actual) != "existing database" {
					t.Fatalf("failed installation replaced the existing database: %v", readErr)
				}
			}
			leftovers, err := filepath.Glob(filepath.Join(outputDir, ".alpha-geoip.*"))
			if err != nil || len(leftovers) != 0 {
				t.Fatalf("temporary files were not cleaned up: %v, %v", leftovers, err)
			}
		})
	}
}

func writeFile(t *testing.T, path string, content []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatal(err)
	}
}
