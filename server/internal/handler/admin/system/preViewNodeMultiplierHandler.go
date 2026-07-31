package system

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/system"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
)

// PreView Node Multiplier
func PreViewNodeMultiplierHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {

		l := system.NewPreViewNodeMultiplierLogic(c.Request.Context(), svcCtx)
		resp, err := l.PreViewNodeMultiplier()
		result.HttpResult(c, resp, err)
	}
}
