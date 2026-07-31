package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/gin-gonic/gin"
)

func TestTelegramHandlerRequiresRuntimeHeaderSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &svc.ServiceContext{}
	service.SetTelegramRuntime(config.Telegram{
		Enable:        true,
		BotToken:      "bot-token",
		WebhookSecret: "webhook-secret",
	}, nil)
	router := gin.New()
	router.POST("/v1/telegram/webhook", TelegramHandler(service))

	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodPost, "/v1/telegram/webhook?secret=webhook-secret", strings.NewReader(`{}`)),
		httptest.NewRequest(http.MethodPost, "/v1/telegram/webhook", strings.NewReader(`{}`)),
	} {
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated telegram response = %d, want 401", response.Code)
		}
	}
}

func TestTelegramHandlerStopsAfterInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &svc.ServiceContext{}
	service.SetTelegramRuntime(config.Telegram{
		Enable:        true,
		BotToken:      "bot-token",
		WebhookSecret: "webhook-secret",
	}, nil)
	router := gin.New()
	router.POST("/v1/telegram/webhook", TelegramHandler(service))

	request := httptest.NewRequest(http.MethodPost, "/v1/telegram/webhook", strings.NewReader(`{"update_id":`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Telegram-Bot-Api-Secret-Token", "webhook-secret")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code < http.StatusBadRequest || response.Code >= http.StatusInternalServerError {
		t.Fatalf("invalid telegram payload response = %d, want 4xx", response.Code)
	}
}
