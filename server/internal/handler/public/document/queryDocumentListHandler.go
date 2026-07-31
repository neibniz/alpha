package document

import (
	"github.com/neibniz/alpha/server/internal/logic/public/document"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

// Get document list
func QueryDocumentListHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.QueryDocumentListRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := document.NewQueryDocumentListLogic(c.Request.Context(), svcCtx)
		resp, err := l.QueryDocumentList(&req)
		result.HttpResult(c, resp, err)
	}
}
