package log

import (
	"context"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnrichSystemLogFieldsStoresStructuredActivityFields(t *testing.T) {
	loginContent, _ := (&Login{
		Method:    "email",
		LoginIP:   "203.0.113.10",
		UserAgent: "AlphaClient/1.0",
		Success:   true,
		Timestamp: 1717200000123,
	}).Marshal()
	loginLog := &SystemLog{Type: TypeLogin.Uint8(), Content: string(loginContent)}
	enrichSystemLogFields(loginLog)
	if loginLog.Method != "email" || loginLog.IP != "203.0.113.10" || loginLog.UserAgent != "AlphaClient/1.0" || !loginLog.Success || loginLog.EventAt != 1717200000123 {
		t.Fatalf("login structured fields not enriched: %#v", loginLog)
	}

	registerContent, _ := (&Register{
		AuthMethod: "github",
		Identifier: "oauth-user-id",
		RegisterIP: "203.0.113.20",
		UserAgent:  "AlphaRegister/1.0",
		Timestamp:  1717200000456,
	}).Marshal()
	registerLog := &SystemLog{Type: TypeRegister.Uint8(), Content: string(registerContent)}
	enrichSystemLogFields(registerLog)
	if registerLog.Method != "github" || registerLog.Identifier != "oaut...r-id" || registerLog.IP != "203.0.113.20" || registerLog.UserAgent != "AlphaRegister/1.0" || registerLog.EventAt != 1717200000456 {
		t.Fatalf("register structured fields not enriched: %#v", registerLog)
	}
}

func TestInsertStoresOnlyMaskedNetworkIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SystemLog{}, &SubscribeAccessLog{}); err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, Hooks{
		LookupIPLocation: func(_ context.Context, ip string) IPLocation {
			if ip != "203.0.113.42" {
				t.Fatalf("GeoIP lookup received %q, want raw IP", ip)
			}
			return IPLocation{CountryShort: "US", Region: "CO"}
		},
	})
	content, _ := (&Login{
		Method:    "email",
		LoginIP:   "203.0.113.42",
		UserAgent: "AlphaClient/1.0 (private device details)",
		Success:   true,
	}).Marshal()
	entry := &SystemLog{Type: TypeLogin.Uint8(), Identifier: "user@example.com", Content: string(content)}
	if err := model.Insert(context.Background(), entry); err != nil {
		t.Fatal(err)
	}

	var stored SystemLog
	if err := db.First(&stored, entry.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.IP != "203.0.113.*" || stored.UserAgent != "AlphaClient/***" || stored.Identifier != "u**r@example.com" {
		t.Fatalf("stored network identity was not masked: %#v", stored)
	}
	if strings.Contains(stored.Content, "203.0.113.42") || strings.Contains(stored.Content, "private device details") {
		t.Fatalf("stored content leaked network identity: %s", stored.Content)
	}
	if stored.Location != "US · CO" {
		t.Fatalf("stored location = %q, want US · CO", stored.Location)
	}
}

func TestEnrichSystemLogFieldsPreservesExplicitValues(t *testing.T) {
	content, _ := (&Login{
		Method:    "email",
		LoginIP:   "203.0.113.10",
		UserAgent: "AlphaClient/1.0",
		Success:   true,
		Timestamp: 1717200000123,
	}).Marshal()
	systemLog := &SystemLog{
		Type:      TypeLogin.Uint8(),
		Content:   string(content),
		IP:        "198.51.100.7",
		Method:    "device",
		UserAgent: "ExistingAgent/1.0",
		EventAt:   1717000000000,
	}
	enrichSystemLogFields(systemLog)
	if systemLog.IP != "198.51.100.7" || systemLog.Method != "device" || systemLog.UserAgent != "ExistingAgent/1.0" || systemLog.EventAt != 1717000000000 {
		t.Fatalf("explicit fields overwritten: %#v", systemLog)
	}
	if !systemLog.Success {
		t.Fatalf("success should still be populated from content: %#v", systemLog)
	}
}

func TestFormatIPLocationKeepsCompactShape(t *testing.T) {
	tests := []struct {
		country string
		region  string
		want    string
	}{
		{country: " US ", region: " CA ", want: "US · CA"},
		{country: "US", region: "", want: "US"},
		{country: "", region: "CA", want: "CA"},
		{country: " ", region: " ", want: ""},
	}
	for _, tt := range tests {
		if got := formatIPLocation(tt.country, tt.region); got != tt.want {
			t.Fatalf("formatIPLocation(%q, %q) = %q, want %q", tt.country, tt.region, got, tt.want)
		}
	}
}

func TestFormatIPLocationSinglePartDoesNotAllocate(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		_ = formatIPLocation("US", "")
	})
	if allocs != 0 {
		t.Fatalf("formatIPLocation single part allocs = %v, want 0", allocs)
	}
}
