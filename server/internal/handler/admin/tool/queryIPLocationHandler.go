package tool

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/tool"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// QueryIPLocationHandler Query IP Location
func QueryIPLocationHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.QueryIPLocationRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := tool.NewQueryIPLocationLogic(c.Request.Context(), svcCtx)
		resp, err := l.QueryIPLocation(&req)
		result.HttpResult(c, resp, err)
	}
}
