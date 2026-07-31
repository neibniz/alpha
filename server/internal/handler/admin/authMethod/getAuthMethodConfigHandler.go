package authMethod

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/authMethod"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Get auth method config
func GetAuthMethodConfigHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.GetAuthMethodConfigRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := authMethod.NewGetAuthMethodConfigLogic(c.Request.Context(), svcCtx)
		resp, err := l.GetAuthMethodConfig(&req)
		result.HttpResult(c, resp, err)
	}
}
