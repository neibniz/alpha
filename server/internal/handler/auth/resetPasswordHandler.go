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

// Reset password
func ResetPasswordHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.ResetPasswordRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}
		// get client ip
		req.IP = clientip.Address(c.Request)
		req.UserAgent = c.Request.UserAgent()
		verifyConfig := svcCtx.ConfigSnapshot().Verify
		if verifyConfig.ResetPasswordVerify {
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
		l := auth.NewResetPasswordLogic(c.Request.Context(), svcCtx)
		resp, err := l.ResetPassword(&req)
		result.HttpResult(c, resp, err)
	}
}
