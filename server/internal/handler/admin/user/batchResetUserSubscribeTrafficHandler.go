package user

import (
	"github.com/neibniz/alpha/server/internal/logic/admin/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

func BatchResetUserSubscribeTrafficHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.BatchResetUserSubscribeTrafficRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			result.ParamErrorResult(c, err)
			return
		}
		if err := svcCtx.Validate(&req); err != nil {
			result.ParamErrorResult(c, err)
			return
		}
		logic := user.NewBatchResetUserSubscribeTrafficLogic(c.Request.Context(), svcCtx)
		resp, err := logic.Reset(&req)
		result.HttpResult(c, resp, err)
	}
}
