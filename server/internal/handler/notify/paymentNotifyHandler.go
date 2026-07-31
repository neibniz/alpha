package notify

import (
	"fmt"
	"net/http"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/logic/notify"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/payment"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

// PaymentNotifyHandler Payment Notify
func PaymentNotifyHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
		platform, ok := c.Request.Context().Value(constant.CtxKeyPlatform).(string)
		if !ok {
			logger.WithContext(c.Request.Context()).Errorf("platform not found")
			result.HttpResult(c, nil, fmt.Errorf("platform not found"))
			return
		}

		switch payment.ParsePlatform(platform) {
		case payment.EPay, payment.CryptoSaaS:
			req := &types.EPayNotifyRequest{}
			if err := c.ShouldBind(req); err != nil {
				result.HttpResult(c, nil, err)
				return
			}
			l := notify.NewEPayNotifyLogic(c, svcCtx)
			if err := l.EPayNotify(req); err != nil {
				logger.WithContext(c.Request.Context()).Errorf("EPayNotify failed: %v", err.Error())
				c.String(http.StatusBadRequest, "invalid payment notification")
				return
			}
			c.String(http.StatusOK, "%s", "success")
		case payment.Stripe:
			l := notify.NewStripeNotifyLogic(c.Request.Context(), svcCtx)
			if err := l.StripeNotify(c.Request, c.Writer); err != nil {
				result.HttpResult(c, nil, err)
				return
			}
			result.HttpResult(c, nil, nil)

		case payment.AlipayF2F:
			if err := c.Request.ParseForm(); err != nil {
				c.String(http.StatusBadRequest, "invalid payment notification")
				return
			}
			l := notify.NewAlipayNotifyLogic(c.Request.Context(), svcCtx)
			if err := l.AlipayNotify(c.Request); err != nil {
				result.HttpResult(c, nil, err)
				return
			}
			// Return success to alipay
			c.String(http.StatusOK, "%s", "success")

		default:
			logger.WithContext(c.Request.Context()).Errorf("platform %s not support", platform)
			c.String(http.StatusNotFound, "unsupported payment platform")
		}
	}
}
