package user

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/neibniz/alpha/server/internal/authconfig"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/oauthcallback"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	githuboauth "github.com/neibniz/alpha/server/pkg/oauth/github"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type BindOAuthCallbackLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBindOAuthCallbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BindOAuthCallbackLogic {
	return &BindOAuthCallbackLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BindOAuthCallbackLogic) BindOAuthCallback(req *types.BindOAuthCallbackRequest) error {
	if _, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User); !ok {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "invalid access")
	}
	if req == nil || req.Method != "github" {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "only github oauth is supported")
	}
	if err := l.github(req); err != nil {
		l.Errorw("github oauth binding callback failed", logger.Field("error", err.Error()))
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "github oauth binding callback failed")
	}
	return nil
}

func (l *BindOAuthCallbackLogic) github(req *types.BindOAuthCallbackRequest) error {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "invalid access")
	}
	callback, err := oauthcallback.NewStateCode(req.Callback.Code, req.Callback.State)
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid github callback")
	}
	redirect, err := l.consumeOAuthState(oauthcallback.GitHubBindStateScope, callback.State, u.Id)
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "get github state code failed")
	}
	authMethod, err := l.svcCtx.AuthModel.FindOneByMethod(l.ctx, "github")
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "find github auth method failed")
	}
	if authMethod.Enabled == nil || !*authMethod.Enabled {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "github oauth is disabled")
	}
	cfg, err := authconfig.LoadGitHub(authMethod.Config, l.svcCtx.Privacy)
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "load github config failed")
	}
	if err := authconfig.ValidateGitHub(cfg); err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "invalid github config")
	}
	redirect, err = oauthcallback.ValidateRedirect(
		redirect,
		cfg.RedirectURL,
		oauthcallback.GitHubBindPath,
		oauthcallback.GitHubBindBridgePath,
	)
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid github redirect")
	}
	client := githuboauth.New(githuboauth.Config{
		ClientID:     cfg.ClientId,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  redirect,
	})
	token, err := client.Exchange(l.ctx, callback.Code)
	if err != nil {
		l.Errorw("exchange github token failed", logger.Field("error", err.Error()))
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "exchange github token failed")
	}
	githubUser, err := client.GetUserInfo(l.ctx, token.AccessToken)
	if err != nil {
		l.Errorw("get github user info failed", logger.Field("error", err.Error()))
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "get github user info failed")
	}
	openID := strconv.FormatInt(githubUser.ID, 10)

	if _, err := l.svcCtx.UserModel.FindUserAuthMethodByPlatform(l.ctx, u.Id, "github"); err == nil {
		return errors.Wrap(xerr.NewErrCode(xerr.UserExist), "github account is already bound")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "query current github binding failed")
	}
	existing, err := l.svcCtx.UserModel.FindUserAuthMethodByOpenID(l.ctx, "github", openID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "query github binding failed")
	}
	if err == nil && existing.Id > 0 {
		return errors.Wrap(xerr.NewErrCode(xerr.UserExist), "github account is already bound")
	}

	binding := &user.AuthMethods{
		UserId:         u.Id,
		AuthType:       "github",
		AuthIdentifier: openID,
		Verified:       true,
	}
	if err := l.svcCtx.UserModel.InsertUserAuthMethods(l.ctx, binding); err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.DatabaseInsertError), "insert github auth method failed")
	}
	return nil
}

func (l *BindOAuthCallbackLogic) consumeOAuthState(scope, state string, userID int64) (string, error) {
	state = strings.TrimSpace(state)
	if state == "" {
		return "", errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "empty oauth state")
	}
	raw, err := l.svcCtx.Redis.GetDel(l.ctx, fmt.Sprintf("%s:%s", scope, state)).Result()
	if err != nil {
		return "", err
	}
	bindState, err := oauthcallback.DecodeBindState(raw)
	if err != nil || bindState.UserID != userID {
		return "", errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "oauth state does not belong to current user")
	}
	return bindState.Redirect, nil
}
