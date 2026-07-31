package handler

import (
	"github.com/neibniz/alpha/server/internal/svc"
	orderLogic "github.com/neibniz/alpha/server/queue/logic/order"
	"github.com/neibniz/alpha/server/queue/logic/subscription"
	"github.com/neibniz/alpha/server/queue/logic/task"
	"github.com/neibniz/alpha/server/queue/logic/traffic"
	"github.com/neibniz/alpha/server/queue/types"
	"github.com/hibiken/asynq"

	emailLogic "github.com/neibniz/alpha/server/queue/logic/email"
)

func RegisterHandlers(mux *asynq.ServeMux, serverCtx *svc.ServiceContext) {
	// Send email task
	mux.Handle(types.ForthwithSendEmail, emailLogic.NewSendEmailLogic(serverCtx))
	// Defer close order task
	mux.Handle(types.DeferCloseOrder, orderLogic.NewDeferCloseOrderLogic(serverCtx))
	// Forthwith activate order task
	mux.Handle(types.ForthwithActivateOrder, orderLogic.NewActivateOrderLogic(serverCtx))

	// Forthwith traffic statistics
	trafficStatisticsHandler := traffic.NewTrafficStatisticsLogic(serverCtx)
	mux.Handle(types.ForthwithTrafficStatistics, trafficStatisticsHandler)
	mux.Handle(types.ForthwithTrafficStatisticsV2, trafficStatisticsHandler)
	mux.Handle(types.ForthwithTrafficAnalyticsSync, traffic.NewTrafficAnalyticsSyncLogic(serverCtx))
	mux.Handle(types.SchedulerTrafficAnalytics, traffic.NewTrafficAnalyticsSyncLogic(serverCtx))

	// Schedule check subscription
	mux.Handle(types.SchedulerCheckSubscription, subscription.NewCheckSubscriptionLogic(serverCtx))

	// Schedule reset traffic
	mux.Handle(types.SchedulerResetTraffic, traffic.NewResetTrafficLogic(serverCtx))

	// ScheduledTrafficStat
	mux.Handle(types.SchedulerTrafficStat, traffic.NewStatLogic(serverCtx))

	// Scheduled exchange-rate refresh
	mux.Handle(types.SchedulerExchangeRate, task.NewRateLogic(serverCtx))
}
