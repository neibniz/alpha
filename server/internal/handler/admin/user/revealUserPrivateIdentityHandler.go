package user

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/clientip"
	"github.com/neibniz/alpha/server/internal/logic/admin/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
)

func RevealUserPrivateIdentityHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.RevealUserPrivateIdentityRequest
		_ = c.ShouldBindJSON(&req)
		if validateErr := svcCtx.Validate(&req); validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		c.Header("Cache-Control", "no-store, max-age=0")
		c.Header("Pragma", "no-cache")
		l := user.NewRevealUserPrivateIdentityLogic(c.Request.Context(), svcCtx)
		resp, err := l.RevealUserPrivateIdentity(&req, clientip.Address(c.Request), c.Request.UserAgent())
		result.HttpResult(c, resp, err)
	}
}
