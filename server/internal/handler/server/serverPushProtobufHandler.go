package server

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/neibniz/alpha/server/internal/agentprotocol"
	logicserver "github.com/neibniz/alpha/server/internal/logic/server"
	"github.com/neibniz/alpha/server/internal/middleware"
	agentv1 "github.com/neibniz/alpha/server/internal/protocol/alpha/agent/v1"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

const agentProtocolHeader = "X-Alpha-Agent-Protocol"

var errUnsupportedProtobufMediaType = errors.New("unsupported protobuf media type")

func ServerPushTrafficReportsProtobufHandler(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var message agentv1.PushTrafficReportsRequest
		if err := bindAgentProtobuf(c, &message, maxTrafficRequestBodyBytes, "traffic report"); err != nil {
			writeAgentProtobufError(c, protobufBindStatus(err), err)
			return
		}
		request := agentprotocol.TrafficRequest(&message)
		serverID, ok := middleware.AuthenticatedServerID(c.Request.Context())
		if !ok {
			writeAgentProtobufError(c, http.StatusUnauthorized, errors.New("node authentication is required"))
			return
		}
		request.ServerId = serverID
		if err := svcCtx.Validate(request); err != nil {
			writeAgentProtobufError(c, http.StatusBadRequest, err)
			return
		}
		logic := logicserver.NewServerPushUserTrafficLogic(c.Request.Context(), svcCtx)
		if err := logic.ServerPushUserTraffic(request); err != nil {
			writeAgentProtobufError(c, http.StatusUnprocessableEntity, err)
			return
		}
		c.Header(agentProtocolHeader, "protobuf-v1")
		c.Status(http.StatusNoContent)
	}
}

func ServerPushAccessEventsProtobufHandler(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var message agentv1.PushAccessEventsRequest
		if err := bindAgentProtobuf(c, &message, maxAccessRequestBodyBytes, "access event"); err != nil {
			writeAgentProtobufError(c, protobufBindStatus(err), err)
			return
		}
		request, err := agentprotocol.AccessRequest(&message)
		if err != nil {
			writeAgentProtobufError(c, http.StatusBadRequest, err)
			return
		}
		serverID, ok := middleware.AuthenticatedServerID(c.Request.Context())
		if !ok {
			writeAgentProtobufError(c, http.StatusUnauthorized, errors.New("node authentication is required"))
			return
		}
		request.ServerId = serverID
		logic := logicserver.NewServerPushAccessLogsLogic(c.Request.Context(), svcCtx)
		response, err := logic.ServerPushAccessLogs(request)
		if err != nil {
			writeAgentProtobufError(c, http.StatusUnprocessableEntity, err)
			return
		}
		writeAgentProtobuf(c, http.StatusOK, agentprotocol.AccessResponse(response))
	}
}

func ServerPushOnlineUsersProtobufHandler(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var message agentv1.PushOnlineUsersRequest
		if err := bindAgentProtobuf(c, &message, maxTrafficRequestBodyBytes, "online users"); err != nil {
			writeAgentProtobufError(c, protobufBindStatus(err), err)
			return
		}
		request := agentprotocol.OnlineUsersRequest(&message)
		serverID, ok := middleware.AuthenticatedServerID(c.Request.Context())
		if !ok {
			writeAgentProtobufError(c, http.StatusUnauthorized, errors.New("node authentication is required"))
			return
		}
		request.ServerId = serverID
		if err := svcCtx.Validate(request); err != nil {
			writeAgentProtobufError(c, http.StatusBadRequest, err)
			return
		}
		logic := logicserver.NewServerPushOnlineUsersLogic(c.Request.Context(), svcCtx)
		if err := logic.ServerPushOnlineUsers(request); err != nil {
			writeAgentProtobufError(c, http.StatusUnprocessableEntity, err)
			return
		}
		c.Header(agentProtocolHeader, "protobuf-v1")
		c.Status(http.StatusNoContent)
	}
}

func bindAgentProtobuf(c *gin.Context, message proto.Message, maxBytes int64, payloadName string) error {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(c.GetHeader("Content-Type")))
	if err != nil || !strings.EqualFold(mediaType, agentprotocol.ContentType) {
		return errUnsupportedProtobufMediaType
	}
	payload, err := readCompressedRequestBody(c.Request.Body, c.GetHeader("Content-Encoding"), maxBytes, payloadName)
	if err != nil {
		return err
	}
	if err := proto.Unmarshal(payload, message); err != nil {
		return fmt.Errorf("invalid protobuf %s payload: %w", payloadName, err)
	}
	return nil
}

func protobufBindStatus(err error) int {
	if errors.Is(err, errUnsupportedProtobufMediaType) {
		return http.StatusUnsupportedMediaType
	}
	return http.StatusBadRequest
}

func writeAgentProtobuf(c *gin.Context, status int, message proto.Message) {
	payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(message)
	if err != nil {
		writeAgentProtobufError(c, http.StatusInternalServerError, errors.New("failed to encode response"))
		return
	}
	c.Header(agentProtocolHeader, "protobuf-v1")
	c.Data(status, agentprotocol.ContentType, payload)
}

func writeAgentProtobufError(c *gin.Context, status int, err error) {
	message := "request failed"
	code := uint32(status)
	var codeError *xerr.CodeError
	if errors.As(err, &codeError) {
		code = codeError.GetErrCode()
		message = codeError.GetErrMsg()
	} else if status < http.StatusUnprocessableEntity && err != nil {
		message = err.Error()
	}
	payload, marshalErr := proto.MarshalOptions{Deterministic: true}.Marshal(&agentv1.ErrorResponse{
		Code:    code,
		Message: message,
	})
	if marshalErr != nil {
		c.AbortWithStatus(status)
		return
	}
	c.Header(agentProtocolHeader, "protobuf-v1")
	c.Data(status, agentprotocol.ContentType, payload)
}
