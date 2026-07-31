package initialize

import (
	"context"

	"github.com/neibniz/alpha/server/pkg/logger"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/tool"
)

type verifyConfig struct {
	TurnstileSiteKey          string
	TurnstileSecret           string
	EnableLoginVerify         bool
	EnableRegisterVerify      bool
	EnableResetPasswordVerify bool
}

func Verify(svc *svc.ServiceContext) {
	logger.Debug("Verify config initialization")
	configs, err := svc.SystemModel.GetVerifyConfig(context.Background())
	if err != nil {
		logger.Error("[Init Verify Config] Get Verify Config Error: ", logger.Field("error", err.Error()))
		return
	}
	var verify verifyConfig
	tool.SystemConfigSliceReflectToStruct(configs, &verify)
	svc.SetVerifyConfig(config.Verify{
		TurnstileSiteKey:    verify.TurnstileSiteKey,
		TurnstileSecret:     verify.TurnstileSecret,
		LoginVerify:         verify.EnableLoginVerify,
		RegisterVerify:      verify.EnableRegisterVerify,
		ResetPasswordVerify: verify.EnableResetPasswordVerify,
	})

	logger.Debug("Verify code config initialization")

	var verifyCodeConfig config.VerifyCode
	cfg, err := svc.SystemModel.GetVerifyCodeConfig(context.Background())
	if err != nil {
		logger.Errorf("[Init Verify Config] Get Verify Code Config Error: %s", err.Error())
		return
	}
	tool.SystemConfigSliceReflectToStruct(cfg, &verifyCodeConfig)
	svc.SetVerifyCodeConfig(verifyCodeConfig)
}
