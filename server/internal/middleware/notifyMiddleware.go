package middleware

import (
	"context"
	"net/http"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/gin-gonic/gin"
)

type PaymentParams struct {
	Platform string `uri:"platform"`
	Token    string `uri:"token"`
}

func NotifyMiddleware(svc *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		var params PaymentParams
		// Get platform and token from uri
		if err := c.ShouldBindUri(&params); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		config, err := svc.PaymentModel.FindOneByPaymentToken(ctx, params.Token)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment endpoint not found"})
			c.Abort()
			return
		}
		ctx = context.WithValue(ctx, constant.CtxKeyPlatform, config.Platform)
		ctx = context.WithValue(ctx, constant.CtxKeyPayment, config)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
