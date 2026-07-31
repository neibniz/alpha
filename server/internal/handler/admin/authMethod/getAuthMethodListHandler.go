package authMethod

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/authMethod"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Get auth method list
func GetAuthMethodListHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {

		l := authMethod.NewGetAuthMethodListLogic(c.Request.Context(), svcCtx)
		resp, err := l.GetAuthMethodList()
		result.HttpResult(c, resp, err)
	}
}
