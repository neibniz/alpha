package emailLogic

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/neibniz/alpha/server/internal/config"
	modelLog "github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/email"
	"github.com/neibniz/alpha/server/queue/types"
	"github.com/hibiken/asynq"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSendEmailLogicRetriesDeliveryFailureAndWritesFailedAudit(t *testing.T) {
	wantErr := errors.New("provider failure")
	sender := &sendEmailTestSender{err: wantErr}
	logic, db := newSendEmailTestLogic(t, sender)
	task := sendEmailTestTask(t, logic.svcCtx.Privacy)

	err := logic.ProcessTask(context.Background(), task)
	if err == nil || errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("ProcessTask() error = %v, want retryable delivery error", err)
	}
	message := lastEmailAuditMessage(t, db)
	if message.Status != 2 || message.To == "user@example.com" {
		t.Fatalf("failed email audit = %#v", message)
	}
}

func TestSendEmailLogicCachesSenderAndCompiledTemplate(t *testing.T) {
	sender := &sendEmailTestSender{}
	logic, _ := newSendEmailTestLogic(t, sender)
	task := sendEmailTestTask(t, logic.svcCtx.Privacy)
	for range 2 {
		if err := logic.ProcessTask(context.Background(), task); err != nil {
			t.Fatal(err)
		}
	}
	if got := sender.calls.Load(); got != 2 {
		t.Fatalf("sender calls = %d, want 2", got)
	}
	if got := logicFactoryCalls.Load(); got != 1 {
		t.Fatalf("sender factory calls = %d, want 1", got)
	}
	logic.templateMu.RLock()
	templateCount := len(logic.templates)
	logic.templateMu.RUnlock()
	if templateCount != 1 {
		t.Fatalf("compiled template count = %d, want 1", templateCount)
	}
}

func TestSendEmailLogicRejectsInvalidTemplateWithoutPanic(t *testing.T) {
	logic, _ := newSendEmailTestLogic(t, &sendEmailTestSender{})
	logic.svcCtx.Config.Email.VerifyEmailTemplate = "{{"
	payload := sendEmailTestPayload(t)
	if _, err := logic.renderContent(payload); err == nil {
		t.Fatal("expected invalid template error")
	}
}

var logicFactoryCalls atomic.Int32

func newSendEmailTestLogic(t *testing.T, sender email.Sender) (*SendEmailLogic, *gorm.DB) {
	t.Helper()
	logicFactoryCalls.Store(0)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelLog.SystemLog{}); err != nil {
		t.Fatal(err)
	}
	codec := privacy.DisabledCodec()
	serviceContext := &svc.ServiceContext{
		Privacy:  codec,
		LogModel: modelLog.NewModel(db),
		Config: config.Config{
			Email: config.EmailConfig{
				Platform:            "test",
				PlatformConfig:      `{"key":"value"}`,
				VerifyEmailTemplate: "code={{.Code}}",
			},
		},
	}
	logic := NewSendEmailLogic(serviceContext)
	logic.senderFactory = func(string, string, string) (email.Sender, error) {
		logicFactoryCalls.Add(1)
		return sender, nil
	}
	return logic, db
}

func sendEmailTestTask(t *testing.T, codec *privacy.Codec) *asynq.Task {
	t.Helper()
	payload := sendEmailTestPayload(t)
	encoded, err := types.EncodePrivatePayload(codec, types.EmailPayloadScope, payload)
	if err != nil {
		t.Fatal(err)
	}
	return asynq.NewTask(types.ForthwithSendEmail, encoded)
}

func sendEmailTestPayload(t *testing.T) types.SendEmailPayload {
	t.Helper()
	content, err := types.EncodeEmailContent(types.VerifyEmailContent{Code: "123456"})
	if err != nil {
		t.Fatal(err)
	}
	return types.SendEmailPayload{
		Type:    types.EmailTypeVerify,
		Email:   "user@example.com",
		Subject: "Verification",
		Content: content,
	}
}

func lastEmailAuditMessage(t *testing.T, db *gorm.DB) modelLog.Message {
	t.Helper()
	var row modelLog.SystemLog
	if err := db.Order("id DESC").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	var message modelLog.Message
	if err := message.Unmarshal([]byte(row.Content)); err != nil {
		t.Fatal(err)
	}
	return message
}

type sendEmailTestSender struct {
	calls atomic.Int32
	err   error
}

func (s *sendEmailTestSender) Send([]string, string, string) error {
	s.calls.Add(1)
	return s.err
}
