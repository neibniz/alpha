package payment

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/payment"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Create Payment Method
func CreatePaymentMethodHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.CreatePaymentMethodRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := payment.NewCreatePaymentMethodLogic(c.Request.Context(), svcCtx)
		resp, err := l.CreatePaymentMethod(&req)
		result.HttpResult(c, resp, err)
	}
}
