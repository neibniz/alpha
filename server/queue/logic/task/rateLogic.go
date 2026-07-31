package task

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/exchangeRate"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/hibiken/asynq"
)

type RateLogic struct {
	svcCtx *svc.ServiceContext
}

func NewRateLogic(svcCtx *svc.ServiceContext) *RateLogic {
	return &RateLogic{
		svcCtx: svcCtx,
	}
}

func (l *RateLogic) ProcessTask(ctx context.Context, _ *asynq.Task) error {
	// Retrieve system currency configuration
	currency, err := l.svcCtx.SystemModel.GetCurrencyConfig(ctx)
	if err != nil {
		logger.Errorw("[PurchaseCheckout] GetCurrencyConfig error", logger.Field("error", err.Error()))
		return err
	}
	// Parse currency configuration
	configs := struct {
		CurrencyUnit   string
		CurrencySymbol string
		AccessKey      string
	}{}
	tool.SystemConfigSliceReflectToStruct(currency, &configs)

	// Skip conversion if no exchange rate API key configured
	if configs.AccessKey == "" {
		logger.Debugf("[RateLogic] skip exchange rate, no access key configured")
		return nil
	}
	// Update exchange rates
	result, err := exchangeRate.GetExchangeRete(configs.CurrencyUnit, "CNY", configs.AccessKey, 1)
	if err != nil {
		logger.Errorw("[RateLogic] GetExchangeRete error", logger.Field("error", err.Error()))
		return err
	}
	if !l.svcCtx.SetExchangeRateForCurrency(configs.CurrencyUnit, result) {
		logger.WithContext(ctx).Info("[RateLogic] discarded stale exchange rate after currency configuration changed")
		return nil
	}
	logger.WithContext(ctx).Infof("[RateLogic] GetExchangeRete success, result: %+v", result)
	return nil
}
