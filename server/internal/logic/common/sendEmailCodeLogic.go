package common

import (
	"context"
	"encoding/json"
	"time"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/limit"
	"github.com/neibniz/alpha/server/pkg/random"
	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	queue "github.com/neibniz/alpha/server/queue/types"
)

type SendEmailCodeLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

const (
	IntervalTime = 60
)

type VerifyTemplate struct {
	Type     uint8
	SiteLogo string
	SiteName string
	Expire   uint8
	Code     string
}
type CacheKeyPayload struct {
	Code   string `json:"code"`
	LastAt int64  `json:"lastAt"`
}

// NewSendEmailCodeLogic Get verification code
func NewSendEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendEmailCodeLogic {
	return &SendEmailCodeLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendEmailCodeLogic) SendEmailCode(req *types.SendCodeRequest) (resp *types.SendCodeResponse, err error) {
	verifyType := constant.ParseVerifyType(req.Type)
	verifyTypeText := verifyType.String()
	privacyToken := l.svcCtx.Privacy.CacheToken("email", req.Email)
	// Check if there is Redis in the code
	cacheKey := config.EmailAuthCodeCacheKey(verifyTypeText, privacyToken)
	// Check if the limit is exceeded of current request
	limiter := limit.NewPeriodLimit(60, 1, l.svcCtx.Redis, config.SendIntervalCacheKey("email", verifyTypeText))
	permit, err := limiter.Take(privacyToken)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "Failed to take limit")
	}
	if !limiter.ParsePermitState(permit) {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.TooManyRequests), "send email too many requests")
	}
	// Check if the limit is exceeded of today
	permit, err = l.svcCtx.AuthLimiter.Take(config.SendCountLimitValue("email", verifyTypeText, privacyToken))
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "Failed to take limit")
	}
	if !l.svcCtx.AuthLimiter.ParsePermitState(permit) {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.TodaySendCountExceedsLimit), "send email too many requests")
	}
	m, err := l.svcCtx.UserModel.FindUserAuthMethodByOpenID(l.ctx, "email", req.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "FindUserAuthMethodByOpenID error")
	}
	authMethodExists := m != nil && m.Id > 0
	if verifyType == constant.Register && authMethodExists {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.UserExist), "mobile already bind")
	} else if verifyType == constant.Security && !authMethodExists {
		return &types.SendCodeResponse{Status: true}, nil
	}

	var payload CacheKeyPayload
	var taskPayload queue.SendEmailPayload
	// Generate verification code
	code, err := random.SecureKey(6, 0)
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "Failed to generate verification code")
	}
	taskPayload.Type = queue.EmailTypeVerify
	taskPayload.Email = req.Email
	taskPayload.Subject = "Verification code"
	taskPayload.Content, err = queue.EncodeEmailContent(queue.VerifyEmailContent{
		Type:     req.Type,
		SiteLogo: l.svcCtx.ConfigSnapshot().Site.SiteLogo,
		SiteName: l.svcCtx.ConfigSnapshot().Site.SiteName,
		Expire:   5,
		Code:     code,
	})
	if err != nil {
		l.Errorw("[SendEmailCode]: Encode content error", logger.Field("error", err.Error()))
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "Failed to encode email content")
	}
	// Save to Redis
	payload = CacheKeyPayload{
		Code:   code,
		LastAt: time.Now().Unix(),
	}
	// Marshal the payload
	val, _ := json.Marshal(payload)
	if err = l.svcCtx.Redis.Set(l.ctx, cacheKey, string(val), time.Second*IntervalTime*5).Err(); err != nil {
		l.Errorw("[SendEmailCode]: Redis Error", logger.Field("error", err.Error()))
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "Failed to set verification code")
	}

	// Marshal the task payload
	payloadBuy, err := queue.EncodePrivatePayload(l.svcCtx.Privacy, queue.EmailPayloadScope, taskPayload)
	if err != nil {
		l.Errorw("[SendEmailCode]: Marshal Error", logger.Field("error", err.Error()))
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "Failed to marshal task payload")
	}
	// Create a queue task
	task := asynq.NewTask(queue.ForthwithSendEmail, payloadBuy, asynq.MaxRetry(3))
	// Enqueue the task
	taskInfo, err := l.svcCtx.Queue.Enqueue(task)
	if err != nil {
		l.Errorw("[SendEmailCode]: Enqueue Error", logger.Field("error", err.Error()))
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "Failed to enqueue task")
	}
	l.Infow("[SendEmailCode]: Enqueue Success", logger.Field("taskID", taskInfo.ID))
	if l.svcCtx.Config.Model == constant.DevMode {
		return &types.SendCodeResponse{
			Code:   payload.Code,
			Status: true,
		}, nil
	} else {
		return &types.SendCodeResponse{
			Status: true,
		}, nil
	}
}
