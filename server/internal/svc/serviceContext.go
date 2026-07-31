package svc

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neibniz/alpha/server/internal/accesspolicy"
	"github.com/neibniz/alpha/server/internal/analytics"
	"github.com/neibniz/alpha/server/internal/model/client"
	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/pkg/device"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/model/ads"
	"github.com/neibniz/alpha/server/internal/model/announcement"
	"github.com/neibniz/alpha/server/internal/model/auth"
	"github.com/neibniz/alpha/server/internal/model/coupon"
	"github.com/neibniz/alpha/server/internal/model/document"
	"github.com/neibniz/alpha/server/internal/model/forwarder"
	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/model/system"
	"github.com/neibniz/alpha/server/internal/model/ticket"
	"github.com/neibniz/alpha/server/internal/model/traffic"
	"github.com/neibniz/alpha/server/internal/model/user"
	privacycodec "github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/pkg/limit"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/nodeMultiplier"
	"github.com/neibniz/alpha/server/pkg/orm"
	"github.com/neibniz/alpha/server/pkg/syncx"
	"github.com/neibniz/alpha/server/pkg/tool"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type ServiceContext struct {
	DB              *gorm.DB
	Redis           *redis.Client
	Config          config.Config
	Queue           *asynq.Client
	Analytics       analytics.Store
	Privacy         *privacycodec.Codec
	AccessIPLimiter *accesspolicy.Store
	GeoIP           *IPLocation
	exchangeRate    atomic.Uint64
	runtimeMu       sync.RWMutex

	//NodeCache   *cache.NodeCacheClient
	AuthModel   auth.Model
	AdsModel    ads.Model
	LogModel    log.Model
	NodeModel   node.Model
	UserModel   user.Model
	OrderModel  order.Model
	ClientModel client.Model
	TicketModel ticket.Model
	//ServerModel        server.Model
	SystemModel       system.Model
	CouponModel       coupon.Model
	PaymentModel      payment.Model
	DocumentModel     document.Model
	ForwarderStore    *forwarder.Store
	SubscribeModel    subscribe.Model
	TrafficLogModel   traffic.Model
	AnnouncementModel announcement.Model

	Restart               func() error
	telegramBot           *tgbotapi.BotAPI
	nodeMultiplierManager atomic.Pointer[nodeMultiplier.Manager]
	AuthLimiter           *limit.PeriodLimit
	DeviceManager         *device.DeviceManager
	NodeRealtime          *realtime.Manager
	ForwarderRealtime     *realtime.ForwarderManager
	DashboardRealtime     *realtime.DashboardManager
	XrayUsersSingleFlight syncx.SingleFlight
}

var defaultNodeMultiplierManager = nodeMultiplier.NewNodeMultiplierManager(nil)

