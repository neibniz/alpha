package auth

import (
	"time"

	"github.com/neibniz/alpha/server/internal/logic/auth"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/neibniz/alpha/server/pkg/turnstile"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// User login
func UserLoginHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.UserLoginRequest
		_ = c.ShouldBind(&req)
		// get client ip
		req.IP = c.ClientIP()
		req.UserAgent = c.Request.UserAgent()
		verifyConfig := svcCtx.ConfigSnapshot().Verify
		if verifyConfig.LoginVerify {
			verifyTurns := turnstile.New(turnstile.Config{
				Secret:  verifyConfig.TurnstileSecret,
				Timeout: 3 * time.Second,
			})
			if verify, err := verifyTurns.Verify(c, req.CfToken, req.IP); err != nil || !verify {
				err = errors.Wrapf(xerr.NewErrCode(xerr.TooManyRequests), "error: %v, verify: %v", err, verify)
				result.HttpResult(c, nil, err)
				return
			}
		}
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := auth.NewUserLoginLogic(c.Request.Context(), svcCtx)
		resp, err := l.UserLogin(&req)
		result.HttpResult(c, resp, err)
	}
}
