package application

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/application"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Update subscribe application
func UpdateSubscribeApplicationHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.UpdateSubscribeApplicationRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := application.NewUpdateSubscribeApplicationLogic(c.Request.Context(), svcCtx)
		resp, err := l.UpdateSubscribeApplication(&req)
		result.HttpResult(c, resp, err)
	}
}
