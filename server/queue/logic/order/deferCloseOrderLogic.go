package orderLogic

import (
	"context"
	"encoding/json"

	"github.com/neibniz/alpha/server/pkg/logger"

	"github.com/neibniz/alpha/server/internal/logic/public/order"
	"github.com/neibniz/alpha/server/internal/svc"
	internal "github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/queue/types"
	"github.com/hibiken/asynq"
)

type DeferCloseOrderLogic struct {
	svc *svc.ServiceContext
}

func NewDeferCloseOrderLogic(svc *svc.ServiceContext) *DeferCloseOrderLogic {
	return &DeferCloseOrderLogic{
		svc: svc,
	}
}

func (l *DeferCloseOrderLogic) ProcessTask(ctx context.Context, task *asynq.Task) error {
	payload := types.DeferCloseOrderPayload{}
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		logger.WithContext(ctx).Error("[DeferCloseOrderLogic] Unmarshal payload failed",
			logger.Field("error", err.Error()),
			logger.Field("payload_bytes", len(task.Payload())),
		)
		return nil
	}

	err := order.NewCloseOrderLogic(ctx, l.svc).CloseOrder(&internal.CloseOrderRequest{
		OrderNo: payload.OrderNo,
	})
	count, ok := asynq.GetRetryCount(ctx)
	if !ok {
		return nil
	}
	if err != nil && count < 3 {
		return err
	}
	return nil
}
