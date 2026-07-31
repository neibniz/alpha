package portal

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/public/portal"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Get Subscription
func GetSubscriptionHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.GetSubscriptionRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := portal.NewGetSubscriptionLogic(c.Request.Context(), svcCtx)
		resp, err := l.GetSubscription(&req)
		result.HttpResult(c, resp, err)
	}
}
