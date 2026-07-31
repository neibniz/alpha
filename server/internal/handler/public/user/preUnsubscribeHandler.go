package user

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/public/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Pre Unsubscribe
func PreUnsubscribeHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.PreUnsubscribeRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := user.NewPreUnsubscribeLogic(c.Request.Context(), svcCtx)
		resp, err := l.PreUnsubscribe(&req)
		result.HttpResult(c, resp, err)
	}
}
