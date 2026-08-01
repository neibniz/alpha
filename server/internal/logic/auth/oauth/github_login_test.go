package oauth

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/model/auth"
	modellog "github.com/neibniz/alpha/server/internal/model/log"
	modeluser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/oauthcallback"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGitHubOAuthLoginCreatesScopedState(t *testing.T) {
	logic, rds := newGitHubLoginLogic(t, true)
	resp, err := logic.OAuthLogin(&types.OAthLoginRequest{
		Method:   "github",
		Redirect: "https://panel.example/oauth/github/callback",
	})
	if err != nil {
		t.Fatal(err)
	}
	redirect, err := url.Parse(resp.Redirect)
	if err != nil {
		t.Fatal(err)
	}
	state := redirect.Query().Get("state")
	if redirect.Host != "github.com" || state == "" || redirect.Query().Get("client_id") != "client-id" {
		t.Fatalf("unexpected github redirect: %s", redirect)
	}
	stored, err := rds.Get(context.Background(), oauthcallback.GitHubLoginStateScope+":"+state).Result()
	if err != nil {
		t.Fatal(err)
	}
	if stored != "https://panel.example/oauth/github/callback" {
		t.Fatalf("stored redirect = %q", stored)
	}
}

func TestGitHubOAuthLoginRejectsDisabledAndUntrustedRedirect(t *testing.T) {
	disabled, _ := newGitHubLoginLogic(t, false)
	if _, err := disabled.OAuthLogin(&types.OAthLoginRequest{Method: "github", Redirect: "https://panel.example/oauth/github/callback"}); err == nil {
		t.Fatal("disabled github oauth should be rejected")
	}

	enabled, _ := newGitHubLoginLogic(t, true)
	if _, err := enabled.OAuthLogin(&types.OAthLoginRequest{Method: "github", Redirect: "https://evil.example/oauth/github/callback"}); err == nil {
		t.Fatal("untrusted github redirect should be rejected")
	}
}

func TestOAuthLoginEndpointsRejectNonGitHubProviders(t *testing.T) {
	service := &svc.ServiceContext{}
	login := NewOAuthLoginLogic(context.Background(), service)
	if _, err := login.OAuthLogin(&types.OAthLoginRequest{Method: "google"}); err == nil {
		t.Fatal("google oauth login should be rejected")
	}
	if _, err := login.OAuthLogin(nil); err == nil {
		t.Fatal("nil oauth login request should be rejected")
	}

	token := NewOAuthLoginGetTokenLogic(context.Background(), service)
	if _, err := token.OAuthLoginGetToken(&types.OAuthLoginGetTokenRequest{Method: "apple"}, "", ""); err == nil {
		t.Fatal("apple oauth token exchange should be rejected")
	}
}

func TestGitHubOAuthCallbackAuthenticatesExistingUser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "callback.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&auth.Auth{}, &modeluser.User{}, &modeluser.AuthMethods{}); err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	enabled := true
	authModel := auth.NewModel(db, rds)
	config := auth.GithubAuthConfig{ClientId: "client-id", ClientSecret: "client-secret", RedirectURL: "https://panel.example/"}
	if err := authModel.Insert(context.Background(), &auth.Auth{Method: "github", Config: config.Marshal(), Enabled: &enabled}); err != nil {
		t.Fatal(err)
	}
	userModel := modeluser.NewModel(db, rds, privacy.DisabledCodec())
	account := &modeluser.User{Enable: &enabled}
	if err := userModel.Insert(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	if err := userModel.InsertUserAuthMethods(context.Background(), &modeluser.AuthMethods{
		UserId: account.Id, AuthType: "github", AuthIdentifier: "42", Verified: true,
	}); err != nil {
		t.Fatal(err)
	}
	state := "callback-state"
	if err := rds.Set(context.Background(), oauthcallback.GitHubLoginStateScope+":"+state, "https://panel.example/oauth/github/callback", 0).Err(); err != nil {
		t.Fatal(err)
	}

	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := ""
		contentType := "application/json"
		switch {
		case req.URL.Host == "github.com" && req.URL.Path == "/login/oauth/access_token":
			body = `{"access_token":"github-token","token_type":"bearer","scope":"read:user,user:email"}`
		case req.URL.Host == "api.github.com" && req.URL.Path == "/user":
			body = `{"id":42,"login":"octocat","avatar_url":"https://avatars.example/42"}`
		case req.URL.Host == "api.github.com" && req.URL.Path == "/user/emails":
			body = `[{"email":"octocat@example.test","verified":true,"primary":true}]`
		default:
			t.Fatalf("unexpected GitHub request: %s %s", req.Method, req.URL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{contentType}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	logic := NewOAuthLoginGetTokenLogic(ctx, &svc.ServiceContext{
		Redis: rds, AuthModel: authModel, UserModel: userModel, Privacy: privacy.DisabledCodec(),
	})
	got, err := logic.github(&types.OAuthLoginGetTokenRequest{
		Method: "github",
		Callback: types.OAuthCallbackPayload{
			Code: "authorization-code", State: state,
		},
	}, "request-id", "192.0.2.1", "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != account.Id {
		t.Fatalf("authenticated user id = %d, want %d", got.Id, account.Id)
	}
	if rds.Exists(context.Background(), oauthcallback.GitHubLoginStateScope+":"+state).Val() != 0 {
		t.Fatal("oauth state was not consumed")
	}
}

func TestOAuthRegistrationHonorsInvite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "register.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modeluser.User{}, &modeluser.AuthMethods{}, &modellog.SystemLog{}); err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	userModel := modeluser.NewModel(db, rds, privacy.DisabledCodec())
	enabled := true
	referer := &modeluser.User{Enable: &enabled, ReferCode: "invite-code"}
	if err := userModel.Insert(context.Background(), referer); err != nil {
		t.Fatal(err)
	}
	logic := NewOAuthLoginGetTokenLogic(context.Background(), &svc.ServiceContext{
		Redis:     rds,
		UserModel: userModel,
		LogModel:  modellog.NewModel(db),
		Privacy:   privacy.DisabledCodec(),
		Config: config.Config{
			Invite: config.InviteConfig{ForcedInvite: true},
		},
	})
	if _, err := logic.register("blocked@example.test", "", "github", "41", "", "request-id", "192.0.2.1", "agent"); err == nil {
		t.Fatal("forced invite should reject an empty invite code")
	}
	registered, err := logic.register("new@example.test", "", "github", "42", "invite-code", "request-id", "192.0.2.1", "agent")
	if err != nil {
		t.Fatal(err)
	}
	if registered.RefererId != referer.Id {
		t.Fatalf("referer id = %d, want %d", registered.RefererId, referer.Id)
	}
	githubMethod, err := userModel.FindUserAuthMethodByOpenID(context.Background(), "github", "42")
	if err != nil {
		t.Fatal(err)
	}
	if githubMethod.UserId != registered.Id {
		t.Fatalf("github method user id = %d, want %d", githubMethod.UserId, registered.Id)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newGitHubLoginLogic(t *testing.T, enabled bool) (*OAuthLoginLogic, *redis.Client) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "oauth.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&auth.Auth{}); err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	config := auth.GithubAuthConfig{
		ClientId:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "https://panel.example/",
	}
	method := &auth.Auth{Method: "github", Config: config.Marshal(), Enabled: &enabled}
	model := auth.NewModel(db, rds)
	if err := model.Insert(context.Background(), method); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rds.Close() })
	service := &svc.ServiceContext{
		Redis:     rds,
		AuthModel: model,
		Privacy:   privacy.DisabledCodec(),
	}
	return NewOAuthLoginLogic(context.Background(), service), rds
}
