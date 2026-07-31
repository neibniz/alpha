package handler

import (
	"crypto/subtle"
	"net/http"

	"github.com/neibniz/alpha/server/internal/logic/telegram"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/result"
	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func RegisterTelegramHandlers(router *gin.Engine, serverCtx *svc.ServiceContext) {
	router.POST("/v1/telegram/webhook", TelegramHandler(serverCtx))
}

func TelegramHandler(svcCtx *svc.ServiceContext) func(c *gin.Context) {
	return func(c *gin.Context) {
		telegramConfig, _ := svcCtx.TelegramRuntime()
		secret := c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
		if !telegramConfig.Enable || telegramConfig.BotToken == "" || telegramConfig.WebhookSecret == "" ||
			len(secret) != len(telegramConfig.WebhookSecret) ||
			subtle.ConstantTimeCompare([]byte(secret), []byte(telegramConfig.WebhookSecret)) != 1 {
			logger.WithContext(c.Request.Context()).Error("[TelegramHandler] Secret is wrong", logger.Field("path", c.Request.URL.Path))
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
		var request tgbotapi.Update
		if err := c.BindJSON(&request); err != nil {
			logger.WithContext(c.Request.Context()).Error("[TelegramHandler] Failed to bind request", logger.Field("error", err.Error()))
			c.Abort()
			result.HttpResult(c, nil, err)
			return
		}
		l := telegram.NewTelegramLogic(c, svcCtx)
		l.TelegramLogic(&request)
	}
}
