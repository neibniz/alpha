package svc

import (
	"sync"
	"testing"

	"github.com/neibniz/alpha/server/internal/config"
)

func TestRuntimeValuesAreSafeForConcurrentAccess(t *testing.T) {
	service := &ServiceContext{}
	var wg sync.WaitGroup
	for index := 0; index < 64; index++ {
		wg.Add(1)
		go func(value int) {
			defer wg.Done()
			service.SetExchangeRate(float64(value) / 10)
			_ = service.CurrentExchangeRate()
			service.SetLogConfig(config.Log{AutoClear: value%2 == 0, ClearDays: int64(value + 1)})
			_ = service.LogConfig()
			service.SetTelegramRuntime(config.Telegram{BotID: int64(value)}, nil)
			_, _ = service.TelegramRuntime()
			service.SetSiteConfig(config.SiteConfig{SiteName: "alpha"})
			service.SetSubscribeConfig(config.SubscribeConfig{SubscribePath: "/alpha"})
			service.SetNodeConfig(config.NodeConfig{NodePushInterval: int64(value)})
			_ = service.ConfigSnapshot()
		}(index)
	}
	wg.Wait()
}
