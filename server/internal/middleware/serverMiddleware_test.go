package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/nodeauth"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/gin-gonic/gin"
)

func TestServerMiddlewareRequiresServerBoundHeaderToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &svc.ServiceContext{Config: config.Config{Node: config.NodeConfig{
		NodeSecret: "0123456789abcdef0123456789abcdef",
	}}}
	token, err := nodeauth.DeriveToken(service.Config.Node.NodeSecret, 7)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.GET("/v2/server/:server_id/xray-config", ServerMiddleware(service), func(c *gin.Context) {
		serverID, ok := AuthenticatedServerID(c.Request.Context())
		if !ok || serverID != 7 {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/v2/server/7/xray-config", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set(serverIDHeader, "7")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("authenticated response = %d, want 204", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/v2/server/8/xray-config?secret_key=legacy", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set(serverIDHeader, "7")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("cross-server response = %d, want 401", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/v2/server/7/xray-config?secret_key=legacy", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("query-secret response = %d, want 401", response.Code)
	}
}

func TestServerDeployMiddlewareUsesIndependentToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &svc.ServiceContext{Config: config.Config{Node: config.NodeConfig{
		NodeSecret:       "node-master-secret",
		NodeDeploySecret: "deployment-secret-0123456789abcdef",
	}}}
	router := gin.New()
	router.GET("/deploy", ServerDeployMiddleware(service), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/deploy", nil)
	request.Header.Set("Authorization", "Bearer "+service.Config.Node.NodeDeploySecret)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("deploy response = %d, want 204", response.Code)
	}
}
