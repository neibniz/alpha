package initialize

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/model/auth"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
)

// Email get email smtp config
func Email(ctx *svc.ServiceContext) {
	logger.Debug("Email config initialization")
	method, err := ctx.AuthModel.FindOneByMethod(context.Background(), "email")
	if err != nil {
		panic(fmt.Sprintf("[Error] Initialization Failed to find email auth method: %v", err.Error()))
	}
	var cfg config.EmailConfig
	var emailConfig = new(auth.EmailAuthConfig)
	emailConfig.Unmarshal(method.Config)
	tool.DeepCopy(&cfg, emailConfig)
	cfg.Enable = *method.Enabled
	value, _ := json.Marshal(emailConfig.PlatformConfig)
	cfg.PlatformConfig = string(value)
	ctx.SetEmailConfig(cfg)
}