func NewServiceContext(c config.Config) *ServiceContext {
	// gorm initialize
	db, err := orm.ConnectPostgres(orm.Postgres{
		Config: c.Postgres,
	})

	if err != nil {
		panic(err.Error())
	}
	privacyCodec, err := privacycodec.LoadCodec(c.Privacy.Enabled, c.Privacy.KeyFile)
	if err != nil {
		panic(fmt.Sprintf("initialize privacy codec: %v", err))
	}
	analyticsStore, analyticsErr := analytics.NewClickHouseStore(context.Background(), c.ClickHouse, privacyCodec)
	if analyticsErr != nil {
		if c.ClickHouse.Enabled && privacyCodec.Enabled() {
			panic(fmt.Sprintf("initialize encrypted ClickHouse analytics storage: %v", analyticsErr))
		}
		logger.Errorf("[ClickHouse] initialize analytics storage failed: %v", analyticsErr)
		analyticsStore = analytics.NewUnavailableStore()
	}

	var geoIP *IPLocation
	if c.GeoIP.Enable {
		geoIPPath := c.GeoIP.DatabasePath
		if geoIPPath == "" {
			geoIPPath = "/usr/share/alpha/geoip/dbip-city-lite.mmdb"
		}
		geoIP, err = NewIPLocation(geoIPPath)
		if err != nil {
			panic(err.Error())
		}
	}

	rds := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Host,
		Password: c.Redis.Pass,
		DB:       c.Redis.DB,
	})
	err = rds.Ping(context.Background()).Err()
	if err != nil {
		panic(err.Error())
	}
	if err = tool.RedisClientWritableCheck(context.Background(), rds); err != nil {
		panic(fmt.Sprintf("redis db %d is not writable: %v", c.Redis.DB, err))
	}
	if c.Redis.DB != AsynqRedisDB {
		if err = tool.RedisWritableCheck(c.Redis.Host, c.Redis.Pass, AsynqRedisDB); err != nil {
			panic(fmt.Sprintf("redis db %d for asynq is not writable: %v", AsynqRedisDB, err))
		}
	}
	authLimiter := limit.NewPeriodLimit(86400, 15, rds, config.SendCountLimitKeyPrefix, limit.Align())
	dashboardRealtime := realtime.NewDashboardManager()
	nodeRealtime := realtime.NewManager()
	forwarderRealtime := realtime.NewForwarderManager(time.Duration(c.Forwarder.StatusTTL) * time.Second)
	var networkSampleMu sync.Mutex
	networkSamplePending := false
	nodeRealtime.SetChangeHook(func(realtime.NodeStatus) {
		networkSampleMu.Lock()
		if !networkSamplePending {
			networkSamplePending = true
			go func() {
				time.Sleep(time.Second)
				states := nodeRealtime.Snapshot(nil)
				now := time.Now()
				if err := realtime.RecordDashboardNetworkSnapshot(context.Background(), rds, states); err != nil {
					logger.Debugf("[DashboardRealtime] record network sample error: %v", err)
				}
				if err := realtime.RecordServerNetworkSnapshot(context.Background(), rds, now, states); err != nil {
					logger.Debugf("[ServerRealtime] record network samples error: %v", err)
				}
				networkSampleMu.Lock()
				networkSamplePending = false
				networkSampleMu.Unlock()
			}()
		}
		networkSampleMu.Unlock()
		dashboardRealtime.Publish(realtime.DashboardEventNodeStatus)
	})
	srv := &ServiceContext{
		DB:              db,
		Redis:           rds,
		Config:          c,
		Queue:           NewAsynqClient(c),
		Analytics:       analyticsStore,
		Privacy:         privacyCodec,
		AccessIPLimiter: accesspolicy.NewStore(rds, privacyCodec),
		GeoIP:           geoIP,
		//NodeCache:   cache.NewNodeCacheClient(rds),
		AuthLimiter: authLimiter,
		AdsModel:    ads.NewModel(db, rds),
		LogModel: log.NewModel(db, log.Hooks{
			LookupIPLocation: func(_ context.Context, ip string) log.IPLocation {
				if geoIP == nil {
					return log.IPLocation{}
				}
				info := geoIP.Lookup(ip)
				return log.IPLocation{
					CountryShort: info.CountryShort,
					Region:       info.Region,
					Location:     info.Location,
				}
			},
			SystemLogInserted: func(ctx context.Context, item *log.SystemLog) {
				switch log.Type(item.Type) {
				case log.TypeLogin, log.TypeRegister, log.TypeLogout:
					dashboardRealtime.Publish(realtime.DashboardEventActivity)
				}
			},
			SubscribeAccessLogInserted: func(context.Context, *log.SubscribeAccessLog) {
				dashboardRealtime.Publish(realtime.DashboardEventActivity)
			},
		}),
		NodeModel:   node.NewModel(db, rds),
		AuthModel:   auth.NewModel(db, rds),
		UserModel:   user.NewModel(db, rds, privacyCodec),
		OrderModel:  order.NewModel(db, rds),
		ClientModel: client.NewSubscribeApplicationModel(db),
		TicketModel: ticket.NewModel(db, rds),
		//ServerModel:       server.NewModel(db, rds),
		SystemModel:           system.NewModel(db, rds),
		CouponModel:           coupon.NewModel(db, rds),
		PaymentModel:          payment.NewModel(db, rds),
		DocumentModel:         document.NewModel(db, rds),
		ForwarderStore:        forwarder.NewStore(db),
		SubscribeModel:        subscribe.NewModel(db, rds),
		TrafficLogModel:       traffic.NewModel(db),
		AnnouncementModel:     announcement.NewModel(db, rds),
		NodeRealtime:          nodeRealtime,
		ForwarderRealtime:     forwarderRealtime,
		DashboardRealtime:     dashboardRealtime,
		XrayUsersSingleFlight: syncx.NewSingleFlight(),
	}
	srv.DeviceManager = NewDeviceManager(srv)
	return srv

}

func (s *ServiceContext) CurrentExchangeRate() float64 {
	if s == nil {
		return 0
	}
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	return math.Float64frombits(s.exchangeRate.Load())
}

func (s *ServiceContext) SetExchangeRate(rate float64) {
	if s == nil {
		return
	}
	s.runtimeMu.Lock()
	s.exchangeRate.Store(math.Float64bits(rate))
	s.runtimeMu.Unlock()
}

