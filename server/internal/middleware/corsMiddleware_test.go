package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/gin-gonic/gin"
)

func TestCorsMiddlewareUsesAllowlist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &svc.ServiceContext{Config: config.Config{HTTP: config.HTTPConfig{
		AllowedOrigins: []string{"https://user.example.test"},
	}}}
	router := gin.New()
	router.Use(CorsMiddleware(service))
	router.GET("/ok", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	request := httptest.NewRequest(http.MethodGet, "https://api.example.test/ok", nil)
	request.Header.Set("Origin", "https://user.example.test")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != "https://user.example.test" {
		t.Fatalf("allowed origin response = %d, %q", response.Code, response.Header().Get("Access-Control-Allow-Origin"))
	}
	if headers := response.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(headers, "Login-Type") {
		t.Fatalf("allowed headers %q do not include Login-Type", headers)
	}

	request = httptest.NewRequest(http.MethodGet, "https://api.example.test/ok", nil)
	request.Header.Set("Origin", "https://evil.example.test")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("denied origin response = %d, %q", response.Code, response.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestOriginAllowedAcceptsSameOriginAndNonBrowser(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "https://api.example.test/ok", nil)
	if !OriginAllowed(request, nil) {
		t.Fatal("request without Origin should be allowed")
	}
	request.Header.Set("Origin", "https://api.example.test")
	if !OriginAllowed(request, nil) {
		t.Fatal("same-origin request should be allowed")
	}
}
