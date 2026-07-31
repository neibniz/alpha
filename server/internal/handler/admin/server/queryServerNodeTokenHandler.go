package server

import (
	"github.com/neibniz/alpha/server/internal/logic/admin/server"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

func QueryServerNodeTokenHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.QueryServerNodeTokenRequest
		_ = c.ShouldBindQuery(&req)
		if err := svcCtx.Validate(&req); err != nil {
			result.ParamErrorResult(c, err)
			return
		}
		logic := server.NewQueryServerNodeTokenLogic(c.Request.Context(), svcCtx)
		resp, err := logic.QueryServerNodeToken(&req)
		result.HttpResult(c, resp, err)
	}
}
