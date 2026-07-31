package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/clientip"
	"github.com/neibniz/alpha/server/internal/logic/auth"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/neibniz/alpha/server/pkg/turnstile"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

// User register
func UserRegisterHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.UserRegisterRequest
		_ = c.ShouldBind(&req)
		// get client ip
		req.IP = clientip.Address(c.Request)
		req.UserAgent = c.Request.UserAgent()
		verifyConfig := svcCtx.ConfigSnapshot().Verify
		if verifyConfig.RegisterVerify {
			verifyTurns := turnstile.New(turnstile.Config{
				Secret:  verifyConfig.TurnstileSecret,
				Timeout: 3 * time.Second,
			})
			if verify, err := verifyTurns.Verify(c, req.CfToken, req.IP); err != nil || !verify {
				result.HttpResult(c, nil, errors.Wrapf(xerr.NewErrCode(xerr.TooManyRequests), "verify error: %v", err.Error()))
				return
			}
		}
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := auth.NewUserRegisterLogic(c.Request.Context(), svcCtx)
		resp, err := l.UserRegister(&req)
		result.HttpResult(c, resp, err)
	}
}
