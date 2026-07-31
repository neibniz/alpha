package oauth

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/authaudit"
	"github.com/neibniz/alpha/server/internal/authconfig"
	"github.com/neibniz/alpha/server/internal/authsession"
	"github.com/neibniz/alpha/server/internal/model/auth"
	modelLog "github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/oauthcallback"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/subscriptionstate"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	githuboauth "github.com/neibniz/alpha/server/pkg/oauth/github"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/uuidx"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	OAuthGitHub = "github"
	authEmail   = "email"
)

type OAuthLoginGetTokenLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewOAuthLoginGetTokenLogic OAuth login get token
func NewOAuthLoginGetTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OAuthLoginGetTokenLogic {
	return &OAuthLoginGetTokenLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OAuthLoginGetTokenLogic) OAuthLoginGetToken(req *types.OAuthLoginGetTokenRequest, ip, userAgent string) (resp *types.LoginResponse, err error) {
	if req == nil || req.Method != OAuthGitHub {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "only github oauth is supported")
	}
	requestID := uuidx.NewUUID().String()
	loginStatus := false
	var userInfo *user.User

	l.Infow("oauth login request started",
		logger.Field("request_id", requestID),
		logger.Field("method", req.Method),
		logger.Field("ip", logIP(ip)),
		logger.Field("user_agent", logUserAgent(userAgent)),
	)

	defer func() {
		l.recordLoginStatus(loginStatus, userInfo, ip, userAgent, requestID, req.Method)
	}()

	userInfo, err = l.handleOAuthProvider(req, requestID, ip, userAgent)
	if err != nil {
		return nil, err
	}

	token, err := l.generateToken(userInfo, req.Method, requestID)
	if err != nil {
		return nil, err
	}

	loginStatus = true
	return &types.LoginResponse{Token: token}, nil
}

func (l *OAuthLoginGetTokenLogic) github(req *types.OAuthLoginGetTokenRequest, requestID, ip, userAgent string) (*user.User, error) {
	startTime := time.Now()
	callback, err := oauthcallback.NewStateCode(req.Callback.Code, req.Callback.State)
	if err != nil {
		l.Errorw("failed to parse github callback data",
			logger.Field("request_id", requestID),
			logger.Field("provider", OAuthGitHub),
			logger.Field("error", err.Error()),
		)
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid github callback")
	}

	redirect, err := l.consumeGitHubLoginState(callback.State, requestID)
	if err != nil {
		return nil, err
	}
	cfg, err := l.getGitHubConfig(requestID)
	if err != nil {
		return nil, err
	}
	redirect, err = oauthcallback.ValidateRedirect(
		redirect,
		cfg.RedirectURL,
		oauthcallback.GitHubLoginPath,
		oauthcallback.GitHubLoginBridgePath,
	)
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid github redirect")
	}

	client := githuboauth.New(githuboauth.Config{
		ClientID:     cfg.ClientId,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  redirect,
	})
	token, err := client.Exchange(l.ctx, callback.Code)
	if err != nil {
		l.Errorw("failed to exchange github authorization code",
			logger.Field("request_id", requestID),
			logger.Field("provider", OAuthGitHub),
			logger.Field("error", err.Error()),
		)
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "exchange github token failed")
	}
	githubUser, err := client.GetUserInfo(l.ctx, token.AccessToken)
	if err != nil {
		l.Errorw("failed to get github user info",
			logger.Field("request_id", requestID),
			logger.Field("provider", OAuthGitHub),
			logger.Field("error", err.Error()),
		)
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "get github user info failed")
	}
	openID := strconv.FormatInt(githubUser.ID, 10)
	l.Infow("github oauth processing completed",
		logger.Field("request_id", requestID),
		logger.Field("provider", OAuthGitHub),
		logger.Field("openid", logIdentifier(openID)),
		logger.Field("email", logIdentifier(githubUser.Email)),
		logger.Field("duration_ms", time.Since(startTime).Milliseconds()),
	)
	return l.findOrRegisterUser(OAuthGitHub, openID, githubUser.Email, githubUser.AvatarURL, req.Invite, requestID, ip, userAgent)
}

