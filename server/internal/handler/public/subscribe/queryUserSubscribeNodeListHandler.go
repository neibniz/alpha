package subscribe

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/public/subscribe"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Get user subscribe node info
func QueryUserSubscribeNodeListHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {

		l := subscribe.NewQueryUserSubscribeNodeListLogic(c.Request.Context(), svcCtx)
		resp, err := l.QueryUserSubscribeNodeList()
		result.HttpResult(c, resp, err)
	}
}
