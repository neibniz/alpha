package coupon

import (
	"github.com/neibniz/alpha/server/internal/logic/admin/coupon"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

// Get coupon
func GetCouponHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.GetCouponRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := coupon.NewGetCouponLogic(c.Request.Context(), svcCtx)
		resp, err := l.GetCoupon(&req)
		result.HttpResult(c, resp, err)
	}
}
