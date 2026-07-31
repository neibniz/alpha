package server

import (
	"net/http"
	"strconv"

	"github.com/neibniz/alpha/server/internal/logic/server"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

func QueryServerXrayConfigHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.QueryServerXrayConfigRequest
		serverID, err := strconv.ParseInt(c.Param("server_id"), 10, 64)
		if err != nil {
			logger.Debugf("[QueryServerXrayConfigHandler] - strconv.ParseInt(server_id) error: %v, Param: %s", err, c.Param("server_id"))
			c.String(http.StatusBadRequest, "Invalid Params")
			c.Abort()
			return
		}
		req.ServerID = serverID
		if err = c.ShouldBindQuery(&req); err != nil {
			logger.Debugf("[QueryServerXrayConfigHandler] - ShouldBindQuery error: %v", err)
			c.String(http.StatusBadRequest, "Invalid Params")
			c.Abort()
			return
		}
		l := server.NewQueryServerXrayConfigLogic(c.Request.Context(), svcCtx)
		resp, err := l.QueryServerXrayConfig(&req)
		result.HttpResult(c, resp, err)
	}
}
