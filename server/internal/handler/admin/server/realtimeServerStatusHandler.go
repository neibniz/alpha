package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/middleware"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/jwt"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var adminStatusUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type realtimeSubscribeMessage struct {
	Type      string  `json:"type"`
	ServerIDs []int64 `json:"server_ids"`
}

const maxRealtimeServerSubscriptions = 256

func RealtimeServerStatusHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		if !validateAdminRealtimeToken(c, svcCtx) {
			c.String(http.StatusForbidden, "Forbidden")
			return
		}

		serverIDs := parseServerIDs(c.Query("server_ids"))
		upgrader := adminStatusUpgrader
		upgrader.CheckOrigin = func(request *http.Request) bool {
			return middleware.OriginAllowed(request, svcCtx.Config.HTTP.AllowedOrigins)
		}
		upgrader.Subprotocols = []string{"alpha-auth"}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.WithContext(c.Request.Context()).Errorf("[RealtimeServerStatusHandler] upgrade error: %v", err)
			return
		}
		defer conn.Close()

		unsubscribe := svcCtx.NodeRealtime.Subscribe(conn, serverIDs)
		defer unsubscribe()

		svcCtx.NodeRealtime.SendSnapshot(conn, serverIDs)

		conn.SetReadLimit(16 * 1024)
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
			return nil
		})

		ticker := time.NewTicker(30 * time.Second)
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
			var msg realtimeSubscribeMessage
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			if msg.Type == "subscribe" {
				serverIDs = normalizeRealtimeServerIDs(msg.ServerIDs)
				svcCtx.NodeRealtime.SetSubscription(conn, serverIDs)
				svcCtx.NodeRealtime.SendSnapshot(conn, serverIDs)
			}
		}
	}
}

func validateAdminRealtimeToken(c *gin.Context, svcCtx *svc.ServiceContext) bool {
	token := adminRealtimeWebSocketToken(c)
	if token == "" {
		return false
	}
	claims, err := jwt.ParseJwtToken(token, svcCtx.Config.JwtAuth.AccessSecret)
	if err != nil {
		logger.WithContext(c.Request.Context()).Debug("[RealtimeServerStatusHandler] ParseJwtToken", logger.Field("error", err.Error()))
		return false
	}
	userIDFloat, ok := claims["UserId"].(float64)
	if !ok {
		return false
	}
	sessionID, ok := claims["SessionId"].(string)
	if !ok || sessionID == "" {
		return false
	}
	key := config.SessionCacheKey(sessionID)
	value, err := svcCtx.Redis.Get(c.Request.Context(), key).Result()
	if err != nil || value != config.SessionUserValue(int64(userIDFloat)) {
		return false
	}
	userInfo, err := svcCtx.UserModel.FindOneForAdminCheck(c.Request.Context(), int64(userIDFloat))
	if err != nil || userInfo.DeletedAt.Valid || userInfo.IsAdmin == nil || !*userInfo.IsAdmin || (userInfo.Enable != nil && !*userInfo.Enable) {
		return false
	}
	return true
}

func adminRealtimeWebSocketToken(c *gin.Context) string {
	if authorization := strings.TrimSpace(c.GetHeader("Authorization")); authorization != "" {
		if scheme, token, ok := strings.Cut(authorization, " "); ok && strings.EqualFold(scheme, "Bearer") {
			return strings.TrimSpace(token)
		}
	}
	protocols := websocket.Subprotocols(c.Request)
	for index := 0; index+1 < len(protocols); index++ {
		if protocols[index] == "alpha-auth" {
			return strings.TrimSpace(protocols[index+1])
		}
	}
	return ""
}

func parseServerIDs(raw string) []int64 {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return normalizeRealtimeServerIDs(ids)
}

func normalizeRealtimeServerIDs(serverIDs []int64) []int64 {
	if len(serverIDs) == 0 {
		return nil
	}
	limit := min(len(serverIDs), maxRealtimeServerSubscriptions)
	ids := make([]int64, 0, limit)
	seen := make(map[int64]struct{}, limit)
	for _, id := range serverIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
		if len(ids) == maxRealtimeServerSubscriptions {
			break
		}
	}
	return ids
}
