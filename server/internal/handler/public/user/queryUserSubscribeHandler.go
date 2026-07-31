package user

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/public/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Query User Subscribe
func QueryUserSubscribeHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {

		l := user.NewQueryUserSubscribeLogic(c.Request.Context(), svcCtx)
		resp, err := l.QueryUserSubscribe()
		result.HttpResult(c, resp, err)
	}
}
