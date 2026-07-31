package announcement

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/logic/admin/announcement"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Delete announcement
func DeleteAnnouncementHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.DeleteAnnouncementRequest
		_ = c.ShouldBind(&req)
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := announcement.NewDeleteAnnouncementLogic(c.Request.Context(), svcCtx)
		err := l.DeleteAnnouncement(&req)
		result.HttpResult(c, nil, err)
	}
}
