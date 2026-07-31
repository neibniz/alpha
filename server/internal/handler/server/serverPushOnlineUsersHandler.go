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

func ServerPushOnlineUsersHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4<<20)
		var req types.ServerPushOnlineUsersRequest
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

		l := server.NewServerPushOnlineUsersLogic(c.Request.Context(), svcCtx)
		err := l.ServerPushOnlineUsers(&req)
		result.HttpResult(c, nil, err)
	}
}
