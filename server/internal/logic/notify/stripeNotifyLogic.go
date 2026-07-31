package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/payment/stripe"
)

type StripeNotifyLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewStripeNotifyLogic Stripe notify
func NewStripeNotifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StripeNotifyLogic {
	return &StripeNotifyLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StripeNotifyLogic) StripeNotify(r *http.Request, w http.ResponseWriter) error {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		l.Errorw("[StripeNotify] error", logger.Field("errors", err.Error()))
		return err
	}
	signature := r.Header.Get("Stripe-Signature")
	stripeConfig, ok := l.ctx.Value(constant.CtxKeyPayment).(*payment.Payment)
	if !ok {
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "payment config not found")
	}
	config := payment.StripeConfig{}
	if err := json.Unmarshal([]byte(stripeConfig.Config), &config); err != nil {
		return err
	}
	client := stripe.NewClient(stripe.Config{
		PublicKey:     config.PublicKey,
		SecretKey:     config.SecretKey,
		WebhookSecret: config.WebhookSecret,
	})

	notify, err := client.ParseNotify(payload, signature)
	if err != nil {
		l.Errorw("[StripeNotify] error", logger.Field("errors", err.Error()))
		return err
	}
	orderInfo, err := l.svcCtx.OrderModel.FindOneForPaymentNotify(l.ctx, notify.OrderNo)
	if err != nil {
		l.Logger.Error("[StripeNotify] Find order failed", logger.Field("error", err.Error()), logger.Field("orderNo", notify.OrderNo))
		return errors.Wrapf(xerr.NewErrCode(xerr.OrderNotExist), "order not exist: %v", notify.OrderNo)
	}
	if notify.EventType == "payment_intent.succeeded" {
		if orderInfo.Status == notifyOrderStatusFinished {
			return nil
		}
		if orderInfo.Status != notifyOrderStatusPending && orderInfo.Status != notifyOrderStatusPaid {
			l.Logger.Info("[StripeNotify] Ignore order with non-payable status", logger.Field("orderNo", notify.OrderNo), logger.Field("status", orderInfo.Status))
			return nil
		}
		if !validateNotifyAmount(l.svcCtx, orderInfo, notify.Amount) {
			l.Logger.Error("[StripeNotify] Amount mismatch", logger.Field("orderNo", notify.OrderNo), logger.Field("orderAmount", orderInfo.Amount), logger.Field("paidAmount", notify.Amount))
			return nil
		}

		paid, err := markOrderPaid(l.ctx, l.svcCtx, notify.OrderNo, notify.TradeNo)
		if err != nil {
			return err
		}
		if !paid && orderInfo.Status != notifyOrderStatusPaid {
			l.Logger.Info("[StripeNotify] Order status changed before processing", logger.Field("orderNo", notify.OrderNo), logger.Field("status", orderInfo.Status))
			return nil
		}

		_, err = enqueueActivateOrder(l.ctx, l.svcCtx, notify.OrderNo)
		if err != nil {
			l.Errorw("[StripeNotify] Enqueue error", logger.Field("errors", err.Error()))
			return err
		}
		l.Infow("[StripeNotify] success", logger.Field("orderNo", notify.OrderNo))
	}
	return nil
}
