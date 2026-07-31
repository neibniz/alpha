package log

import (
	adminLog "github.com/neibniz/alpha/server/internal/logic/admin/log"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

func FilterProxyAccessLogHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.FilterProxyAccessLogRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			result.ParamErrorResult(c, err)
			return
		}
		logic := adminLog.NewFilterProxyAccessLogLogic(c.Request.Context(), svcCtx)
		resp, err := logic.FilterProxyAccessLog(&req)
		result.HttpResult(c, resp, err)
	}
}
