package subscribe

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/public/subscribe"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// QuerySubscribeListHandler Get subscribe list
func QuerySubscribeListHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.QuerySubscribeListRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := subscribe.NewQuerySubscribeListLogic(c.Request.Context(), svcCtx)
		resp, err := l.QuerySubscribeList(&req)
		result.HttpResult(c, resp, err)
	}
}
