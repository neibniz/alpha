package authMethod

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/neibniz/alpha/server/internal/authconfig"
	modelauth "github.com/neibniz/alpha/server/internal/model/auth"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUpdateGitHubConfigEncryptsAndPreservesSecret(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelauth.Auth{}); err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	codec, err := privacy.NewCodec(privacy.Keyring{
		ActiveKeyID:   "test",
		EncryptionKey: map[string]string{"test": base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 32)))},
		IndexKey:      base64.StdEncoding.EncodeToString([]byte(strings.Repeat("b", 32))),
	})
	if err != nil {
		t.Fatal(err)
	}
	enabled := false
	model := modelauth.NewModel(db, rds)
	if err := model.Insert(context.Background(), &modelauth.Auth{
		Method:  "github",
		Config:  (&modelauth.GithubAuthConfig{}).Marshal(),
		Enabled: &enabled,
	}); err != nil {
		t.Fatal(err)
	}
	logic := NewUpdateAuthMethodConfigLogic(context.Background(), &svc.ServiceContext{
		AuthModel: model,
		Privacy:   codec,
	})

	enabled = true
	response, err := logic.UpdateAuthMethodConfig(&types.UpdateAuthMethodConfigRequest{
		Method:  "github",
		Enabled: &enabled,
		Config:  json.RawMessage(`{"client_id":"client-1","client_secret":"secret-1","redirect_url":"https://panel.example/"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(response.Config), "secret-1") {
		t.Fatalf("response leaked secret: %s", response.Config)
	}

	stored, err := model.FindOneByMethodForAdmin(context.Background(), "github", true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored.Config, "secret-1") || !strings.Contains(stored.Config, "enc:v1:test:") {
		t.Fatalf("stored config is not encrypted: %s", stored.Config)
	}

	response, err = logic.UpdateAuthMethodConfig(&types.UpdateAuthMethodConfigRequest{
		Method: "github",
		Config: json.RawMessage(`{"client_id":"client-2","client_secret":"","redirect_url":"https://panel.example/"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	stored, err = model.FindOneByMethodForAdmin(context.Background(), "github", true)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := authconfig.LoadGitHub(stored.Config, codec)
	if err != nil {
		t.Fatal(err)
	}
	if opened.ClientId != "client-2" || opened.ClientSecret != "secret-1" {
		t.Fatalf("secret was not preserved: %+v", opened)
	}
	if strings.Contains(string(response.Config), "secret-1") {
		t.Fatalf("response leaked preserved secret: %s", response.Config)
	}
}

func TestUpdateGitHubConfigRejectsIncompleteEnable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelauth.Auth{}); err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	enabled := false
	model := modelauth.NewModel(db, rds)
	if err := model.Insert(context.Background(), &modelauth.Auth{Method: "github", Config: `{}`, Enabled: &enabled}); err != nil {
		t.Fatal(err)
	}
	logic := NewUpdateAuthMethodConfigLogic(context.Background(), &svc.ServiceContext{AuthModel: model, Privacy: privacy.DisabledCodec()})
	enabled = true
	if _, err := logic.UpdateAuthMethodConfig(&types.UpdateAuthMethodConfigRequest{Method: "github", Enabled: &enabled}); err == nil {
		t.Fatal("incomplete github config should not be enabled")
	}
}
