package user

import (
	"github.com/neibniz/alpha/server/internal/logic/admin/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

// Get user subcribe access
func GetUserSubscribeAccessHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.GetUserSubscribeAccessRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := user.NewGetUserSubscribeAccessLogic(c.Request.Context(), svcCtx)
		resp, err := l.GetUserSubscribeAccess(&req)
		result.HttpResult(c, resp, err)
	}
}
