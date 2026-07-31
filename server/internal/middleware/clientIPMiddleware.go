package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/clientip"
	"github.com/neibniz/alpha/server/pkg/constant"
)

func ClientIPMiddleware(resolver *clientip.Resolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		result := resolver.Resolve(c.Request)
		ctx := clientip.WithResult(c.Request.Context(), result)
		ctx = context.WithValue(ctx, constant.CtxKeyUserAgent, c.Request.UserAgent())
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
