package user

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/public/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Query User Subscribe Preset List
func QueryUserSubscribePresetListHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.QueryUserSubscribePresetListRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := user.NewQueryUserSubscribePresetListLogic(c.Request.Context(), svcCtx)
		resp, err := l.QueryUserSubscribePresetList(&req)
		result.HttpResult(c, resp, err)
	}
}
