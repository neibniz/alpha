package server

import (
	"net/http"

	"github.com/neibniz/alpha/server/internal/logic/server"
	"github.com/neibniz/alpha/server/internal/middleware"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

// Push server status
func ServerPushStatusHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
		var req types.ServerPushStatusRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			result.ParamErrorResult(c, err)
			return
		}
		_ = c.ShouldBindQuery(&req.ServerCommon)
		serverID, ok := middleware.AuthenticatedServerID(c.Request.Context())
		if !ok {
			c.AbortWithStatus(401)
			return
		}
		req.ServerId = serverID
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := server.NewServerPushStatusLogic(c.Request.Context(), svcCtx)
		err := l.ServerPushStatus(&req)
		result.HttpResult(c, nil, err)
	}
}
