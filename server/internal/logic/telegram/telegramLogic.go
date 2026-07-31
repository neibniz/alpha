package telegram

import (
	"context"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type TelegramLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTelegramLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TelegramLogic {
	return &TelegramLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TelegramLogic) TelegramLogic(req *tgbotapi.Update) {
	if req.Message != nil && req.Message.Text != "" {
		switch req.Message.Command() {
		case "traffic":
			if err := l.traffic(req.Message.Chat.ID); err != nil {
				l.Logger.Error("[TelegramLogic] Traffic Error: ", logger.Field("error", err.Error()), logger.Field("command", req.Message.Command()), logger.Field("chat_id", privacy.MaskIdentifier(strconv.FormatInt(req.Message.Chat.ID, 10))))
			}
		case "bind":
			if err := l.bind(req.Message.Chat.ID, req.Message.CommandArguments()); err != nil {
				l.Logger.Error("[TelegramLogic] Bind Error: ", logger.Field("error", err.Error()), logger.Field("command", req.Message.Command()), logger.Field("chat_id", privacy.MaskIdentifier(strconv.FormatInt(req.Message.Chat.ID, 10))))
			}
		case "start":
			if err := l.start(req); err != nil {
				l.Logger.Error("[TelegramLogic] Start Error: ", logger.Field("error", err.Error()), logger.Field("command", req.Message.Command()), logger.Field("chat_id", privacy.MaskIdentifier(strconv.FormatInt(req.Message.Chat.ID, 10))))
			}
		}
	} else {
		l.Logger.Error("[TelegramLogic] Message is empty")
	}
}

func (l *TelegramLogic) sendMessage(bot *tgbotapi.BotAPI, message string, userId int64) error {
	if bot == nil {
		return errors.New("telegram bot is unavailable")
	}
	msg := tgbotapi.NewMessage(userId, message)
	msg.ParseMode = "Markdown"
	_, err := bot.Send(msg)
	return err
}

func (l *TelegramLogic) traffic(userId int64) error {
	return nil
}

func (l *TelegramLogic) bind(userId int64, token string) error {
	return nil
}

func (l *TelegramLogic) start(req *tgbotapi.Update) error {
	bot := l.svcCtx.TelegramBotClient()
	token := strings.TrimSpace(req.Message.CommandArguments())
	if len(token) != 64 {
		return l.sendMessage(bot, "Please bind account!", req.Message.Chat.ID)
	} else {
		if _, err := hex.DecodeString(token); err != nil {
			return l.sendMessage(bot, "Bind failed!", req.Message.Chat.ID)
		}
		value, err := l.svcCtx.Redis.GetDel(l.ctx, config.TelegramBindCacheKey(token)).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			l.Errorw("TelegramLogic start Redis Get Error: ", logger.Field("error", err.Error()))
			return l.sendMessage(bot, "Bind failed!", req.Message.Chat.ID)
		}
		if value == "" {
			return l.sendMessage(bot, "Bind failed!", req.Message.Chat.ID)
		}
		userId, err := strconv.ParseInt(value, 10, 64)
		if err != nil || userId <= 0 {
			l.Errorw("TelegramLogic start ParseInt Error: ", logger.Field("error", err))
			return l.sendMessage(bot, "Bind failed!", req.Message.Chat.ID)
		}
		chatID := strconv.FormatInt(req.Message.Chat.ID, 10)
		boundMethod, boundErr := l.svcCtx.UserModel.FindUserAuthMethodByOpenID(l.ctx, "telegram", chatID)
		if boundErr == nil && boundMethod.UserId != userId {
			return l.sendMessage(bot, "Bind failed!", req.Message.Chat.ID)
		}
		if boundErr != nil && !errors.Is(boundErr, gorm.ErrRecordNotFound) {
			return l.sendMessage(bot, "Bind failed!", req.Message.Chat.ID)
		}

		method, err := l.svcCtx.UserModel.FindUserAuthMethodByPlatform(l.ctx, userId, "telegram")
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			l.Errorw("TelegramLogic start FindUserAuthMethodByPlatform Error: ", logger.Field("error", err.Error()), logger.Field("userId", userId))
			return l.sendMessage(bot, "Bind failed!", req.Message.Chat.ID)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := l.svcCtx.UserModel.InsertUserAuthMethods(l.ctx, &user.AuthMethods{
				UserId:         userId,
				AuthType:       "telegram",
				AuthIdentifier: chatID,
				Verified:       true,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}); err != nil {
				l.Errorw("TelegramLogic start InsertUserAuthMethod Error: ", logger.Field("error", err.Error()), logger.Field("userId", userId))
				return l.sendMessage(bot, "Bind failed!", req.Message.Chat.ID)
			}
		} else {
			if err := l.svcCtx.UserModel.UpdateUserAuthMethods(l.ctx, method.UserId, method.AuthType, user.UserAuthMethodFieldsUpdate{
				AuthIdentifier: user.UpdateValue(chatID),
				Verified:       user.UpdateValue(true),
			}); err != nil {
				l.Errorw("TelegramLogic start UpdateUserAuthMethod Error: ", logger.Field("error", err.Error()), logger.Field("userId", userId))
				return l.sendMessage(bot, "Bind failed!", req.Message.Chat.ID)
			}
		}

		text, err := tool.RenderTemplateToString(BindNotify, map[string]string{
			"Id":   strconv.FormatInt(userId, 10),
			"Time": time.Now().Format("2006-01-02 15:04:05"),
		})
		if err != nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "render template failed")
		}
		return l.sendMessage(bot, text, req.Message.Chat.ID)
	}
}
