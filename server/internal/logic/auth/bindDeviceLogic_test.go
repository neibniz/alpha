package auth

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/alicebob/miniredis/v2"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	privacycodec "github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateDeviceForUserEncryptsPrivateFields(t *testing.T) {
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
	logic := NewBindDeviceLogic(context.Background(), &svc.ServiceContext{
		UserModel: modelUser.NewModel(db, rds, codec),
	})
	const (
		identifier = "private-device-id"
		ip         = "192.0.2.91"
		userAgent  = "AlphaClient/2.1 (private platform details)"
	)
	if err := logic.createDeviceForUser(identifier, ip, userAgent, 42); err != nil {
		t.Fatal(err)
	}

	var stored modelUser.Device
	if err := db.First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if !privacycodec.IsSealed(stored.Identifier) || !privacycodec.IsSealed(stored.Ip) || !privacycodec.IsSealed(stored.UserAgent) {
		t.Fatalf("device private fields were stored in plaintext: %#v", stored)
	}
	if stored.IdentifierBlindIndex == "" || stored.IPDisplay != "192.0.2.*" || stored.UserAgentDisplay != "AlphaClient/***" {
		t.Fatalf("device privacy projections are invalid: %#v", stored)
	}
}
