package server

import (
	"strconv"

	"github.com/neibniz/alpha/server/internal/logic/server"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

func QueryServerXrayUsersHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.QueryServerXrayUsersRequest
		serverID, err := strconv.ParseInt(c.Param("server_id"), 10, 64)
		if err != nil {
			logger.Debugf("[QueryServerXrayUsersHandler] parse server_id error: %v, Param: %s", err, c.Param("server_id"))
			result.ParamErrorResult(c, err)
			return
		}
		req.ServerID = serverID
		if err := c.ShouldBindQuery(&req); err != nil {
			logger.Debugf("[QueryServerXrayUsersHandler] bind query error: %v", err)
			result.ParamErrorResult(c, err)
			return
		}
		l := server.NewQueryServerXrayUsersLogic(c.Request.Context(), svcCtx)
		resp, err := l.QueryServerXrayUsers(&req)
		result.HttpResult(c, resp, err)
	}
}
