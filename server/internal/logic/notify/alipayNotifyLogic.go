package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/payment/alipay"
)

type AlipayNotifyLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Alipay notify
func NewAlipayNotifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlipayNotifyLogic {
	return &AlipayNotifyLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AlipayNotifyLogic) AlipayNotify(r *http.Request) error {
	data, ok := l.ctx.Value(constant.CtxKeyPayment).(*payment.Payment)
	if !ok {
		return fmt.Errorf("payment config not found")
	}
	var config payment.AlipayF2FConfig
	if err := json.Unmarshal([]byte(data.Config), &config); err != nil {
		l.Logger.Error("[AlipayNotify] Unmarshal config failed", logger.Field("error", err.Error()))
		return err
	}
	client := alipay.NewClient(alipay.Config{
		AppId:       config.AppId,
		PrivateKey:  config.PrivateKey,
		PublicKey:   config.PublicKey,
		InvoiceName: config.InvoiceName,
		NotifyURL:   data.Domain + "/v1/payment/alipay/notify",
	})
	notify, err := client.DecodeNotification(r.Form)
	if err != nil {
		l.Logger.Error("[AlipayNotify] Decode notification failed", logger.Field("error", err.Error()))
		return err
	}
	if notify.Status == alipay.Success {
		orderInfo, err := l.svcCtx.OrderModel.FindOneForPaymentNotify(l.ctx, notify.OrderNo)
		if err != nil {
			l.Logger.Error("[AlipayNotify] Find order failed", logger.Field("error", err.Error()), logger.Field("orderNo", notify.OrderNo))
			return errors.Wrapf(xerr.NewErrCode(xerr.OrderNotExist), "order not exist: %v", notify.OrderNo)
		}

		if orderInfo.Status == notifyOrderStatusFinished {
			return nil
		}
		if orderInfo.Status != notifyOrderStatusPending && orderInfo.Status != notifyOrderStatusPaid {
			l.Logger.Info("[AlipayNotify] Ignore order with non-payable status", logger.Field("orderNo", notify.OrderNo), logger.Field("status", orderInfo.Status))
			return nil
		}
		if !validateNotifyAmount(l.svcCtx, orderInfo, notify.Amount) {
			l.Logger.Error("[AlipayNotify] Amount mismatch", logger.Field("orderNo", notify.OrderNo), logger.Field("orderAmount", orderInfo.Amount), logger.Field("paidAmount", notify.Amount))
			return nil
		}

		paid, err := markOrderPaid(l.ctx, l.svcCtx, notify.OrderNo, "")
		if err != nil {
			l.Logger.Error("[AlipayNotify] Update order status failed", logger.Field("error", err.Error()), logger.Field("orderNo", notify.OrderNo))
			return err
		}
		if !paid && orderInfo.Status != notifyOrderStatusPaid {
			l.Logger.Info("[AlipayNotify] Order status changed before processing", logger.Field("orderNo", notify.OrderNo), logger.Field("status", orderInfo.Status))
			return nil
		}
		l.Logger.Info("[AlipayNotify] Notify status success", logger.Field("orderNo", notify.OrderNo))
		taskInfo, err := enqueueActivateOrder(l.ctx, l.svcCtx, notify.OrderNo)
		if err != nil {
			l.Logger.Error("[AlipayNotify] Enqueue task failed", logger.Field("error", err.Error()))
			return err
		}
		l.Logger.Info("[AlipayNotify] Enqueue task success", logger.Field("taskInfo", taskInfo))
	} else {
		l.Logger.Error("[AlipayNotify] Notify status failed", logger.Field("status", string(notify.Status)))
	}
	return nil
}
