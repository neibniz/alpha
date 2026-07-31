package user

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	privacycodec "github.com/neibniz/alpha/server/internal/privacy"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDevicePrivacyEncryptsStorageAndKeepsInternalLookup(t *testing.T) {
	db, model, mini := newDevicePrivacyTestModel(t)
	ctx := context.Background()
	device := &Device{
		UserId:     7,
		Ip:         "192.0.2.44",
		Identifier: "device-private-identifier",
		UserAgent:  "AlphaClient/1.2.3 (private platform details)",
		Enabled:    true,
	}
	if err := model.InsertDevice(ctx, device); err != nil {
		t.Fatal(err)
	}

	var stored Device
	if err := db.First(&stored, device.Id).Error; err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"identifier": stored.Identifier,
		"IP":         stored.Ip,
		"user agent": stored.UserAgent,
	} {
		if !strings.HasPrefix(value, "enc:v1:k1:") {
			t.Fatalf("stored %s was not encrypted: %q", name, value)
		}
	}
	if stored.IdentifierBlindIndex == "" || stored.IPDisplay != "192.0.2.*" || stored.UserAgentDisplay != "AlphaClient/***" {
		t.Fatalf("device privacy projections are invalid: %#v", stored)
	}

	lookup, err := model.FindOneDeviceByIdentifier(ctx, device.Identifier)
	if err != nil {
		t.Fatal(err)
	}
	if lookup.Identifier != device.Identifier || lookup.Ip != device.Ip || lookup.UserAgent != device.UserAgent {
		t.Fatalf("internal lookup did not decrypt device: %#v", lookup)
	}
	for _, key := range mini.Keys() {
		if strings.Contains(key, device.Identifier) {
			t.Fatalf("Redis key leaked device identifier: %q", key)
		}
	}
}

func TestMigrateDevicePrivacyBackfillsLegacyRows(t *testing.T) {
	db, model, _ := newDevicePrivacyTestModel(t)
	legacy := Device{
		UserId:     9,
		Ip:         "198.51.100.8",
		Identifier: "legacy-device-identifier",
		UserAgent:  "LegacyClient/4.5.6",
		Enabled:    true,
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}

	migrated, err := model.MigrateDevicePrivacy(context.Background(), 10)
	if err != nil || migrated != 1 {
		t.Fatalf("MigrateDevicePrivacy() = %d, %v", migrated, err)
	}
	var stored Device
	if err := db.First(&stored, legacy.Id).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored.Identifier, "enc:v1:k1:") || stored.IdentifierBlindIndex == "" || stored.IPDisplay != "198.51.100.*" {
		t.Fatalf("legacy device was not migrated: %#v", stored)
	}
	lookup, err := model.FindOneDeviceByIdentifier(context.Background(), legacy.Identifier)
	if err != nil || lookup.Id != legacy.Id || lookup.Ip != legacy.Ip {
		t.Fatalf("migrated device lookup = %#v, %v", lookup, err)
	}
}

func newDevicePrivacyTestModel(t *testing.T) (*gorm.DB, Model, *miniredis.Miniredis) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE user_device (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ip TEXT NOT NULL,
		ip_display TEXT NOT NULL DEFAULT '',
		user_id INTEGER NOT NULL,
		user_agent TEXT NOT NULL DEFAULT '',
		user_agent_display TEXT NOT NULL DEFAULT '',
		identifier TEXT NOT NULL DEFAULT '',
		identifier_bidx TEXT NOT NULL DEFAULT '',
		identifier_display TEXT NOT NULL DEFAULT '',
		online BOOLEAN NOT NULL DEFAULT FALSE,
		enabled BOOLEAN NOT NULL DEFAULT TRUE,
		created_at DATETIME,
		updated_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	indexKey := base64.StdEncoding.EncodeToString([]byte("abcdef0123456789abcdef0123456789"))
	codec, err := privacycodec.NewCodec(privacycodec.Keyring{
		ActiveKeyID:   "k1",
		EncryptionKey: map[string]string{"k1": key},
		IndexKey:      indexKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	return db, NewModel(db, rds, codec), mini
}
