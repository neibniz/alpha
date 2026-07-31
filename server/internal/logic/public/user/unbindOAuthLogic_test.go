package user

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/alicebob/miniredis/v2"
	modeluser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUnbindOAuthKeepsLastSignInMethod(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "user.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modeluser.AuthMethods{}); err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	model := modeluser.NewModel(db, rds, privacy.DisabledCodec())
	if err := db.Create(&modeluser.AuthMethods{UserId: 7, AuthType: "github", AuthIdentifier: "42", Verified: true}).Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), constant.CtxKeyUser, &modeluser.User{Id: 7})
	logic := NewUnbindOAuthLogic(ctx, &svc.ServiceContext{UserModel: model})
	request := &types.UnbindOAuthRequest{Method: "github"}

	if err := logic.UnbindOAuth(request); err == nil {
		t.Fatal("last sign-in method should not be removed")
	}
	if err := db.Create(&modeluser.AuthMethods{UserId: 7, AuthType: "email", AuthIdentifier: "user@example.test", Verified: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := logic.UnbindOAuth(request); err != nil {
		t.Fatalf("github should be removable when another method exists: %v", err)
	}
	var count int64
	if err := db.Model(&modeluser.AuthMethods{}).Where("user_id = ? AND auth_type = ?", 7, "github").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("github binding count = %d", count)
	}
}
