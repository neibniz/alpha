package subscribe

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/subscribe"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Reset all subscribe tokens
func ResetAllSubscribeTokenHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {

		l := subscribe.NewResetAllSubscribeTokenLogic(c.Request.Context(), svcCtx)
		resp, err := l.ResetAllSubscribeToken()
		result.HttpResult(c, resp, err)
	}
}
