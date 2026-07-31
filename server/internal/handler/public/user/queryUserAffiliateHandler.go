package user

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/public/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Query User Affiliate Count
func QueryUserAffiliateHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {

		l := user.NewQueryUserAffiliateLogic(c.Request.Context(), svcCtx)
		resp, err := l.QueryUserAffiliate()
		result.HttpResult(c, resp, err)
	}
}
