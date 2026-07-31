package user

import (
	"github.com/neibniz/alpha/server/internal/logic/public/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

func QueryUserSubscribeTrafficSeriesHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.QueryUserSubscribeTrafficSeriesRequest
		_ = c.ShouldBind(&req)
		if err := svcCtx.Validate(&req); err != nil {
			result.ParamErrorResult(c, err)
			return
		}

		logic := user.NewQueryUserSubscribeTrafficSeriesLogic(c.Request.Context(), svcCtx)
		resp, err := logic.QueryUserSubscribeTrafficSeries(&req)
		result.HttpResult(c, resp, err)
	}
}
