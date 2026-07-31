package initialize

import (
	"context"
	"path"

	"github.com/neibniz/alpha/server/pkg/logger"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/logic/telegram"
	"github.com/neibniz/alpha/server/internal/model/auth"
	"github.com/neibniz/alpha/server/internal/oauthcallback"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/random"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func Telegram(svc *svc.ServiceContext) {

	method, err := svc.AuthModel.FindOneByMethod(context.Background(), "telegram")
	if err != nil {
		logger.Errorf("[Init Telegram Config] Get Telegram Config Error: %s", err.Error())
		return
	}
	tgConfig := new(auth.TelegramAuthConfig)
	if err = tgConfig.Unmarshal(method.Config); err != nil {
		logger.Errorf("[Init Telegram Config] Unmarshal Telegram Config Error: %s", err.Error())
		return
	}
	enabled := method.Enabled != nil && *method.Enabled
	runtimeConfig := config.Telegram{
		Enable:        enabled,
		BotToken:      tgConfig.BotToken,
		EnableNotify:  enabled,
		WebHookDomain: tgConfig.WebHookDomain,
	}

	if !enabled || tgConfig.BotToken == "" {
		svc.SetTelegramRuntime(runtimeConfig, nil)
		logger.Debug("[Init Telegram Config] Telegram Token is empty")
		return
	}

	bot, err := tgbotapi.NewBotAPI(tgConfig.BotToken)
	if err != nil {
		logger.Error("[Init Telegram Config] New Bot API Error: ", logger.Field("error", err.Error()))
		return
	}

	if tgConfig.WebHookDomain == "" || svc.Config.Debug {
		// set Long Polling mode
		updateConfig := tgbotapi.NewUpdate(0)
		updateConfig.Timeout = 60
		updates := bot.GetUpdatesChan(updateConfig)
		go func() {
			for update := range updates {
				if update.Message != nil {
					ctx := context.Background()
					l := telegram.NewTelegramLogic(ctx, svc)
					l.TelegramLogic(&update)
				}
			}
		}()
	} else {
		webhookBase, err := oauthcallback.ValidateBaseURL(tgConfig.WebHookDomain)
		if err != nil {
			logger.Errorf("[Init Telegram Config] Invalid webhook domain: %s", err.Error())
			return
		}
		webhookBase.Path = path.Join(webhookBase.Path, "/v1/telegram/webhook")
		webhookSecret, err := random.SecureKey(48, 1)
		if err != nil {
			logger.Errorf("[Init Telegram Config] Generate webhook secret error: %s", err.Error())
			return
		}
		runtimeConfig.WebhookSecret = webhookSecret
		// Publish the secret before Telegram can deliver the first update.
		svc.SetTelegramRuntime(runtimeConfig, bot)
		_, err = bot.MakeRequest("setWebhook", tgbotapi.Params{
			"url":          webhookBase.String(),
			"secret_token": webhookSecret,
		})
		if err != nil {
			runtimeConfig.WebhookSecret = ""
			svc.SetTelegramRuntime(runtimeConfig, nil)
			logger.Errorf("[Init Telegram Config] Request Webhook Error: %s", err.Error())
			return
		}
	}

	user, err := bot.GetMe()
	if err != nil {
		logger.Error("[Init Telegram Config] Get Bot Info Error: ", logger.Field("error", err.Error()))
		return
	}
	runtimeConfig.BotID = user.ID
	runtimeConfig.BotName = user.UserName
	svc.SetTelegramRuntime(runtimeConfig, bot)

	logger.Info("[Init Telegram Config] Webhook set success")
}
