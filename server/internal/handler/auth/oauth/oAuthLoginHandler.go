package oauth

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/auth/oauth"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// OAuth login
func OAuthLoginHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.OAthLoginRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := oauth.NewOAuthLoginLogic(c.Request.Context(), svcCtx)
		resp, err := l.OAuthLogin(&req)
		result.HttpResult(c, resp, err)
	}
}
