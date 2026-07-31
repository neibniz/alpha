package user

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	modelLog "github.com/neibniz/alpha/server/internal/model/log"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRevealUserPrivateIdentityDecryptsAllMethodsAndAuditsAccess(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "private-identity.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelUser.User{}, &modelUser.AuthMethods{}, &modelLog.SystemLog{}); err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })

	codec, err := privacy.NewCodec(privacy.Keyring{
		ActiveKeyID: "k1",
		EncryptionKey: map[string]string{
			"k1": base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
		},
		IndexKey: base64.StdEncoding.EncodeToString([]byte("abcdef0123456789abcdef0123456789")),
	})
	if err != nil {
		t.Fatal(err)
	}
	userModel := modelUser.NewModel(db, rds, codec)
	admin := true
	enabled := true
	operator := &modelUser.User{
		Password:  "password-hash",
		IsAdmin:   &admin,
		Enable:    &enabled,
		ReferCode: "ADMIN",
	}
	target := &modelUser.User{
		Password:  "password-hash",
		Enable:    &enabled,
		ReferCode: "TARGET",
		AuthMethods: []modelUser.AuthMethods{
			{AuthType: "github", AuthIdentifier: "github-user-42", Verified: true},
			{AuthType: "email", AuthIdentifier: "GitHub.User@Example.test", Verified: true},
		},
	}
	if err := userModel.Insert(context.Background(), operator); err != nil {
		t.Fatal(err)
	}
	if err := userModel.Insert(context.Background(), target); err != nil {
		t.Fatal(err)
	}

	service := &svc.ServiceContext{
		DB:        db,
		Redis:     rds,
		UserModel: userModel,
		LogModel:  modelLog.NewModel(db),
	}
	ctx := context.WithValue(context.Background(), constant.CtxKeyUser, operator)
	response, err := NewRevealUserPrivateIdentityLogic(ctx, service).RevealUserPrivateIdentity(
		&types.RevealUserPrivateIdentityRequest{
			UserId: target.Id,
			Reason: "account_verification",
		},
		"203.0.113.42",
		"AdminBrowser/1.0 private-device",
	)
	if err != nil {
		t.Fatal(err)
	}
	if response.Email != "github.user@example.test" || len(response.AuthMethods) != 2 {
		t.Fatalf("private identity response = %#v", response)
	}
	if response.ExpiresAt-response.RevealedAt != privateIdentityRevealTTL.Milliseconds() {
		t.Fatalf("unexpected reveal lifetime: %#v", response)
	}

	var storedMethods []modelUser.AuthMethods
	if err := db.Where("user_id = ?", target.Id).Order("id ASC").Find(&storedMethods).Error; err != nil {
		t.Fatal(err)
	}
	for _, method := range storedMethods {
		if !privacy.IsSealed(method.AuthIdentifier) {
			t.Fatalf("identity was stored in plaintext: %#v", method)
		}
	}

	var audit modelLog.SystemLog
	if err := db.Where("type = ?", modelLog.TypePrivateDataReveal.Uint8()).First(&audit).Error; err != nil {
		t.Fatal(err)
	}
	if !audit.Success || audit.ObjectID != target.Id || strings.Contains(audit.Content, "example.test") || strings.Contains(audit.Content, "github-user-42") {
		t.Fatalf("unsafe or incomplete privacy audit: %#v", audit)
	}
	var detail modelLog.PrivateDataReveal
	if err := detail.Unmarshal([]byte(audit.Content)); err != nil {
		t.Fatal(err)
	}
	if detail.OperatorUserId != operator.Id || detail.TargetUserId != target.Id || detail.Reason != "account_verification" || len(detail.AuthTypes) != 2 {
		t.Fatalf("privacy audit detail = %#v", detail)
	}
}

func TestRevealUserPrivateIdentityRejectsNonAdmin(t *testing.T) {
	admin := false
	ctx := context.WithValue(context.Background(), constant.CtxKeyUser, &modelUser.User{Id: 7, IsAdmin: &admin})
	_, err := NewRevealUserPrivateIdentityLogic(ctx, &svc.ServiceContext{}).RevealUserPrivateIdentity(
		&types.RevealUserPrivateIdentityRequest{UserId: 8, Reason: "customer_support"},
		"",
		"",
	)
	if err == nil {
		t.Fatal("non-admin private identity access must fail")
	}
}

func TestRevealUserPrivateIdentityRejectsUnknownReason(t *testing.T) {
	admin := true
	ctx := context.WithValue(context.Background(), constant.CtxKeyUser, &modelUser.User{Id: 7, IsAdmin: &admin})
	_, err := NewRevealUserPrivateIdentityLogic(ctx, &svc.ServiceContext{}).RevealUserPrivateIdentity(
		&types.RevealUserPrivateIdentityRequest{UserId: 8, Reason: "free-form-reason"},
		"",
		"",
	)
	if err == nil {
		t.Fatal("unknown private identity reveal reason must fail")
	}
}
