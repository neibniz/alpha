package ticket

import (
	"net/http"

	"github.com/neibniz/alpha/server/internal/logic/admin/ticket"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/ticketpolicy"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
)

// Create ticket follow
func CreateTicketFollowHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		var req types.CreateTicketFollowRequest
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, ticketpolicy.MaxRequestBodyBytes)
		if err := c.ShouldBindJSON(&req); err != nil {
			result.ParamErrorResult(c, err)
			return
		}
		validateErr := svcCtx.Validate(&req)
		if validateErr != nil {
			result.ParamErrorResult(c, validateErr)
			return
		}

		l := ticket.NewCreateTicketFollowLogic(c.Request.Context(), svcCtx)
		err := l.CreateTicketFollow(&req)
		result.HttpResult(c, nil, err)
	}
}
