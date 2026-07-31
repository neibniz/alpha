package system

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/system"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Get register config
func GetRegisterConfigHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {

		l := system.NewGetRegisterConfigLogic(c.Request.Context(), svcCtx)
		resp, err := l.GetRegisterConfig()
		result.HttpResult(c, resp, err)
	}
}
