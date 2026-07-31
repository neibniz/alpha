package emailLogic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"text/template"
	"time"

	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/email"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/queue/types"
	"github.com/hibiken/asynq"
)

type emailSenderFactory func(platform, config, siteName string) (email.Sender, error)

type emailTemplateKey struct {
	typeName string
	digest   [sha256.Size]byte
}

type SendEmailLogic struct {
	svcCtx *svc.ServiceContext

	senderMu      sync.Mutex
	senderKey     [sha256.Size]byte
	sender        email.Sender
	senderFactory emailSenderFactory

	templateMu sync.RWMutex
	templates  map[emailTemplateKey]*template.Template
}

func NewSendEmailLogic(svcCtx *svc.ServiceContext) *SendEmailLogic {
	return &SendEmailLogic{
		svcCtx:        svcCtx,
		senderFactory: email.NewSender,
		templates:     make(map[emailTemplateKey]*template.Template),
	}
}

func (l *SendEmailLogic) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload types.SendEmailPayload
	if err := types.DecodePrivatePayload(l.svcCtx.Privacy, types.EmailPayloadScope, task.Payload(), &payload); err != nil {
		logger.WithContext(ctx).Error("[SendEmailLogic] decode payload failed", logger.Field("error_type", fmt.Sprintf("%T", err)))
		return asynq.SkipRetry
	}
	message := &log.Message{
		Platform: l.svcCtx.ConfigSnapshot().Email.Platform,
		To:       privacy.MaskIdentifier(payload.Email),
		Subject:  payload.Subject,
		Content:  []byte(`"***"`),
		Template: payload.Type,
		Status:   2,
	}

	content, err := l.renderContent(payload)
	if err != nil {
		logger.WithContext(ctx).Error("[SendEmailLogic] render content failed",
			logger.Field("type", payload.Type), logger.Field("error_type", fmt.Sprintf("%T", err)))
		l.writeMessageLog(ctx, message)
		return asynq.SkipRetry
	}
	sender, err := l.getSender()
	if err != nil {
		logger.WithContext(ctx).Error("[SendEmailLogic] create sender failed", logger.Field("error_type", fmt.Sprintf("%T", err)))
		l.writeMessageLog(ctx, message)
		return asynq.SkipRetry
	}
	if err := sender.Send([]string{payload.Email}, payload.Subject, content); err != nil {
		logger.WithContext(ctx).Error("[SendEmailLogic] delivery failed",
			logger.Field("recipient", payload.Email), logger.Field("error_type", fmt.Sprintf("%T", err)))
		l.writeMessageLog(ctx, message)
		return fmt.Errorf("email delivery failed (%T)", err)
	}

	message.Status = 1
	l.writeMessageLog(ctx, message)
	return nil
}

func (l *SendEmailLogic) getSender() (email.Sender, error) {
	runtime := l.svcCtx.ConfigSnapshot()
	cfg := runtime.Email
	key := sha256.Sum256([]byte(cfg.Platform + "\x00" + cfg.PlatformConfig + "\x00" + runtime.Site.SiteName))
	l.senderMu.Lock()
	defer l.senderMu.Unlock()
	if l.sender != nil && l.senderKey == key {
		return l.sender, nil
	}
	sender, err := l.senderFactory(cfg.Platform, cfg.PlatformConfig, runtime.Site.SiteName)
	if err != nil {
		return nil, err
	}
	l.sender = sender
	l.senderKey = key
	return sender, nil
}

func (l *SendEmailLogic) renderContent(payload types.SendEmailPayload) (string, error) {
	emailConfig := l.svcCtx.ConfigSnapshot().Email
	switch payload.Type {
	case types.EmailTypeVerify:
		data, err := types.DecodeEmailContent[types.VerifyEmailContent](payload.Content)
		if err != nil {
			return "", err
		}
		return l.executeTemplate("verify", emailConfig.VerifyEmailTemplate, data)
	case types.EmailTypeMaintenance:
		data, err := types.DecodeEmailContent[types.SiteEmailContent](payload.Content)
		if err != nil {
			return "", err
		}
		return l.executeTemplate("maintenance", emailConfig.MaintenanceEmailTemplate, data)
	case types.EmailTypeExpiration:
		data, err := types.DecodeEmailContent[types.ExpirationEmailContent](payload.Content)
		if err != nil {
			return "", err
		}
		return l.executeTemplate("expiration", emailConfig.ExpirationEmailTemplate, data)
	case types.EmailTypeTrafficExceed:
		data, err := types.DecodeEmailContent[types.SiteEmailContent](payload.Content)
		if err != nil {
			return "", err
		}
		return l.executeTemplate("traffic_exceed", emailConfig.TrafficExceedEmailTemplate, data)
	case types.EmailTypeCustom:
		data, err := types.DecodeEmailContent[types.CustomEmailContent](payload.Content)
		if err != nil {
			return "", err
		}
		if data.Content == "" {
			return "", fmt.Errorf("custom email content is empty")
		}
		return data.Content, nil
	default:
		return "", fmt.Errorf("unsupported email type %v", payload.Type)
	}
}

func (l *SendEmailLogic) executeTemplate(typeName, source string, data any) (string, error) {
	key := emailTemplateKey{typeName: typeName, digest: sha256.Sum256([]byte(source))}
	l.templateMu.RLock()
	tpl := l.templates[key]
	l.templateMu.RUnlock()
	if tpl == nil {
		parsed, err := template.New(typeName).Parse(source)
		if err != nil {
			return "", err
		}
		l.templateMu.Lock()
		if existing := l.templates[key]; existing != nil {
			tpl = existing
		} else {
			l.templates[key] = parsed
			tpl = parsed
		}
		l.templateMu.Unlock()
	}
	var result bytes.Buffer
	if err := tpl.Execute(&result, data); err != nil {
		return "", err
	}
	return result.String(), nil
}

func (l *SendEmailLogic) writeMessageLog(ctx context.Context, message *log.Message) {
	if l.svcCtx.LogModel == nil || message == nil {
		return
	}
	content, err := message.Marshal()
	if err != nil {
		logger.WithContext(ctx).Error("[SendEmailLogic] marshal audit log failed", logger.Field("error_type", fmt.Sprintf("%T", err)))
		return
	}
	if err := l.svcCtx.LogModel.Insert(ctx, &log.SystemLog{
		Type:    log.TypeEmailMessage.Uint8(),
		Date:    time.Now().Format(time.DateOnly),
		Content: string(content),
	}); err != nil {
		logger.WithContext(ctx).Error("[SendEmailLogic] insert audit log failed", logger.Field("error_type", fmt.Sprintf("%T", err)))
	}
}
