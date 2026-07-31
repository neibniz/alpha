package order

import (
	"context"
	"encoding/json"
	"time"

	"github.com/neibniz/alpha/server/pkg/logger"
	queue "github.com/neibniz/alpha/server/queue/types"
	"github.com/hibiken/asynq"
)

func enqueueDeferredCloseOrder(ctx context.Context, queueClient *asynq.Client, log logger.Logger, orderNo string) error {
	payload := queue.DeferCloseOrderPayload{OrderNo: orderNo}
	val, err := json.Marshal(payload)
	if err != nil {
		log.Errorw("[Order] Marshal defer close payload error", logger.Field("error", err.Error()))
		return err
	}
	task := asynq.NewTask(queue.DeferCloseOrder, val, asynq.MaxRetry(3), asynq.Unique(time.Duration(CloseOrderTimeMinutes+5)*time.Minute))
	taskInfo, err := queueClient.EnqueueContext(ctx, task, asynq.ProcessIn(CloseOrderTimeMinutes*time.Minute))
	if err != nil {
		log.Errorw("[Order] Enqueue defer close task error", logger.Field("error", err.Error()), logger.Field("task", task))
		return err
	}
	log.Infow("[Order] Enqueue defer close task success", logger.Field("TaskID", taskInfo.ID), logger.Field("order_no", orderNo))
	return nil
}
