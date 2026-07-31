package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/gin-gonic/gin"
)

func CorsMiddleware(svc *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" {
			if !OriginAllowed(c.Request, svc.Config.HTTP.AllowedOrigins) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Origin, X-CSRF-Token, Authorization, Range, Login-Type")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Max-Age", "600")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func OriginAllowed(request *http.Request, allowedOrigins []string) bool {
	if request == nil {
		return false
	}
	origin := strings.TrimSpace(request.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.Path != "" {
		return false
	}
	if strings.EqualFold(parsed.Host, request.Host) {
		return true
	}
	canonical := parsed.Scheme + "://" + parsed.Host
	for _, allowed := range allowedOrigins {
		if strings.EqualFold(strings.TrimRight(strings.TrimSpace(allowed), "/"), canonical) {
			return true
		}
	}
	return false
}
