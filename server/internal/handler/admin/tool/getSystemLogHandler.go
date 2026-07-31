package tool

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/tool"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Get System Log
func GetSystemLogHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {

		l := tool.NewGetSystemLogLogic(c.Request.Context(), svcCtx)
		resp, err := l.GetSystemLog()
		result.HttpResult(c, resp, err)
	}
}
