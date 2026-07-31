package svc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewIPLocationDoesNotDownloadMissingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.mmdb")
	if _, err := NewIPLocation(path); err == nil {
		t.Fatal("expected a missing bundled database to fail")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("missing database was unexpectedly created: %v", err)
	}
}

func TestBundledGeoIPDatabase(t *testing.T) {
	path := os.Getenv("ALPHA_TEST_GEOIP_DATABASE")
	if path == "" {
		t.Skip("ALPHA_TEST_GEOIP_DATABASE is not set")
	}
	location, err := NewIPLocation(path)
	if err != nil {
		t.Fatal(err)
	}
	defer location.Close()

	info := location.Lookup("8.8.8.8")
	if info.CountryShort == "" || info.Country == "" {
		t.Fatalf("bundled database returned an incomplete location: %#v", info)
	}
}

func TestFormatIPLocation(t *testing.T) {
	if got := FormatIPLocation("US", "California"); got != "US · California" {
		t.Fatalf("FormatIPLocation() = %q", got)
	}
	if got := FormatIPLocation("", ""); got != "" {
		t.Fatalf("FormatIPLocation() empty = %q", got)
	}
}
