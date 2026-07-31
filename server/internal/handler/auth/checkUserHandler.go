package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/auth"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Check user is exist
func CheckUserHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.CheckUserRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := auth.NewCheckUserLogic(c.Request.Context(), svcCtx)
		resp, err := l.CheckUser(&req)
		result.HttpResult(c, resp, err)
	}
}
