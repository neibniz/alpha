package payment

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/payment"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// GetPaymentMethodListHandler Get Payment Method List
func GetPaymentMethodListHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.GetPaymentMethodListRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := payment.NewGetPaymentMethodListLogic(c.Request.Context(), svcCtx)
		resp, err := l.GetPaymentMethodList(&req)
		result.HttpResult(c, resp, err)
	}
}
