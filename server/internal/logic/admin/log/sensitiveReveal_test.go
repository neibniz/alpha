package log

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	modelLog "github.com/neibniz/alpha/server/internal/model/log"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	privacycodec "github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/constant"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testSensitiveRevealCodec(t *testing.T) *privacycodec.Codec {
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

func sensitiveRevealAdminContext(admin bool) context.Context {
	ctx := context.WithValue(context.Background(), constant.CtxKeyUser, &modelUser.User{
		Id:      42,
		IsAdmin: &admin,
	})
	ctx = context.WithValue(ctx, constant.CtxKeyClientIP, "203.0.113.42")
	return context.WithValue(ctx, constant.CtxKeyUserAgent, "AdminConsole/1.0 private-details")
}

func TestPrepareSensitiveLogRevealRequiresAdminAndReason(t *testing.T) {
	svcCtx := &svc.ServiceContext{Privacy: testSensitiveRevealCodec(t)}
	if _, err := prepareSensitiveLogReveal(sensitiveRevealAdminContext(true), svcCtx, true, " security_incident ", "login"); err != nil {
		t.Fatalf("valid reveal rejected: %v", err)
	}
	if _, err := prepareSensitiveLogReveal(sensitiveRevealAdminContext(true), svcCtx, true, "", "login"); err == nil {
		t.Fatal("reveal without reason was accepted")
	}
	if _, err := prepareSensitiveLogReveal(sensitiveRevealAdminContext(false), svcCtx, true, "security_incident", "login"); err == nil {
		t.Fatal("non-admin reveal was accepted")
	}
	if _, err := prepareSensitiveLogReveal(context.Background(), &svc.ServiceContext{Privacy: privacycodec.DisabledCodec()}, true, "security_incident", "login"); err == nil {
		t.Fatal("reveal without encrypted storage was accepted")
	}
}

func TestSensitiveLogRevealWritesMaskedAuditRecord(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelLog.SystemLog{}); err != nil {
		t.Fatal(err)
	}
	codec := testSensitiveRevealCodec(t)
	svcCtx := &svc.ServiceContext{
		Privacy:  codec,
		LogModel: modelLog.NewModel(db, modelLog.Hooks{Privacy: codec}),
	}
	reveal, err := prepareSensitiveLogReveal(sensitiveRevealAdminContext(true), svcCtx, true, " security_incident ", "login")
	if err != nil {
		t.Fatal(err)
	}
	if err := reveal.audit(3); err != nil {
		t.Fatal(err)
	}

	var stored modelLog.SystemLog
	if err := db.Where("type = ?", modelLog.TypeSensitiveLogReveal.Uint8()).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.IP != "203.0.113.*" || stored.UserAgent != "AdminConsole/***" {
		t.Fatalf("audit network fields were not masked: %#v", stored)
	}
	var content modelLog.SensitiveLogReveal
	if err := content.Unmarshal([]byte(stored.Content)); err != nil {
		t.Fatal(err)
	}
	if content.OperatorUserId != 42 || content.Scope != "login" || content.Reason != "security_incident" || content.RecordCount != 3 {
		t.Fatalf("unexpected audit content: %#v", content)
	}
}
