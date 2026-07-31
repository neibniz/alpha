package subscribe

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/subscribe"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Delete subscribe group
func DeleteSubscribeGroupHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.DeleteSubscribeGroupRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := subscribe.NewDeleteSubscribeGroupLogic(c.Request.Context(), svcCtx)
		err := l.DeleteSubscribeGroup(&req)
		result.HttpResult(c, nil, err)
	}
}
