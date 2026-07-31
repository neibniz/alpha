package user

import (
	"github.com/neibniz/alpha/server/internal/logic/admin/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

func QuoteUserSubscribeOrderHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.IssueUserSubscribeOrderRequest
		_ = c.ShouldBind(&req)
		if err := svcCtx.Validate(&req); err != nil {
			result.ParamErrorResult(c, err)
			return
		}
		logic := user.NewIssueUserSubscribeOrderLogic(c.Request.Context(), svcCtx)
		resp, err := logic.Quote(&req)
		result.HttpResult(c, resp, err)
	}
}
