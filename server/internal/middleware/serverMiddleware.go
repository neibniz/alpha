package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"

	"github.com/neibniz/alpha/server/internal/forwarderauth"
	"github.com/neibniz/alpha/server/internal/nodeauth"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/gin-gonic/gin"
)

const serverIDHeader = "X-Alpha-Server-ID"
const forwarderIDHeader = "X-Alpha-Forwarder-ID"

type forwarderContextKey struct{}

func ServerMiddleware(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverID, err := strconv.ParseInt(strings.TrimSpace(c.GetHeader(serverIDHeader)), 10, 64)
		if err != nil || serverID <= 0 {
			abortNodeAuth(c)
			return
		}
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok || !nodeauth.VerifyToken(svc.ConfigSnapshot().Node.NodeSecret, serverID, token) {
			abortNodeAuth(c)
			return
		}
		if pathID := strings.TrimSpace(c.Param("server_id")); pathID != "" {
			parsedPathID, parseErr := strconv.ParseInt(pathID, 10, 64)
			if parseErr != nil || parsedPathID != serverID {
				abortNodeAuth(c)
				return
			}
		}
		ctx := context.WithValue(c.Request.Context(), constant.CtxKeyServerID, serverID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func ServerDeployMiddleware(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		expected := svc.ConfigSnapshot().Node.NodeDeploySecret
		if !ok || expected == "" || len(expected) != len(token) || subtle.ConstantTimeCompare([]byte(expected), []byte(token)) != 1 {
			abortNodeAuth(c)
			return
		}
		c.Next()
	}
}

func ForwarderBootstrapMiddleware(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		expected := strings.TrimSpace(svc.ConfigSnapshot().Forwarder.BootstrapSecret)
		if !ok || expected == "" || len(expected) != len(token) || subtle.ConstantTimeCompare([]byte(expected), []byte(token)) != 1 {
			abortNodeAuth(c)
			return
		}
		c.Next()
	}
}

func ForwarderInstanceMiddleware(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		instanceID := strings.TrimSpace(c.GetHeader(forwarderIDHeader))
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if instanceID == "" || !ok || !forwarderauth.VerifyToken(svc.ConfigSnapshot().Forwarder.BootstrapSecret, instanceID, token) {
			abortNodeAuth(c)
			return
		}
		ctx := context.WithValue(c.Request.Context(), forwarderContextKey{}, instanceID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func AuthenticatedForwarderID(ctx context.Context) (string, bool) {
	instanceID, ok := ctx.Value(forwarderContextKey{}).(string)
	instanceID = strings.TrimSpace(instanceID)
	return instanceID, ok && instanceID != ""
}

func AuthenticatedServerID(ctx context.Context) (int64, bool) {
	serverID, ok := ctx.Value(constant.CtxKeyServerID).(int64)
	return serverID, ok && serverID > 0
}

func bearerToken(value string) (string, bool) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(value), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" || strings.Contains(token, " ") {
		return "", false
	}
	return token, true
}

func abortNodeAuth(c *gin.Context) {
	c.Header("WWW-Authenticate", "Bearer")
	c.String(http.StatusUnauthorized, "Unauthorized")
	c.Abort()
}
