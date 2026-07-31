package common

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/common"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Check verification code
func CheckVerificationCodeHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.CheckVerificationCodeRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := common.NewCheckVerificationCodeLogic(c.Request.Context(), svcCtx)
		resp, err := l.CheckVerificationCode(&req)
		result.HttpResult(c, resp, err)
	}
}
