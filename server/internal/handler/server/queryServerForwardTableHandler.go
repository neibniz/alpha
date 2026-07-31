package server

import (
	"net/http"

	"github.com/neibniz/alpha/server/internal/logic/server"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

func QueryServerForwardTableHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.QueryServerForwardTableRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			logger.Debugf("[QueryServerForwardTableHandler] - ShouldBindQuery error: %v", err)
			c.String(http.StatusBadRequest, "Invalid Params")
			c.Abort()
			return
		}
		l := server.NewQueryServerForwardTableLogic(c.Request.Context(), svcCtx)
		resp, err := l.QueryServerForwardTable(&req)
		result.HttpResult(c, resp, err)
	}
}
