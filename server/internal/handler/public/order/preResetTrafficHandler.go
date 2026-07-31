package order

import (
	"github.com/neibniz/alpha/server/internal/logic/public/order"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

func PreResetTrafficHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.PreResetTrafficOrderRequest
		_ = c.ShouldBind(&req)
		if validateErr := svcCtx.Validate(&req); validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		logic := order.NewPreResetTrafficLogic(c.Request.Context(), svcCtx)
		resp, err := logic.PreResetTraffic(&req)
		result.HttpResult(c, resp, err)
	}
}
