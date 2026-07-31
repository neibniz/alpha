package log

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	privacycodec "github.com/neibniz/alpha/server/internal/privacy"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testLogPrivacyCodec(t *testing.T) *privacycodec.Codec {
	t.Helper()
	encode := func(value byte) string {
		return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{value}, 32))
	}
	codec, err := privacycodec.NewCodec(privacycodec.Keyring{
		ActiveKeyID:   "test",
		EncryptionKey: map[string]string{"test": encode(0x31)},
		IndexKey:      encode(0x52),
	})
	if err != nil {
		t.Fatal(err)
	}
	return codec
}

func TestSensitiveLogFieldsAreEncryptedAndRevealedOnDemand(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SystemLog{}, &SubscribeAccessLog{}); err != nil {
		t.Fatal(err)
	}
	model := NewModel(db, Hooks{Privacy: testLogPrivacyCodec(t)})

	loginContent, _ := (&Login{
		Method:    "email",
		LoginIP:   "203.0.113.42",
		UserAgent: "AlphaClient/1.0 (private details)",
		Success:   true,
	}).Marshal()
	entry := &SystemLog{Type: TypeLogin.Uint8(), ObjectID: 7, Content: string(loginContent)}
	if err := model.Insert(context.Background(), entry); err != nil {
		t.Fatal(err)
	}

	var stored SystemLog
	if err := db.First(&stored, entry.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.IP != "203.0.113.*" || stored.UserAgent != "AlphaClient/***" {
		t.Fatalf("display fields were not masked: %#v", stored)
	}
	if !privacycodec.IsSealed(stored.PrivateData) || stored.PrivacyRef == "" {
		t.Fatalf("private fields were not encrypted: %#v", stored)
	}
	if strings.Contains(stored.PrivateData, "203.0.113.42") || strings.Contains(stored.PrivateData, "private details") {
		t.Fatalf("encrypted private data contains plaintext: %q", stored.PrivateData)
	}
	serialized, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(serialized), "PrivateData") || strings.Contains(string(serialized), "PrivacyRef") || strings.Contains(string(serialized), stored.PrivateData) {
		t.Fatalf("serialized log leaked private storage fields: %s", serialized)
	}

	masked, _, err := model.FilterLoginLogs(context.Background(), &FilterParams{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(masked) != 1 || masked[0].LoginIP != "203.0.113.*" || masked[0].Revealed {
		t.Fatalf("default query did not stay masked: %#v", masked)
	}

	revealed, _, err := model.FilterLoginLogs(context.Background(), &FilterParams{Page: 1, Size: 10, RevealSensitive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(revealed) != 1 || revealed[0].LoginIP != "203.0.113.42" || revealed[0].UserAgent != "AlphaClient/1.0 (private details)" || !revealed[0].Revealed {
		t.Fatalf("revealed query did not decrypt fields: %#v", revealed)
	}
}

func TestSubscribeAccessPrivateFieldsExcludeSubscriptionToken(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeAccessLog{}); err != nil {
		t.Fatal(err)
	}
	model := NewModel(db, Hooks{Privacy: testLogPrivacyCodec(t)})
	entry := &SubscribeAccessLog{
		UserId:          7,
		UserSubscribeId: 9,
		TokenHash:       HashSubscribeToken("private-subscription-token"),
		TokenPrefix:     TokenPrefix("private-subscription-token"),
		ClientIP:        "198.51.100.20",
		UserAgent:       "Clash/2.0 (private details)",
	}
	if err := model.InsertSubscribeAccessLog(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	privatePayload, err := model.(*customSystemLogModel).privacyCodec.Open(
		subscribeAccessPrivacyScope(entry.PrivacyRef),
		entry.PrivateData,
	)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(privatePayload, "private-subscription-token") {
		t.Fatal("subscription credential was stored in reversible access-log data")
	}

	rows, _, err := model.FilterSubscribeAccessLog(context.Background(), &SubscribeAccessFilterParams{
		Page: 1, Size: 10, RevealSensitive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ClientIP != "198.51.100.20" || rows[0].UserAgent != "Clash/2.0 (private details)" || !rows[0].Revealed {
		t.Fatalf("subscribe access fields were not revealed: %#v", rows)
	}
}

func TestSystemSubscribePrivateFieldsExcludeSubscriptionToken(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SystemLog{}); err != nil {
		t.Fatal(err)
	}
	model := NewModel(db, Hooks{Privacy: testLogPrivacyCodec(t)})
	content, _ := (&Subscribe{
		Token:           "private-subscription-token",
		ClientIP:        "198.51.100.21",
		UserAgent:       "Clash/2.1 (private details)",
		UserSubscribeId: 9,
	}).Marshal()
	entry := &SystemLog{Type: TypeSubscribe.Uint8(), ObjectID: 7, Content: string(content)}
	if err := model.Insert(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	private, err := model.(*customSystemLogModel).openSystemLogPrivate(entry.PrivacyRef, entry.PrivateData)
	if err != nil {
		t.Fatal(err)
	}
	if private == nil || private.IP != "198.51.100.21" || private.UserAgent != "Clash/2.1 (private details)" {
		t.Fatalf("system subscribe private fields = %#v", private)
	}
	if private.Content != "" || strings.Contains(private.Content, "private-subscription-token") {
		t.Fatal("subscription credential was stored in reversible system-log data")
	}
}
