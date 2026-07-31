package payment

import (
	"github.com/neibniz/alpha/server/internal/logic/admin/payment"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

// GetPaymentMethodHandler Get Payment Method
func GetPaymentMethodHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.GetPaymentMethodRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := payment.NewGetPaymentMethodLogic(c.Request.Context(), svcCtx)
		resp, err := l.GetPaymentMethod(&req)
		result.HttpResult(c, resp, err)
	}
}
