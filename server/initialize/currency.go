package initialize

import (
	"context"
	"fmt"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
)

func Currency(ctx *svc.ServiceContext) {
	// Retrieve system currency configuration
	currency, err := ctx.SystemModel.GetCurrencyConfig(context.Background())
	if err != nil {
		logger.Errorf("[INIT] Failed to get currency configuration: %v", err.Error())
		panic(fmt.Sprintf("[INIT] Failed to get currency configuration: %v", err.Error()))
	}
	// Parse currency configuration
	configs := struct {
		CurrencyUnit   string
		CurrencySymbol string
		AccessKey      string
	}{}
	tool.SystemConfigSliceReflectToStruct(currency, &configs)
	currencyConfig := config.Currency{
		Unit:      configs.CurrencyUnit,
		Symbol:    configs.CurrencySymbol,
		AccessKey: configs.AccessKey,
	}
	ctx.SetCurrencyRuntime(currencyConfig, 0)
	logger.Infof("[INIT] Currency configuration: %v", currencyConfig)
}
