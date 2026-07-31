package document

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/public/document"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Get document detail
func QueryDocumentDetailHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.QueryDocumentDetailRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := document.NewQueryDocumentDetailLogic(c.Request.Context(), svcCtx)
		resp, err := l.QueryDocumentDetail(&req)
		result.HttpResult(c, resp, err)
	}
}
