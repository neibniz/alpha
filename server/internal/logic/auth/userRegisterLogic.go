package auth

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/authsession"
	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/subscriptionstate"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/verificationcode"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/uuidx"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type UserRegisterLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUserRegisterLogic User register
func NewUserRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserRegisterLogic {
	return &UserRegisterLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserRegisterLogic) UserRegister(req *types.UserRegisterRequest) (resp *types.LoginResponse, err error) {

	runtime := l.svcCtx.ConfigSnapshot()
	c := runtime.Register
	email := runtime.Email
	invite := runtime.Invite
	var referer *user.User
	// Check if the registration is stopped
	if c.StopRegister {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.StopRegister), "stop register")
	}

	if req.Invite == "" {
		if invite.ForcedInvite {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.InviteCodeError), "invite code is required")
		}
	} else {
		// Check if the invite code is valid
		referer, err = l.svcCtx.UserModel.FindOneByReferCode(l.ctx, req.Invite)
		if err != nil {
			l.Errorw("FindOneByReferCode Error", logger.Field("error", err))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.InviteCodeError), "invite code is invalid")
		}
	}

	// if the email verification is enabled, the verification code is required
	if email.EnableVerify {
		cacheKey := config.EmailAuthCodeCacheKey(constant.Register.String(), l.svcCtx.Privacy.CacheToken("email", req.Email))
		verified, verifyErr := verificationcode.Consume(l.ctx, l.svcCtx.Redis, cacheKey, req.Code)
		if verifyErr != nil {
			l.Errorw("verification code storage error", logger.Field("error", verifyErr.Error()))
		}
		if verifyErr != nil || !verified {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.VerifyCodeError), "code error")
		}
	}
	// Check if the user exists
	u, err := l.svcCtx.UserModel.FindOneByEmail(l.ctx, req.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		l.Errorw("FindOneByEmail Error", logger.Field("error", err))
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "query user info failed")
	} else if err == nil && !u.DeletedAt.Valid {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.UserExist), "email is already registered")
	} else if err == nil && u.DeletedAt.Valid {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.UserDisabled), "account is disabled")
	}

	// Generate password
	pwd, err := tool.EncodePassWord(req.Password)
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "password does not meet policy")
	}
	userInfo := &user.User{
		Password:          pwd,
		Algo:              tool.PasswordAlgo,
		OnlyFirstPurchase: &invite.OnlyFirstPurchase,
	}
	if referer != nil {
		userInfo.RefererId = referer.Id
	}
	err = l.svcCtx.UserModel.Transaction(l.ctx, func(db *gorm.DB) error {
		// Save user information
		if err := l.svcCtx.UserModel.Insert(l.ctx, userInfo, db); err != nil {
			return err
		}
		// Generate ReferCode
		userInfo.ReferCode = uuidx.UserInviteCode(userInfo.Id)
		if err := l.svcCtx.UserModel.UpdateUserReferCode(l.ctx, userInfo.Id, userInfo.ReferCode, db); err != nil {
			return err
		}
		// create user auth info
		authInfo := &user.AuthMethods{
			UserId:         userInfo.Id,
			AuthType:       "email",
			AuthIdentifier: req.Email,
			Verified:       email.EnableVerify,
		}
		if err = l.svcCtx.UserModel.InsertUserAuthMethods(l.ctx, authInfo, db); err != nil {
			return err
		}

		if c.EnableTrial {
			// Active trial
			if err = l.activeTrial(userInfo.Id, db); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		l.Errorw("register transaction failed", logger.Field("email", logIdentifier(req.Email)), logger.Field("error", err.Error()))
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseInsertError), "register failed")
	}
	// Bind device to user if identifier is provided
	if req.Identifier != "" {
		bindLogic := NewBindDeviceLogic(l.ctx, l.svcCtx)
		if err := bindLogic.BindDeviceToUser(req.Identifier, req.IP, req.UserAgent, userInfo.Id); err != nil {
			l.Errorw("failed to bind device to user",
				logger.Field("user_id", userInfo.Id),
				logger.Field("identifier", logIdentifier(req.Identifier)),
				logger.Field("error", err.Error()),
			)
			// Don't fail register if device binding fails, just log the error
		}
	}
	loginType := authsession.ResolveLoginType(l.ctx, req.LoginType)
	token, err := authsession.Issue(l.ctx, l.svcCtx.Redis, l.svcCtx.Config.JwtAuth, userInfo.Id, loginType)
	if err != nil {
		l.Errorw("failed to issue authentication session",
			logger.Field("user_id", userInfo.Id),
			logger.Field("error", err.Error()),
		)
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "create authentication session")
	}
	defer func() {
		if token != "" && userInfo.Id != 0 {
			if err := recordLoginAudit(l.ctx, l.svcCtx, userInfo.Id, "email", req.IP, req.UserAgent, true); err != nil {
				l.Errorw("failed to insert login log",
					logger.Field("user_id", userInfo.Id),
					logger.Field("error", err.Error()),
				)
			}
			if err := recordRegisterAudit(l.ctx, l.svcCtx, userInfo.Id, "email", req.Email, req.IP, req.UserAgent); err != nil {
				l.Errorw("failed to insert register log",
					logger.Field("user_id", userInfo.Id),
					logger.Field("error", err.Error()),
				)
			}
		}
	}()
	return &types.LoginResponse{
		Token: token,
	}, nil
}

func (l *UserRegisterLogic) activeTrial(uid int64, tx ...*gorm.DB) error {
	register := l.svcCtx.ConfigSnapshot().Register
	sub, err := l.svcCtx.SubscribeModel.FindOneForOrderSnapshot(l.ctx, register.TrialSubscribe)
	if err != nil {
		return err
	}
	productSnapshot := ordersnapshot.FromSubscribe(sub)
	subscribeToken, err := uuidx.NewSubscribeToken()
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "generate trial subscription token")
	}
	startAt := time.Now()
	expireAt := tool.AddTime(register.TrialTimeUnit, register.TrialTime, startAt)
	resetCycle := subscriptionstate.NormalizeResetCycle(productSnapshot.ResetCycle)
	userSub := &user.Subscribe{
		UserId:          uid,
		OrderId:         0,
		SubscribeId:     sub.Id,
		ProductName:     productSnapshot.Name,
		ProductSnapshot: ordersnapshot.Marshal(productSnapshot),
		StartTime:       startAt,
		ExpireTime:      expireAt,
		Traffic:         sub.Traffic,
		Download:        0,
		Upload:          0,
		ResetCycle:      resetCycle,
		NextTrafficResetAt: subscriptionstate.NextTrafficResetAt(
			resetCycle,
			startAt,
			startAt,
			expireAt,
			subscriptionstate.TrafficResetLocation(),
		),
		Token:  subscribeToken,
		UUID:   uuidx.NewUUID().String(),
		Status: 1,
	}
	return l.svcCtx.UserModel.InsertSubscribe(l.ctx, userSub, tx...)
}
