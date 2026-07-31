package order

import (
	"github.com/neibniz/alpha/server/internal/logic/public/order"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

func PreRenewalHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.PreRenewalOrderRequest
		_ = c.ShouldBind(&req)
		if validateErr := svcCtx.Validate(&req); validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		logic := order.NewPreRenewalLogic(c.Request.Context(), svcCtx)
		resp, err := logic.PreRenewal(&req)
		result.HttpResult(c, resp, err)
	}
}
