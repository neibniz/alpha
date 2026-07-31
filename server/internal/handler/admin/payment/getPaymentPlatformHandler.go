package payment

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/payment"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Get supported payment platform
func GetPaymentPlatformHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {

		l := payment.NewGetPaymentPlatformLogic(c.Request.Context(), svcCtx)
		resp, err := l.GetPaymentPlatform()
		result.HttpResult(c, resp, err)
	}
}
