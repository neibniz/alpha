package application

import (
	"github.com/neibniz/alpha/server/internal/logic/admin/application"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

// Get subscribe application
func GetSubscribeApplicationHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.GetSubscribeApplicationRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := application.NewGetSubscribeApplicationLogic(c.Request.Context(), svcCtx)
		resp, err := l.GetSubscribeApplication(&req)
		result.HttpResult(c, resp, err)
	}
}
