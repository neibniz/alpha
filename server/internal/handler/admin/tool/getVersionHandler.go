package tool

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/tool"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
)

// GetVersionHandler Get Version
func GetVersionHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {

		l := tool.NewGetVersionLogic(c.Request.Context(), svcCtx)
		resp, err := l.GetVersion()
		result.HttpResult(c, resp, err)
	}
}