func (l *OAuthLoginGetTokenLogic) register(email, avatar, method, openid, invite, requestID, ip, userAgent string) (*user.User, error) {
	startTime := time.Now()
	runtime := l.svcCtx.ConfigSnapshot()
	registerConfig := runtime.Register
	inviteConfig := runtime.Invite
	l.Infow("user registration started",
		logger.Field("request_id", requestID),
		logger.Field("auth_method", method),
		logger.Field("email", logIdentifier(email)),
		logger.Field("openid", logIdentifier(openid)),
	)

	if registerConfig.StopRegister {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.StopRegister), "stop register")
	}
	var referer *user.User
	if invite == "" {
		if inviteConfig.ForcedInvite {
			l.Errorw("registration blocked due to forced invite policy",
				logger.Field("request_id", requestID),
				logger.Field("auth_method", method),
			)
			return nil, errors.Wrap(xerr.NewErrCode(xerr.InviteCodeError), "invite code is required")
		}
	} else {
		var err error
		referer, err = l.svcCtx.UserModel.FindOneByReferCode(l.ctx, invite)
		if err != nil {
			l.Errorw("invalid oauth registration invite",
				logger.Field("request_id", requestID),
				logger.Field("auth_method", method),
				logger.Field("error", err.Error()),
			)
			return nil, errors.Wrap(xerr.NewErrCode(xerr.InviteCodeError), "invite code is invalid")
		}
	}

	var userInfo *user.User
	err := l.svcCtx.UserModel.Transaction(l.ctx, func(db *gorm.DB) error {
		if email != "" {
			l.Debugw("checking if email already exists",
				logger.Field("request_id", requestID),
				logger.Field("email", logIdentifier(email)),
			)
			if err := l.checkEmailExists(db, email, requestID); err != nil {
				return err
			}
		}

		l.Debugw("creating new user record",
			logger.Field("request_id", requestID),
		)

		userInfo = &user.User{Avatar: avatar, OnlyFirstPurchase: &inviteConfig.OnlyFirstPurchase}
		if referer != nil {
			userInfo.RefererId = referer.Id
		}
		if err := l.svcCtx.UserModel.Insert(l.ctx, userInfo, db); err != nil {
			l.Errorw("failed to create user record",
				logger.Field("request_id", requestID),
				logger.Field("error", err.Error()),
			)
			return errors.Wrap(xerr.NewErrCode(xerr.DatabaseInsertError), "create user info failed")
		}

		userInfo.ReferCode = uuidx.UserInviteCode(userInfo.Id)
		l.Debugw("updating user refer code",
			logger.Field("request_id", requestID),
			logger.Field("user_id", userInfo.Id),
		)

		if err := l.svcCtx.UserModel.UpdateUserReferCode(l.ctx, userInfo.Id, userInfo.ReferCode, db); err != nil {
			l.Errorw("failed to update refer code",
				logger.Field("request_id", requestID),
				logger.Field("user_id", userInfo.Id),
				logger.Field("error", err.Error()),
			)
			return errors.Wrap(xerr.NewErrCode(xerr.DatabaseUpdateError), "update refer code failed")
		}

		if err := l.createAuthMethod(db, userInfo.Id, method, openid, requestID); err != nil {
			return err
		}

		if email != "" {
			if err := l.createAuthMethod(db, userInfo.Id, authEmail, email, requestID); err != nil {
				return err
			}
		}

		if registerConfig.EnableTrial {
			l.Debugw("activating trial subscription",
				logger.Field("request_id", requestID),
				logger.Field("user_id", userInfo.Id),
			)
			if err := l.activeTrial(userInfo.Id, requestID, db); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		l.Errorw("user registration failed",
			logger.Field("request_id", requestID),
			logger.Field("auth_method", method),
			logger.Field("error", err.Error()),
			logger.Field("duration_ms", time.Since(startTime).Milliseconds()),
		)
		return userInfo, err
	}

	l.Infow("user registration completed successfully",
		logger.Field("request_id", requestID),
		logger.Field("user_id", userInfo.Id),
		logger.Field("auth_method", method),
		logger.Field("email", logIdentifier(email)),
		logger.Field("duration_ms", time.Since(startTime).Milliseconds()),
	)

	if err := authaudit.RecordRegister(l.ctx, l.svcCtx.LogModel, authaudit.RegisterEvent{
		UserID:     userInfo.Id,
		Method:     method,
		Identifier: openid,
		IP:         ip,
		UserAgent:  userAgent,
	}); err != nil {
		l.Errorw("failed to insert register log",
			logger.Field("request_id", requestID),
			logger.Field("user_id", userInfo.Id),
			logger.Field("error", err.Error()),
		)
	}

	return userInfo, nil
}

func (l *OAuthLoginGetTokenLogic) checkEmailExists(_ *gorm.DB, email, requestID string) error {
	methodInfo, err := l.svcCtx.UserModel.FindUserAuthMethodByOpenID(l.ctx, "email", email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		l.Errorw("failed to check email existence",
			logger.Field("request_id", requestID),
			logger.Field("email", logIdentifier(email)),
			logger.Field("error", err.Error()),
		)
		return errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "check email exists failed")
	}
	if methodInfo != nil && methodInfo.UserId != 0 {
		l.Errorw("email already exists for another user",
			logger.Field("request_id", requestID),
			logger.Field("email", logIdentifier(email)),
			logger.Field("existing_user_id", methodInfo.UserId),
		)
		return errors.Wrap(xerr.NewErrCode(xerr.UserExist), "email is already registered")
	}
	l.Debugw("email availability confirmed",
		logger.Field("request_id", requestID),
		logger.Field("email", logIdentifier(email)),
	)
	return nil
}

