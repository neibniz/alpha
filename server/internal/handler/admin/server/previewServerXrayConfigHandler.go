package server

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/server"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

func PreviewServerXrayConfigHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.PreviewServerXrayConfigRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}
		l := server.NewPreviewServerXrayConfigLogic(c.Request.Context(), svcCtx)
		resp, err := l.PreviewServerXrayConfig(&req)
		result.HttpResult(c, resp, err)
	}
}
