package membership

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/membership"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

func UpdateMembershipPlanHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.UpdateMembershipPlanRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := membership.NewUpdateMembershipPlanLogic(c.Request.Context(), svcCtx)
		resp, err := l.UpdateMembershipPlan(&req)
		result.HttpResult(c, resp, err)
	}
}
