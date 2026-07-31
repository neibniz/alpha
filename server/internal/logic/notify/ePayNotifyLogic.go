package notify

import (
	"encoding/json"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/gin-gonic/gin"

	"github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/queryparams"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/payment/epay"
)

type EPayNotifyLogic struct {
	logger.Logger
	ctx    *gin.Context
	svcCtx *svc.ServiceContext
}

// EPay notify
func NewEPayNotifyLogic(ctx *gin.Context, svcCtx *svc.ServiceContext) *EPayNotifyLogic {
	return &EPayNotifyLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EPayNotifyLogic) EPayNotify(req *types.EPayNotifyRequest) error {

	// Find payment config
	data, ok := l.ctx.Request.Context().Value(constant.CtxKeyPayment).(*payment.Payment)
	if !ok {
		l.Logger.Error("[EPayNotify] Payment not found in context")
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "payment config not found")
	}
	orderInfo, err := l.svcCtx.OrderModel.FindOneForPaymentNotify(l.ctx, req.OutTradeNo)
	if err != nil {
		l.Logger.Error("[EPayNotify] Find order failed", logger.Field("error", err.Error()), logger.Field("orderNo", req.OutTradeNo))
		return errors.Wrapf(xerr.NewErrCode(xerr.OrderNotExist), "order not exist: %v", req.OutTradeNo)
	}

	var config payment.EPayConfig
	if err := json.Unmarshal([]byte(data.Config), &config); err != nil {
		l.Logger.Errorw("[EPayNotify] Unmarshal config failed", logger.Field("error", err.Error()))
		return err
	}
	// Verify sign
	client := epay.NewClient(config.Pid, config.Url, config.Key, config.Type)
	if !client.VerifySign(queryparams.ParseFirstValues(l.ctx.Request.Form.Encode())) {
		l.Logger.Error("[EPayNotify] Verify sign failed")
		return errors.New("invalid payment signature")
	}
	if req.TradeStatus != "TRADE_SUCCESS" {
		l.Logger.Error("[EPayNotify] Trade status is not success", logger.Field("orderNo", req.OutTradeNo), logger.Field("tradeStatus", req.TradeStatus))
		return nil
	}
	if orderInfo.Status == notifyOrderStatusFinished {
		return nil
	}
	if orderInfo.Status != notifyOrderStatusPending && orderInfo.Status != notifyOrderStatusPaid {
		l.Logger.Info("[EPayNotify] Ignore order with non-payable status", logger.Field("orderNo", req.OutTradeNo), logger.Field("status", orderInfo.Status))
		return nil
	}
	paidAmount, err := parseYuanToCents(req.Money)
	if err != nil {
		l.Logger.Error("[EPayNotify] Parse amount failed", logger.Field("error", err.Error()), logger.Field("orderNo", req.OutTradeNo), logger.Field("money", req.Money))
		return err
	}
	if !validateNotifyAmount(l.svcCtx, orderInfo, paidAmount) {
		l.Logger.Error("[EPayNotify] Amount mismatch", logger.Field("orderNo", req.OutTradeNo), logger.Field("orderAmount", orderInfo.Amount), logger.Field("paidAmount", paidAmount))
		return nil
	}

	paid, err := markOrderPaid(l.ctx, l.svcCtx, req.OutTradeNo, req.TradeNo)
	if err != nil {
		l.Logger.Error("[EPayNotify] Update order status failed", logger.Field("error", err.Error()), logger.Field("orderNo", req.OutTradeNo))
		return err
	}
	if !paid && orderInfo.Status != notifyOrderStatusPaid {
		l.Logger.Info("[EPayNotify] Order status changed before processing", logger.Field("orderNo", req.OutTradeNo), logger.Field("status", orderInfo.Status))
		return nil
	}

	taskInfo, err := enqueueActivateOrder(l.ctx, l.svcCtx, req.OutTradeNo)
	if err != nil {
		l.Logger.Error("[EPayNotify] Enqueue task failed", logger.Field("error", err.Error()))
		return err
	}
	l.Logger.Info("[EPayNotify] Enqueue task success", logger.Field("taskInfo", taskInfo))
	return nil
}
