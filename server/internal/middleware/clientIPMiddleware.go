package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/clientip"
)

func ClientIPMiddleware(resolver *clientip.Resolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		result := resolver.Resolve(c.Request)
		c.Request = c.Request.WithContext(clientip.WithResult(c.Request.Context(), result))
		c.Next()
	}
}
