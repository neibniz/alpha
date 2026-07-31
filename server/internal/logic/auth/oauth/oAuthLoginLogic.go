package oauth

import (
	"context"
	"fmt"
	"time"

	"github.com/neibniz/alpha/server/internal/authconfig"
	"github.com/neibniz/alpha/server/internal/oauthcallback"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	githuboauth "github.com/neibniz/alpha/server/pkg/oauth/github"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type OAuthLoginLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// OAuth login
func NewOAuthLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OAuthLoginLogic {
	return &OAuthLoginLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OAuthLoginLogic) OAuthLogin(req *types.OAthLoginRequest) (resp *types.OAuthLoginResponse, err error) {
	if req == nil || req.Method != OAuthGitHub {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "only github oauth is supported")
	}
	uri, err := l.github(req)
	if err != nil {
		l.Errorw("github oauth login failed", logger.Field("error", err.Error()))
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "github oauth login failed")
	}
	return &types.OAuthLoginResponse{
		Redirect: uri,
	}, nil
}

func (l *OAuthLoginLogic) github(req *types.OAthLoginRequest) (string, error) {
	authMethod, err := l.svcCtx.AuthModel.FindOneByMethod(l.ctx, "github")
	if err != nil {
		return "", err
	}
	if authMethod.Enabled == nil || !*authMethod.Enabled {
		return "", errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "github oauth is disabled")
	}
	cfg, err := authconfig.LoadGitHub(authMethod.Config, l.svcCtx.Privacy)
	if err != nil {
		return "", err
	}
	if err := authconfig.ValidateGitHub(cfg); err != nil {
		return "", err
	}
	redirect, err := oauthcallback.ValidateRedirect(
		req.Redirect,
		cfg.RedirectURL,
		oauthcallback.GitHubLoginPath,
		oauthcallback.GitHubLoginBridgePath,
	)
	if err != nil {
		return "", err
	}
	state, err := oauthcallback.NewState()
	if err != nil {
		return "", err
	}
	stateKey := fmt.Sprintf("%s:%s", oauthcallback.GitHubLoginStateScope, state)
	if err := l.svcCtx.Redis.Set(l.ctx, stateKey, redirect, 5*time.Minute).Err(); err != nil {
		return "", err
	}
	client := githuboauth.New(githuboauth.Config{
		ClientID:     cfg.ClientId,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  redirect,
	})
	return client.AuthCodeURL(state), nil
}