func (l *OAuthLoginGetTokenLogic) createAuthMethod(db *gorm.DB, userID int64, authType, identifier, requestID string) error {
	l.Debugw("creating auth method",
		logger.Field("request_id", requestID),
		logger.Field("user_id", userID),
		logger.Field("auth_type", authType),
		logger.Field("identifier", logIdentifier(identifier)),
	)

	authMethod := &user.AuthMethods{
		UserId:         userID,
		AuthType:       authType,
		AuthIdentifier: identifier,
		Verified:       true,
	}
	if err := l.svcCtx.UserModel.InsertUserAuthMethods(l.ctx, authMethod, db); err != nil {
		l.Errorw("failed to create auth method",
			logger.Field("request_id", requestID),
			logger.Field("user_id", userID),
			logger.Field("auth_type", authType),
			logger.Field("identifier", logIdentifier(identifier)),
			logger.Field("error", err.Error()),
		)
		return errors.Wrap(xerr.NewErrCode(xerr.DatabaseInsertError), "create auth method failed")
	}

	l.Debugw("auth method created successfully",
		logger.Field("request_id", requestID),
		logger.Field("user_id", userID),
		logger.Field("auth_type", authType),
		logger.Field("auth_method_id", authMethod.Id),
	)
	return nil
}

func (l *OAuthLoginGetTokenLogic) recordLoginStatus(loginStatus bool, userInfo *user.User, ip, userAgent, requestID, authType string) {
	if userInfo != nil && userInfo.Id != 0 {
		if err := authaudit.RecordLogin(l.ctx, l.svcCtx.LogModel, authaudit.LoginEvent{
			UserID:    userInfo.Id,
			Method:    authType,
			IP:        ip,
			UserAgent: userAgent,
			Success:   loginStatus,
		}); err != nil {
			l.Errorw("failed to insert login log",
				logger.Field("request_id", requestID),
				logger.Field("user_id", userInfo.Id),
				logger.Field("error", err.Error()),
			)
		}
	}
}

func (l *OAuthLoginGetTokenLogic) handleOAuthProvider(req *types.OAuthLoginGetTokenRequest, requestID, ip, userAgent string) (*user.User, error) {
	l.Debugw("handling oauth provider",
		logger.Field("request_id", requestID),
		logger.Field("provider", req.Method),
	)
	if req.Method != OAuthGitHub {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "only github oauth is supported")
	}
	return l.github(req, requestID, ip, userAgent)
}

