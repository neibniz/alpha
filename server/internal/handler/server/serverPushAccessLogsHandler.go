package server

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"github.com/neibniz/alpha/server/internal/logic/server"
	"github.com/neibniz/alpha/server/internal/middleware"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

const maxAccessRequestBodyBytes = int64(4 << 20)

func ServerPushAccessLogsHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		payload, err := readAccessRequestBody(c.Request.Body, c.GetHeader("Content-Encoding"))
		if err != nil {
			result.ParamErrorResult(c, err)
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(payload))
		var req types.ServerPushAccessLogsRequest
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
		logic := server.NewServerPushAccessLogsLogic(c.Request.Context(), svcCtx)
		resp, err := logic.ServerPushAccessLogs(&req)
		result.HttpResult(c, resp, err)
	}
}

func readAccessRequestBody(body io.Reader, contentEncoding string) ([]byte, error) {
	return readCompressedRequestBody(body, contentEncoding, maxAccessRequestBodyBytes, "access log")
}

func readCompressedRequestBody(body io.Reader, contentEncoding string, maxBytes int64, payloadName string) ([]byte, error) {
	compressed, err := readRequestBodyLimit(body, maxBytes, payloadName)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(contentEncoding)) {
	case "":
		return compressed, nil
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(compressed))
		if err != nil {
			return nil, fmt.Errorf("invalid gzip %s payload: %w", payloadName, err)
		}
		defer reader.Close()
		return readRequestBodyLimit(reader, maxBytes, payloadName)
	default:
		return nil, fmt.Errorf("unsupported content encoding")
	}
}

func readRequestBodyLimit(reader io.Reader, maxBytes int64, payloadName string) ([]byte, error) {
	payload, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > maxBytes {
		return nil, fmt.Errorf("%s payload exceeds %d bytes", payloadName, maxBytes)
	}
	return payload, nil
}
