package user

import (
	"github.com/neibniz/alpha/server/internal/logic/public/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

func QueryUserWalletLogHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.QueryUserWalletLogListRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			result.ParamErrorResult(c, err)
			return
		}
		resp, err := user.NewQueryUserWalletLogLogic(c.Request.Context(), svcCtx).QueryUserWalletLog(&req)
		result.HttpResult(c, resp, err)
	}
}
