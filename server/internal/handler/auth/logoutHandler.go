package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/neibniz/alpha/server/internal/clientip"
	"github.com/neibniz/alpha/server/internal/logic/auth"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/result"
)

// Logout records a logout event and invalidates the current session.
func LogoutHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		l := auth.NewLogoutLogic(c.Request.Context(), svcCtx)
		err := l.Logout(clientip.Address(c.Request), c.Request.UserAgent())
		result.HttpResult(c, nil, err)
	}
}
