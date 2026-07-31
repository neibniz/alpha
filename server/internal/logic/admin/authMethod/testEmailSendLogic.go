package authMethod

import (
	"context"
	"fmt"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/email"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type TestEmailSendLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Test email send
func NewTestEmailSendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TestEmailSendLogic {
	return &TestEmailSendLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TestEmailSendLogic) TestEmailSend(req *types.TestEmailSendRequest) error {
	runtime := l.svcCtx.ConfigSnapshot()
	client, err := email.NewSender(runtime.Email.Platform, runtime.Email.PlatformConfig, runtime.Site.SiteName)
	if err != nil {
		l.Errorw("new email sender err", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "new email sender err: %v", err.Error())
	}
	err = client.Send([]string{req.Email}, "Test Email Send", "this a test email send by alpha")
	if err != nil {
		return errors.Wrapf(xerr.NewErrCodeMsg(500, fmt.Sprintf("send email err: %v", err.Error())), "send email err: %v", err.Error())
	}
	return nil
}
