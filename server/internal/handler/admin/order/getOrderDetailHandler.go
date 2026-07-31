package order

import (
	"github.com/neibniz/alpha/server/internal/logic/admin/order"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

func GetOrderDetailHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.GetOrderDetailRequest
		_ = c.ShouldBind(&req)
		if err := svcCtx.Validate(&req); err != nil {
			result.ParamErrorResult(c, err)
			return
		}

		logic := order.NewGetOrderDetailLogic(c.Request.Context(), svcCtx)
		resp, err := logic.GetOrderDetail(&req)
		result.HttpResult(c, resp, err)
	}
}