func (l *OAuthLoginGetTokenLogic) generateToken(userInfo *user.User, loginType, requestID string) (string, error) {
	startTime := time.Now()

	l.Debugw("issuing authentication session",
		logger.Field("request_id", requestID),
		logger.Field("user_id", userInfo.Id),
	)

	token, err := authsession.Issue(l.ctx, l.svcCtx.Redis, l.svcCtx.Config.JwtAuth, userInfo.Id, loginType)
	if err != nil {
		l.Errorw("failed to issue authentication session",
			logger.Field("request_id", requestID),
			logger.Field("user_id", userInfo.Id),
			logger.Field("error", err.Error()),
		)
		return "", errors.Wrap(xerr.NewErrCode(xerr.ERROR), "create authentication session")
	}

	l.Infow("authentication session issued successfully",
		logger.Field("request_id", requestID),
		logger.Field("user_id", userInfo.Id),
		logger.Field("duration_ms", time.Since(startTime).Milliseconds()),
	)

	return token, nil
}

func (l *OAuthLoginGetTokenLogic) consumeGitHubLoginState(state, requestID string) (string, error) {
	state = strings.TrimSpace(state)
	l.Debugw("validating github oauth state", logger.Field("request_id", requestID))

	if state == "" {
		l.Errorw("empty github oauth state", logger.Field("request_id", requestID))
		return "", errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "empty oauth state")
	}

	stateKey := fmt.Sprintf("%s:%s", oauthcallback.GitHubLoginStateScope, state)
	redirect, err := l.svcCtx.Redis.GetDel(l.ctx, stateKey).Result()
	if err != nil {
		l.Errorw("failed to validate github oauth state",
			logger.Field("request_id", requestID),
			logger.Field("error", err.Error()),
		)
		return "", errors.Wrap(xerr.NewErrCode(xerr.ERROR), "oauth state validation failed")
	}

	l.Debugw("github oauth state validated", logger.Field("request_id", requestID))
	return redirect, nil
}

func (l *OAuthLoginGetTokenLogic) getGitHubConfig(requestID string) (*auth.GithubAuthConfig, error) {
	authMethod, err := l.svcCtx.AuthModel.FindOneByMethod(l.ctx, OAuthGitHub)
	if err != nil {
		l.Errorw("failed to find github auth method",
			logger.Field("request_id", requestID),
			logger.Field("provider", OAuthGitHub),
			logger.Field("error", err.Error()),
		)
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "find github auth method failed")
	}
	if authMethod.Enabled == nil || !*authMethod.Enabled {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "github oauth is disabled")
	}
	cfg, err := authconfig.LoadGitHub(authMethod.Config, l.svcCtx.Privacy)
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "load github config failed")
	}
	if err := authconfig.ValidateGitHub(cfg); err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "invalid github config")
	}
	return &cfg, nil
}

func (l *OAuthLoginGetTokenLogic) findOrRegisterUser(authType, openID, email, avatar, invite, requestID, ip, userAgent string) (*user.User, error) {
	l.Debugw("finding or registering user",
		logger.Field("request_id", requestID),
		logger.Field("auth_type", authType),
		logger.Field("openid", logIdentifier(openID)),
		logger.Field("email", logIdentifier(email)),
	)

	userAuthMethod, err := l.svcCtx.UserModel.FindUserAuthMethodByOpenID(l.ctx, authType, openID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Infow("user not found, starting registration",
				logger.Field("request_id", requestID),
				logger.Field("auth_type", authType),
				logger.Field("openid", logIdentifier(openID)),
				logger.Field("email", logIdentifier(email)),
			)
			return l.register(email, avatar, authType, openID, invite, requestID, ip, userAgent)
		}
		l.Errorw("failed to find user auth method by openid",
			logger.Field("request_id", requestID),
			logger.Field("auth_type", authType),
			logger.Field("openid", logIdentifier(openID)),
			logger.Field("error", err.Error()),
		)
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "find user auth method by openid failed")
	}

	l.Debugw("found existing user auth method",
		logger.Field("request_id", requestID),
		logger.Field("auth_type", authType),
		logger.Field("user_id", userAuthMethod.UserId),
	)

	userInfo, err := l.svcCtx.UserModel.FindOneForLogin(l.ctx, userAuthMethod.UserId)
	if err != nil {
		l.Errorw("failed to find user by id",
			logger.Field("request_id", requestID),
			logger.Field("user_id", userAuthMethod.UserId),
			logger.Field("error", err.Error()),
		)
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "find user info failed")
	}
	if userInfo.DeletedAt.Valid || userInfo.Enable == nil || !*userInfo.Enable {
		l.Errorw("user is disabled",
			logger.Field("request_id", requestID),
			logger.Field("user_id", userAuthMethod.UserId),
		)
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "invalid access")
	}

	l.Infow("existing user found successfully",
		logger.Field("request_id", requestID),
		logger.Field("user_id", userInfo.Id),
		logger.Field("auth_type", authType),
	)

	return userInfo, nil
}

