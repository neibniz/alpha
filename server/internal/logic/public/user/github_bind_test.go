package user

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	modelauth "github.com/neibniz/alpha/server/internal/model/auth"
	modeluser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/oauthcallback"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGitHubBindCreatesSeparateState(t *testing.T) {
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
	enabled := true
	config := modelauth.GithubAuthConfig{
		ClientId:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "https://panel.example/",
	}
	model := modelauth.NewModel(db, rds)
	if err := model.Insert(context.Background(), &modelauth.Auth{Method: "github", Config: config.Marshal(), Enabled: &enabled}); err != nil {
		t.Fatal(err)
	}
	account := &modeluser.User{Id: 7}
	ctx := context.WithValue(context.Background(), constant.CtxKeyUser, account)
	logic := NewBindOAuthLogic(ctx, &svc.ServiceContext{
		Redis:     rds,
		AuthModel: model,
		Privacy:   privacy.DisabledCodec(),
	})
	resp, err := logic.BindOAuth(&types.BindOAuthRequest{Method: "github", Redirect: "https://panel.example/bind/github/index.html"})
	if err != nil {
		t.Fatal(err)
	}
	redirect, err := url.Parse(resp.Redirect)
	if err != nil {
		t.Fatal(err)
	}
	state := redirect.Query().Get("state")
	if state == "" {
		t.Fatalf("github bind redirect has no state: %s", redirect)
	}
	stored, err := rds.Get(context.Background(), oauthcallback.GitHubBindStateScope+":"+state).Result()
	if err != nil {
		t.Fatal(err)
	}
	bindState, err := oauthcallback.DecodeBindState(stored)
	if err != nil {
		t.Fatal(err)
	}
	if bindState.UserID != account.Id || bindState.Redirect != "https://panel.example/bind/github/index.html" {
		t.Fatalf("stored bind state = %#v", bindState)
	}
	if rds.Exists(context.Background(), oauthcallback.GitHubLoginStateScope+":"+state).Val() != 0 {
		t.Fatal("bind state must not be accepted by the login callback")
	}
}

func TestOAuthBindingEndpointsRejectNonGitHubProviders(t *testing.T) {
	if _, err := NewBindOAuthLogic(context.Background(), &svc.ServiceContext{}).BindOAuth(
		&types.BindOAuthRequest{Method: "google"},
	); err == nil {
		t.Fatal("google oauth binding should be rejected")
	}

	account := &modeluser.User{Id: 7}
	ctx := context.WithValue(context.Background(), constant.CtxKeyUser, account)
	callback := NewBindOAuthCallbackLogic(ctx, &svc.ServiceContext{})
	if err := callback.BindOAuthCallback(&types.BindOAuthCallbackRequest{Method: "apple"}); err == nil {
		t.Fatal("apple oauth binding callback should be rejected")
	}
}

func TestGitHubBindStateIsBoundToInitiatingUser(t *testing.T) {
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	stateValue, err := oauthcallback.EncodeBindState(8, "https://panel.example/bind/github/")
	if err != nil {
		t.Fatal(err)
	}
	if err := rds.Set(context.Background(), oauthcallback.GitHubBindStateScope+":state", stateValue, 0).Err(); err != nil {
		t.Fatal(err)
	}
	logic := NewBindOAuthCallbackLogic(context.Background(), &svc.ServiceContext{Redis: rds})
	if _, err := logic.consumeOAuthState(oauthcallback.GitHubBindStateScope, "state", 7); err == nil {
		t.Fatal("state issued to another user should be rejected")
	}
	if rds.Exists(context.Background(), oauthcallback.GitHubBindStateScope+":state").Val() != 0 {
		t.Fatal("rejected state should still be consumed")
	}
}

func TestGitHubBindCallbackStoresStableUserID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "bind.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelauth.Auth{}, &modeluser.User{}, &modeluser.AuthMethods{}); err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	enabled := true
	authModel := modelauth.NewModel(db, rds)
	config := modelauth.GithubAuthConfig{ClientId: "client-id", ClientSecret: "client-secret", RedirectURL: "https://panel.example/"}
	if err := authModel.Insert(context.Background(), &modelauth.Auth{Method: "github", Config: config.Marshal(), Enabled: &enabled}); err != nil {
		t.Fatal(err)
	}
	userModel := modeluser.NewModel(db, rds, privacy.DisabledCodec())
	account := &modeluser.User{Enable: &enabled}
	if err := userModel.Insert(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	if err := userModel.InsertUserAuthMethods(context.Background(), &modeluser.AuthMethods{
		UserId: account.Id, AuthType: "email", AuthIdentifier: "user@example.test", Verified: true,
	}); err != nil {
		t.Fatal(err)
	}
	state := "bind-callback-state"
	stateValue, err := oauthcallback.EncodeBindState(account.Id, "https://panel.example/bind/github/")
	if err != nil {
		t.Fatal(err)
	}
	if err := rds.Set(context.Background(), oauthcallback.GitHubBindStateScope+":"+state, stateValue, 0).Err(); err != nil {
		t.Fatal(err)
	}
	httpClient := &http.Client{Transport: bindRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := ""
		switch {
		case req.URL.Host == "github.com" && req.URL.Path == "/login/oauth/access_token":
			body = `{"access_token":"github-token","token_type":"bearer"}`
		case req.URL.Host == "api.github.com" && req.URL.Path == "/user":
			body = `{"id":42,"login":"octocat"}`
		case req.URL.Host == "api.github.com" && req.URL.Path == "/user/emails":
			body = `[{"email":"octocat@example.test","verified":true,"primary":true}]`
		default:
			t.Fatalf("unexpected GitHub request: %s %s", req.Method, req.URL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	ctx = context.WithValue(ctx, constant.CtxKeyUser, account)
	logic := NewBindOAuthCallbackLogic(ctx, &svc.ServiceContext{
		Redis: rds, AuthModel: authModel, UserModel: userModel, Privacy: privacy.DisabledCodec(),
	})
	if err := logic.github(&types.BindOAuthCallbackRequest{
		Method: "github",
		Callback: types.OAuthCallbackPayload{
			Code: "authorization-code", State: state,
		},
	}); err != nil {
		t.Fatal(err)
	}
	binding, err := userModel.FindUserAuthMethodByOpenID(context.Background(), "github", "42")
	if err != nil {
		t.Fatal(err)
	}
	if binding.UserId != account.Id || !binding.Verified {
		t.Fatalf("unexpected github binding: %+v", binding)
	}
}

type bindRoundTripFunc func(*http.Request) (*http.Response, error)

func (f bindRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
