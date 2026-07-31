package server

import (
	"net/http"
	"time"

	"github.com/neibniz/alpha/server/internal/middleware"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var nodeStatusUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// NodeRealtimeStatusHandler receives high-frequency realtime node status over websocket.
func NodeRealtimeStatusHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		authenticatedServerID, ok := middleware.AuthenticatedServerID(c.Request.Context())
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		exists, err := svcCtx.NodeModel.ServerExists(c.Request.Context(), authenticatedServerID)
		if err != nil || !exists {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		upgrader := nodeStatusUpgrader
		upgrader.CheckOrigin = func(request *http.Request) bool {
			return middleware.OriginAllowed(request, svcCtx.Config.HTTP.AllowedOrigins)
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.WithContext(c.Request.Context()).Errorf("[NodeRealtimeStatusHandler] upgrade error: %v", err)
			return
		}
		defer conn.Close()

		serverID := authenticatedServerID
		var lastSeen int64
		conn.SetReadLimit(64 * 1024)
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		done := make(chan struct{})
		defer close(done)
		go func() {
			for {
				select {
				case <-ticker.C:
					_ = conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
				case <-done:
					return
				}
			}
		}()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				if serverID > 0 {
					svcCtx.NodeRealtime.MarkOfflineIfLastSeen(serverID, lastSeen)
				}
				return
			}
			state, err := realtime.DecodeNodeStatus(data)
			if err != nil || state.ServerID <= 0 {
				_ = conn.WriteJSON(gin.H{"type": "error", "msg": "invalid status payload"})
				continue
			}
			if state.ServerID != serverID {
				_ = conn.WriteJSON(gin.H{"type": "error", "msg": "server_id cannot change on this connection"})
				continue
			}
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			updated := svcCtx.NodeRealtime.Update(state)
			lastSeen = updated.LastSeen
			_ = conn.WriteJSON(gin.H{
				"type":       "ack",
				"server_id":  updated.ServerID,
				"updated_at": updated.UpdatedAt,
			})
		}
	}
}