func (l *OAuthLoginGetTokenLogic) activeTrial(uid int64, requestID string, tx ...*gorm.DB) error {
	register := l.svcCtx.ConfigSnapshot().Register
	l.Debugw("fetching trial subscription template",
		logger.Field("request_id", requestID),
		logger.Field("user_id", uid),
		logger.Field("trial_subscribe_id", register.TrialSubscribe),
	)

	sub, err := l.svcCtx.SubscribeModel.FindOneForOrderSnapshot(l.ctx, register.TrialSubscribe)
	if err != nil {
		l.Errorw("failed to find trial subscription template",
			logger.Field("request_id", requestID),
			logger.Field("user_id", uid),
			logger.Field("trial_subscribe_id", register.TrialSubscribe),
			logger.Field("error", err.Error()),
		)
		return err
	}

	startTime := time.Now()
	expireTime := tool.AddTime(register.TrialTimeUnit, register.TrialTime, startTime)
	subscribeToken, err := uuidx.NewSubscribeToken()
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "generate trial subscription token")
	}
	subscribeUUID := uuidx.NewUUID().String()
	productSnapshot := ordersnapshot.FromSubscribe(sub)
	resetCycle := subscriptionstate.NormalizeResetCycle(productSnapshot.ResetCycle)

	l.Debugw("creating trial subscription",
		logger.Field("request_id", requestID),
		logger.Field("user_id", uid),
		logger.Field("subscribe_id", sub.Id),
		logger.Field("start_time", startTime),
		logger.Field("expire_time", expireTime),
		logger.Field("traffic", sub.Traffic),
		logger.Field("token_prefix", modelLog.TokenPrefix(subscribeToken)),
	)

	userSub := &user.Subscribe{
		Id:              0,
		UserId:          uid,
		OrderId:         0,
		SubscribeId:     sub.Id,
		ProductName:     productSnapshot.Name,
		ProductSnapshot: ordersnapshot.Marshal(productSnapshot),
		StartTime:       startTime,
		ExpireTime:      expireTime,
		Traffic:         sub.Traffic,
		Download:        0,
		Upload:          0,
		ResetCycle:      resetCycle,
		NextTrafficResetAt: subscriptionstate.NextTrafficResetAt(
			resetCycle,
			startTime,
			startTime,
			expireTime,
			subscriptionstate.TrafficResetLocation(),
		),
		Token:  subscribeToken,
		UUID:   subscribeUUID,
		Status: 1,
	}

	if err := l.svcCtx.UserModel.InsertSubscribe(l.ctx, userSub, tx...); err != nil {
		l.Errorw("failed to insert trial subscription",
			logger.Field("request_id", requestID),
			logger.Field("user_id", uid),
			logger.Field("error", err.Error()),
		)
		return err
	}

	l.Infow("trial subscription activated successfully",
		logger.Field("request_id", requestID),
		logger.Field("user_id", uid),
		logger.Field("subscribe_id", sub.Id),
		logger.Field("expire_time", expireTime),
		logger.Field("traffic", sub.Traffic),
	)
	return nil
}
