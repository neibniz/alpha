package server

import (
	"net/http"
	"time"

	"github.com/neibniz/alpha/server/internal/middleware"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/gin-gonic/gin"
)

func RealtimeForwarderStatusHandler(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validateAdminRealtimeToken(c, svcCtx) {
			c.String(http.StatusForbidden, "Forbidden")
			return
		}
		upgrader := adminStatusUpgrader
		upgrader.CheckOrigin = func(request *http.Request) bool {
			return middleware.OriginAllowed(request, svcCtx.Config.HTTP.AllowedOrigins)
		}
		upgrader.Subprotocols = []string{"alpha-auth"}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		unsubscribe := svcCtx.ForwarderRealtime.Subscribe(conn)
		defer unsubscribe()

		conn.SetReadLimit(4 * 1024)
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
			return nil
		})
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}
}
