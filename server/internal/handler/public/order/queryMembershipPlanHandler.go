package order

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/public/order"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
)

func QueryMembershipPlanHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		l := order.NewQueryMembershipPlanLogic(c.Request.Context(), svcCtx)
		resp, err := l.QueryMembershipPlan()
		result.HttpResult(c, resp, err)
	}
}
