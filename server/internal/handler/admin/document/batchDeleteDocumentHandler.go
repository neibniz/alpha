package document

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/document"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Batch delete document
func BatchDeleteDocumentHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.BatchDeleteDocumentRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := document.NewBatchDeleteDocumentLogic(c.Request.Context(), svcCtx)
		err := l.BatchDeleteDocument(&req)
		result.HttpResult(c, nil, err)
	}
}
