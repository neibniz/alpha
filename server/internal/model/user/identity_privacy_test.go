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

func newIdentityPrivacyTestModel(t *testing.T) (*gorm.DB, Model, *miniredis.Miniredis) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
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

func TestIdentityPrivacyEncryptsStorageAndUsesMaskedProjection(t *testing.T) {
	db, model, mini := newIdentityPrivacyTestModel(t)
	ctx := context.Background()
	data := &User{
		Password: "password-hash",
		AuthMethods: []AuthMethods{{
			AuthType:       "email",
			AuthIdentifier: "User@Example.com",
			Verified:       true,
		}},
	}
	if err := model.Insert(ctx, data); err != nil {
		t.Fatal(err)
	}

	var stored AuthMethods
	if err := db.First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored.AuthIdentifier, "enc:v1:k1:") || len(stored.AuthIdentifierBlindIndex) != 64 {
		t.Fatalf("identity was not encrypted/indexed: %#v", stored)
	}
	if stored.AuthIdentifierDisplay != "u**r@example.com" || strings.Contains(stored.AuthIdentifier, "example.com") {
		t.Fatalf("unexpected stored identity projection: %#v", stored)
	}

	lookup, err := model.FindUserAuthMethodByOpenID(ctx, "email", "user@example.com")
	if err != nil || lookup.AuthIdentifier != "user@example.com" {
		t.Fatalf("lookup = %#v, %v", lookup, err)
	}
	summaries, err := model.FindUserAuthMethodSummaries(ctx, data.Id)
	if err != nil || len(summaries) != 1 || summaries[0].AuthIdentifier != "u**r@example.com" {
		t.Fatalf("summaries = %#v, %v", summaries, err)
	}
	if _, err := model.FindOneForLoginByEmail(ctx, "user@example.com"); err != nil {
		t.Fatal(err)
	}
	for _, key := range mini.Keys() {
		if strings.Contains(strings.ToLower(key), "user@example.com") {
			t.Fatalf("redis key exposes email: %q", key)
		}
	}
}

func TestMigratePrivacyBackfillsLegacyIdentity(t *testing.T) {
	db, model, _ := newIdentityPrivacyTestModel(t)
	legacy := AuthMethods{UserId: 9, AuthType: "mobile", AuthIdentifier: "+15551234567", Verified: true}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	migrated, err := model.MigratePrivacy(context.Background(), 10)
	if err != nil || migrated != 1 {
		t.Fatalf("MigratePrivacy() = %d, %v", migrated, err)
	}
	var stored AuthMethods
	if err := db.First(&stored, legacy.Id).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored.AuthIdentifier, "enc:v1:k1:") || stored.AuthIdentifierBlindIndex == "" {
		t.Fatalf("legacy identity was not migrated: %#v", stored)
	}
	lookup, err := model.FindUserAuthMethodByOpenID(context.Background(), "mobile", "+15551234567")
	if err != nil || lookup.UserId != 9 {
		t.Fatalf("migrated lookup = %#v, %v", lookup, err)
	}
}

func TestIdentityPrivacySeparatesDisplayContextFromNotificationReveal(t *testing.T) {
	_, model, _ := newIdentityPrivacyTestModel(t)
	ctx := context.Background()
	admin := true
	enabled := true
	data := &User{
		Password: "password-hash",
		IsAdmin:  &admin,
		Enable:   &enabled,
		AuthMethods: []AuthMethods{{
			AuthType:       "telegram",
			AuthIdentifier: "1234567890",
			Verified:       true,
		}},
	}
	if err := model.Insert(ctx, data); err != nil {
		t.Fatal(err)
	}

	contextUser, err := model.FindOneForContext(ctx, data.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(contextUser.AuthMethods) != 1 || contextUser.AuthMethods[0].AuthIdentifier != "1234...7890" {
		t.Fatalf("context auth method must use masked display: %#v", contextUser.AuthMethods)
	}

	activationUser, err := model.FindOneForOrderActivation(ctx, data.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(activationUser.AuthMethods) != 1 || activationUser.AuthMethods[0].AuthIdentifier != "1234567890" {
		t.Fatalf("notification auth method was not decrypted: %#v", activationUser.AuthMethods)
	}

	admins, err := model.QueryAdminUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(admins) != 1 || len(admins[0].AuthMethods) != 1 || admins[0].AuthMethods[0].AuthIdentifier != "1234567890" {
		t.Fatalf("admin notification auth method was not decrypted: %#v", admins)
	}
}
