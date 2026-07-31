package server

import (
	"github.com/neibniz/alpha/server/internal/logic/server"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

func QueryDeployServerListHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		l := server.NewQueryDeployServerListLogic(c.Request.Context(), svcCtx)
		resp, err := l.QueryDeployServerList()
		result.HttpResult(c, resp, err)
	}
}
