package server

import (
	"bytes"
	"io"

	"github.com/neibniz/alpha/server/internal/logic/server"
	"github.com/neibniz/alpha/server/internal/middleware"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

const maxTrafficRequestBodyBytes = int64(4 << 20)

// ServerPushUserTrafficHandler Push user Traffic
func ServerPushUserTrafficHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		payload, err := readCompressedRequestBody(
			c.Request.Body,
			c.GetHeader("Content-Encoding"),
			maxTrafficRequestBodyBytes,
			"traffic report",
		)
		if err != nil {
			result.ParamErrorResult(c, err)
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(payload))
		var req types.ServerPushUserTrafficRequest
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

		l := server.NewServerPushUserTrafficLogic(c.Request.Context(), svcCtx)
		err = l.ServerPushUserTraffic(&req)
		result.HttpResult(c, nil, err)
	}
}