func (s *ServiceContext) CurrencyRuntime() (config.Currency, float64) {
	if s == nil {
		return config.Currency{}, 0
	}
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	return s.Config.Currency, math.Float64frombits(s.exchangeRate.Load())
}

func (s *ServiceContext) SetCurrencyRuntime(value config.Currency, rate float64) {
	if s == nil {
		return
	}
	s.runtimeMu.Lock()
	s.Config.Currency = value
	s.exchangeRate.Store(math.Float64bits(rate))
	s.runtimeMu.Unlock()
}

func (s *ServiceContext) SetExchangeRateForCurrency(unit string, rate float64) bool {
	if s == nil {
		return false
	}
	s.runtimeMu.Lock()
	defer s.runtimeMu.Unlock()
	if s.Config.Currency.Unit != unit {
		return false
	}
	s.exchangeRate.Store(math.Float64bits(rate))
	return true
}

func (s *ServiceContext) ConfigSnapshot() config.Config {
	if s == nil {
		return config.Config{}
	}
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	return s.Config
}

func (s *ServiceContext) SetSiteConfig(value config.SiteConfig) {
	s.setRuntimeConfig(func(runtime *config.Config) { runtime.Site = value })
}

func (s *ServiceContext) SetSubscribeConfig(value config.SubscribeConfig) {
	s.setRuntimeConfig(func(runtime *config.Config) { runtime.Subscribe = value })
}

func (s *ServiceContext) SetVerifyConfig(value config.Verify) {
	s.setRuntimeConfig(func(runtime *config.Config) { runtime.Verify = value })
}

func (s *ServiceContext) SetVerifyCodeConfig(value config.VerifyCode) {
	s.setRuntimeConfig(func(runtime *config.Config) { runtime.VerifyCode = value })
}

func (s *ServiceContext) SetNodeConfig(value config.NodeConfig) {
	s.setRuntimeConfig(func(runtime *config.Config) { runtime.Node = value })
}

func (s *ServiceContext) SetEmailConfig(value config.EmailConfig) {
	s.setRuntimeConfig(func(runtime *config.Config) { runtime.Email = value })
}

func (s *ServiceContext) SetRegisterConfig(value config.RegisterConfig) {
	s.setRuntimeConfig(func(runtime *config.Config) { runtime.Register = value })
}

func (s *ServiceContext) SetInviteConfig(value config.InviteConfig) {
	s.setRuntimeConfig(func(runtime *config.Config) { runtime.Invite = value })
}

func (s *ServiceContext) setRuntimeConfig(update func(*config.Config)) {
	if s == nil || update == nil {
		return
	}
	s.runtimeMu.Lock()
	update(&s.Config)
	s.runtimeMu.Unlock()
}

func (s *ServiceContext) NodeMultiplierManager() *nodeMultiplier.Manager {
	if s == nil {
		return defaultNodeMultiplierManager
	}
	manager := s.nodeMultiplierManager.Load()
	if manager == nil {
		return defaultNodeMultiplierManager
	}
	return manager
}

func (s *ServiceContext) SetNodeMultiplierManager(manager *nodeMultiplier.Manager) {
	if s == nil {
		return
	}
	if manager == nil {
		manager = defaultNodeMultiplierManager
	}
	s.nodeMultiplierManager.Store(manager)
}

func (s *ServiceContext) LogConfig() config.Log {
	if s == nil {
		return config.Log{}
	}
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	return s.Config.Log
}

func (s *ServiceContext) SetLogConfig(value config.Log) {
	if s == nil {
		return
	}
	s.runtimeMu.Lock()
	s.Config.Log = value
	s.runtimeMu.Unlock()
}

func (s *ServiceContext) TelegramRuntime() (config.Telegram, *tgbotapi.BotAPI) {
	if s == nil {
		return config.Telegram{}, nil
	}
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	return s.Config.Telegram, s.telegramBot
}

func (s *ServiceContext) TelegramBotClient() *tgbotapi.BotAPI {
	_, bot := s.TelegramRuntime()
	return bot
}

func (s *ServiceContext) SetTelegramRuntime(value config.Telegram, bot *tgbotapi.BotAPI) {
	if s == nil {
		return
	}
	s.runtimeMu.Lock()
	oldBot := s.telegramBot
	s.Config.Telegram = value
	s.telegramBot = bot
	s.runtimeMu.Unlock()
	if oldBot != nil && oldBot != bot {
		oldBot.StopReceivingUpdates()
	}
}
