package user

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/neibniz/alpha/server/internal/config"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type adminUserSecurityEnv struct {
	db      *gorm.DB
	redis   *redis.Client
	service *svc.ServiceContext
}

func newAdminUserSecurityEnv(t *testing.T) adminUserSecurityEnv {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "admin-user.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelUser.User{}, &modelUser.AuthMethods{}); err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	return adminUserSecurityEnv{
		db:    db,
		redis: rds,
		service: &svc.ServiceContext{
			DB:        db,
			Redis:     rds,
			UserModel: modelUser.NewModel(db, rds, privacy.DisabledCodec()),
		},
	}
}

func createSecurityTestUser(t *testing.T, db *gorm.DB, id int64, enabled, admin bool) *modelUser.User {
	t.Helper()
	user := &modelUser.User{
		Id:        id,
		Password:  "encoded-password",
		Algo:      "default",
		Enable:    boolPointer(enabled),
		IsAdmin:   boolPointer(admin),
		ReferCode: "CODE",
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	return user
}

func boolPointer(value bool) *bool {
	return &value
}

func basicSecurityUpdate(userID int64, enabled, admin bool) *types.UpdateUserBasiceInfoRequest {
	return &types.UpdateUserBasiceInfoRequest{
		UserId:            userID,
		Enable:            enabled,
		IsAdmin:           admin,
		OnlyFirstPurchase: true,
	}
}

func TestUpdateUserBasicInfoRejectsCurrentAdminLockout(t *testing.T) {
	tests := []struct {
		name    string
		enable  bool
		isAdmin bool
	}{
		{name: "disable current admin", enable: false, isAdmin: true},
		{name: "demote current admin", enable: true, isAdmin: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := newAdminUserSecurityEnv(t)
			current := createSecurityTestUser(t, env.db, 1, true, true)
			ctx := context.WithValue(context.Background(), constant.CtxKeyUser, current)
			err := NewUpdateUserBasicInfoLogic(ctx, env.service).UpdateUserBasicInfo(
				basicSecurityUpdate(current.Id, test.enable, test.isAdmin),
			)
			if err == nil {
				t.Fatal("current admin lockout must be rejected")
			}

			var stored modelUser.User
			if err := env.db.First(&stored, current.Id).Error; err != nil {
				t.Fatal(err)
			}
			if stored.Enable == nil || !*stored.Enable || stored.IsAdmin == nil || !*stored.IsAdmin {
				t.Fatalf("current admin state changed: %#v", stored)
			}
		})
	}
}

func TestUpdateUserBasicInfoProtectsLastActiveAdminAndAllowsHandoff(t *testing.T) {
	env := newAdminUserSecurityEnv(t)
	last := createSecurityTestUser(t, env.db, 1, true, true)
	otherContext := context.WithValue(context.Background(), constant.CtxKeyUser, &modelUser.User{Id: 99})
	if err := NewUpdateUserBasicInfoLogic(otherContext, env.service).UpdateUserBasicInfo(
		basicSecurityUpdate(last.Id, true, false),
	); err == nil {
		t.Fatal("last active admin demotion must be rejected")
	}

	second := createSecurityTestUser(t, env.db, 2, true, true)
	currentContext := context.WithValue(context.Background(), constant.CtxKeyUser, last)
	if err := NewUpdateUserBasicInfoLogic(currentContext, env.service).UpdateUserBasicInfo(
		basicSecurityUpdate(second.Id, true, false),
	); err != nil {
		t.Fatalf("demoting a second admin should be allowed: %v", err)
	}
	var stored modelUser.User
	if err := env.db.First(&stored, second.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.IsAdmin == nil || *stored.IsAdmin {
		t.Fatalf("second admin was not demoted: %#v", stored)
	}
}

func TestDeleteUserAuthMethodKeepsOneSignInMethodAndRevokesSessions(t *testing.T) {
	env := newAdminUserSecurityEnv(t)
	createSecurityTestUser(t, env.db, 10, true, false)
	if err := env.db.Create(&modelUser.AuthMethods{
		UserId: 10, AuthType: "email", AuthIdentifier: "user@example.test", Verified: true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	logic := NewDeleteUserAuthMethodLogic(context.Background(), env.service)
	if err := logic.DeleteUserAuthMethod(&types.DeleteUserAuthMethodRequest{UserId: 10, AuthType: "email"}); err == nil {
		t.Fatal("last sign-in method removal must be rejected")
	}

	if err := env.db.Create(&modelUser.AuthMethods{
		UserId: 10, AuthType: "github", AuthIdentifier: "github-10", Verified: true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	const sessionID = "admin-auth-change-session"
	if err := env.redis.SAdd(context.Background(), config.UserSessionCacheKey(10), sessionID).Err(); err != nil {
		t.Fatal(err)
	}
	if err := env.redis.Set(context.Background(), config.SessionCacheKey(sessionID), config.SessionUserValue(10), time.Hour).Err(); err != nil {
		t.Fatal(err)
	}
	if err := logic.DeleteUserAuthMethod(&types.DeleteUserAuthMethodRequest{UserId: 10, AuthType: "github"}); err != nil {
		t.Fatalf("removing a secondary sign-in method failed: %v", err)
	}

	var count int64
	if err := env.db.Model(&modelUser.AuthMethods{}).Where("user_id = ?", 10).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("remaining auth method count = %d, want 1", count)
	}
	if got := env.redis.Exists(context.Background(), config.SessionCacheKey(sessionID), config.UserSessionCacheKey(10)).Val(); got != 0 {
		t.Fatalf("authentication session keys remain after auth change: %d", got)
	}
}

func TestAdminAuthMethodMutationsRejectCrossUserIdentifierConflicts(t *testing.T) {
	env := newAdminUserSecurityEnv(t)
	createSecurityTestUser(t, env.db, 11, true, false)
	createSecurityTestUser(t, env.db, 12, true, false)
	methods := []modelUser.AuthMethods{
		{UserId: 11, AuthType: "email", AuthIdentifier: "owner@example.test", Verified: true},
		{UserId: 11, AuthType: "github", AuthIdentifier: "github-owner", Verified: true},
		{UserId: 12, AuthType: "email", AuthIdentifier: "other@example.test", Verified: true},
	}
	if err := env.db.Create(&methods).Error; err != nil {
		t.Fatal(err)
	}

	if err := NewUpdateUserAuthMethodLogic(context.Background(), env.service).UpdateUserAuthMethod(
		&types.UpdateUserAuthMethodRequest{UserId: 12, AuthType: "email", AuthIdentifier: "OWNER@EXAMPLE.TEST"},
	); err == nil {
		t.Fatal("cross-user email reassignment must be rejected")
	}
	if err := NewCreateUserAuthMethodLogic(context.Background(), env.service).CreateUserAuthMethod(
		&types.CreateUserAuthMethodRequest{UserId: 12, AuthType: "github", AuthIdentifier: "github-owner"},
	); err == nil {
		t.Fatal("cross-user oauth identifier reassignment must be rejected")
	}

	stored, err := env.service.UserModel.FindUserAuthMethodByPlatform(context.Background(), 12, "email")
	if err != nil {
		t.Fatal(err)
	}
	if stored.AuthIdentifier != "other@example.test" {
		t.Fatalf("conflicting update changed identifier to %q", stored.AuthIdentifier)
	}
}
