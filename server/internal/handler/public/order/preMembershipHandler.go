package order

import (
	"github.com/gin-gonic/gin"
	orderLogic "github.com/neibniz/alpha/server/internal/logic/public/order"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

func PreMembershipHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.PreMembershipOrderRequest
		_ = c.ShouldBind(&req)
		if validateErr := svcCtx.Validate(&req); validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		logic := orderLogic.NewPreMembershipLogic(c.Request.Context(), svcCtx)
		resp, err := logic.PreMembership(&req)
		result.HttpResult(c, resp, err)
	}
}
