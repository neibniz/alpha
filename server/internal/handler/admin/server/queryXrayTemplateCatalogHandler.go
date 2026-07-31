package server

import (
	"github.com/neibniz/alpha/server/internal/logic/admin/server"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

func QueryXrayTemplateCatalogHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.QueryXrayTemplateCatalogRequest
		_ = c.ShouldBindQuery(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}
		l := server.NewQueryXrayTemplateCatalogLogic(c.Request.Context(), svcCtx)
		resp, err := l.QueryXrayTemplateCatalog(&req)
		result.HttpResult(c, resp, err)
	}
}
