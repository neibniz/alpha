package system

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/system"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Get site config
func GetSiteConfigHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {

		l := system.NewGetSiteConfigLogic(c.Request.Context(), svcCtx)
		resp, err := l.GetSiteConfig()
		result.HttpResult(c, resp, err)
	}
}
