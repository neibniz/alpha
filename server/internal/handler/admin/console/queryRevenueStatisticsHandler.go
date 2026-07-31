package console

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/console"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Query revenue statistics
func QueryRevenueStatisticsHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.StatisticsPeriodRequest
		_ = c.ShouldBindQuery(&req)

		l := console.NewQueryRevenueStatisticsLogic(c.Request.Context(), svcCtx)
		resp, err := l.QueryRevenueStatistics(&req)
		result.HttpResult(c, resp, err)
	}
}
