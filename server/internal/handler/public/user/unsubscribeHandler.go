package user

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/public/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Unsubscribe
func UnsubscribeHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.UnsubscribeRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := user.NewUnsubscribeLogic(c.Request.Context(), svcCtx)
		err := l.Unsubscribe(&req)
		result.HttpResult(c, nil, err)
	}
}
