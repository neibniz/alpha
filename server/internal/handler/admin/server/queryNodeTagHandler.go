package server

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/server"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Query all node tags
func QueryNodeTagHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {

		l := server.NewQueryNodeTagLogic(c.Request.Context(), svcCtx)
		resp, err := l.QueryNodeTag()
		result.HttpResult(c, resp, err)
	}
}
