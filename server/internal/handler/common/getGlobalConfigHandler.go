package common

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/common"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Get global config
func GetGlobalConfigHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		l := common.NewGetGlobalConfigLogic(c.Request.Context(), svcCtx)
		resp, err := l.GetGlobalConfig()
		result.HttpResult(c, resp, err)
	}
}
