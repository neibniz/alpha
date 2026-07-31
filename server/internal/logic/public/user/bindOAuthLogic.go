package user

import (
	"context"
	"fmt"
	"time"

	"github.com/neibniz/alpha/server/internal/authconfig"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/oauthcallback"
	githuboauth "github.com/neibniz/alpha/server/pkg/oauth/github"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type BindOAuthLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Bind OAuth
func NewBindOAuthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BindOAuthLogic {
	return &BindOAuthLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BindOAuthLogic) BindOAuth(req *types.BindOAuthRequest) (resp *types.BindOAuthResponse, err error) {
	if req == nil || req.Method != "github" {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "only github oauth is supported")
	}
	uri, err := l.github(req)
	if err != nil {
		l.Errorw("github oauth binding failed", logger.Field("error", err.Error()))
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "github oauth binding failed")
	}
	return &types.BindOAuthResponse{
		Redirect: uri,
	}, nil
}

func (l *BindOAuthLogic) github(req *types.BindOAuthRequest) (string, error) {
	currentUser, ok := l.ctx.Value(constant.CtxKeyUser).(*modelUser.User)
	if !ok || currentUser == nil || currentUser.Id <= 0 {
		return "", errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "invalid access")
	}
	authMethod, err := l.svcCtx.AuthModel.FindOneByMethod(l.ctx, "github")
	if err != nil {
		return "", err
	}
	if authMethod.Enabled == nil || !*authMethod.Enabled {
		return "", errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "github oauth is disabled")
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
		oauthcallback.GitHubBindPath,
		oauthcallback.GitHubBindBridgePath,
	)
	if err != nil {
		return "", err
	}
	state, err := oauthcallback.NewState()
	if err != nil {
		return "", err
	}
	stateKey := fmt.Sprintf("%s:%s", oauthcallback.GitHubBindStateScope, state)
	stateValue, err := oauthcallback.EncodeBindState(currentUser.Id, redirect)
	if err != nil {
		return "", err
	}
	if err := l.svcCtx.Redis.Set(l.ctx, stateKey, stateValue, 5*time.Minute).Err(); err != nil {
		return "", err
	}
	client := githuboauth.New(githuboauth.Config{
		ClientID:     cfg.ClientId,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  redirect,
	})
	return client.AuthCodeURL(state), nil
}
