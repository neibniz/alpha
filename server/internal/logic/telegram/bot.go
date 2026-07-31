package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/pkg/logger"

	"github.com/neibniz/alpha/server/internal/model/auth"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const (
	adminTelegramCacheKey = "telegram:admin_chat_ids"
	adminTelegramCacheTTL = 5 * time.Minute
)

var telegramHTTPClient = &http.Client{Timeout: 10 * time.Second}

func GetTelegramConfig(ctx context.Context, svcCtx *svc.ServiceContext) (*types.TelegramConfig, error) {

	data, err := svcCtx.AuthModel.FindOneByMethod(ctx, "telegram")
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get Telegram config failed: %v", err.Error())
	}
	var telegramConfig auth.TelegramAuthConfig
	err = json.Unmarshal([]byte(data.Config), &telegramConfig)
	if err != nil {
		logger.WithContext(ctx).Error("unmarshal telegram config failed", logger.Field("error", err.Error()))
		return nil, err
	}

	enabled := data.Enabled != nil && *data.Enabled
	return &types.TelegramConfig{
		TelegramBotToken:      telegramConfig.BotToken,
		TelegramNotify:        enabled,
		TelegramWebHookDomain: telegramConfig.WebHookDomain,
	}, nil
}

func ApiLink(ctx *gin.Context, svcCtx *svc.ServiceContext, method string) (string, error) {
	cfg, err := GetTelegramConfig(ctx, svcCtx)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(cfg.TelegramBotToken) == "" {
		return "", errors.New("telegram bot token is unavailable")
	}
	return "https://api.telegram.org/bot" + cfg.TelegramBotToken + "/" + method, nil
}

func SendUserMessage(ctx *gin.Context, svcCtx *svc.ServiceContext, u user.User, text string, parseMode string) {
	apiLink, err := ApiLink(ctx, svcCtx, "sendMessage")
	if err != nil {
		logger.WithContext(ctx).Error("build telegram API request failed", logger.Field("error", err.Error()))
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiLink, nil)
	if err != nil {
		return
	}
	q := req.URL.Query()

	userTelegramChatId, ok := findTelegram(&u)
	if !ok {
		return
	}
	q.Add("chat_id", strconv.FormatInt(userTelegramChatId, 10))
	if parseMode == "markdown" {
		text = strings.ReplaceAll(text, "_", "\\_")
	}
	q.Add("text", text)
	q.Add("parse_mode", parseMode)
	req.URL.RawQuery = q.Encode()
	resp, err := telegramHTTPClient.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()

}

func SendAdminMessage(ctx *gin.Context, svcCtx *svc.ServiceContext, text string, parseMode string) {
	var adminTelegram []int64
	f := false
	adminTelegramJson, err := svcCtx.Redis.Get(ctx, adminTelegramCacheKey).Result()
	if err == nil {
		err = json.Unmarshal([]byte(adminTelegramJson), &adminTelegram)
		if err == nil {
			f = true
		}
	}
	if !f {
		svcCtx.DB.Model(&user.User{}).Where("is_admin = true AND telegram > 0").Pluck("telegram", &adminTelegram)
		val, err := json.Marshal(adminTelegram)
		if err != nil {
			logger.WithContext(ctx).Error("marshal admin telegram chat ids failed", logger.Field("error", err.Error()))
		} else if err := svcCtx.Redis.Set(ctx, adminTelegramCacheKey, string(val), adminTelegramCacheTTL).Err(); err != nil {
			logger.WithContext(ctx).Error("cache admin telegram chat ids failed", logger.Field("error", err.Error()))
		}
	}
	apiLink, err := ApiLink(ctx, svcCtx, "sendMessage")
	if err != nil {
		logger.WithContext(ctx).Error("build telegram API request failed", logger.Field("error", err.Error()))
		return
	}
	message := text
	if parseMode == "markdown" {
		message = strings.ReplaceAll(message, "_", "\\_")
	}
	for _, telegram := range adminTelegram {
		if telegram <= 0 {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiLink, nil)
		if err != nil {
			continue
		}
		q := req.URL.Query()
		q.Add("text", message)
		q.Add("parse_mode", parseMode)
		q.Add("chat_id", strconv.FormatInt(telegram, 10))
		req.URL.RawQuery = q.Encode()
		resp, err := telegramHTTPClient.Do(req)
		if err != nil {
			logger.WithContext(ctx).Error("send admin telegram message failed", logger.Field("error", err.Error()))
			continue
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
	}
}

func SetWebhook(ctx *gin.Context, svcCtx *svc.ServiceContext) error {
	configs, _ := svcCtx.SystemModel.GetSiteConfig(ctx)
	cfg := &types.SiteConfig{}
	tool.SystemConfigSliceReflectToStruct(configs, cfg)
	apiLink, err := ApiLink(ctx, svcCtx, "setWebhook")
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiLink, nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("url", cfg.Host+"/telegram/webhook")
	req.URL.RawQuery = q.Encode()
	resp, err := telegramHTTPClient.Do(req)
	if err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "set webhook error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("telegram set webhook returned status %d", resp.StatusCode)
	}
	return nil
}

func findTelegram(u *user.User) (int64, bool) {
	for _, item := range u.AuthMethods {
		if item.AuthType == "telegram" {
			// string to int64
			parseInt, err := strconv.ParseInt(item.AuthIdentifier, 10, 64)
			if err != nil {
				return 0, false
			}
			return parseInt, true
		}

	}
	return 0, false
}
