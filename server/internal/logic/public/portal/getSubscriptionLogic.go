package portal

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetSubscriptionLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetSubscriptionLogic Get Subscription
func NewGetSubscriptionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSubscriptionLogic {
	return &GetSubscriptionLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSubscriptionLogic) GetSubscription(req *types.GetSubscriptionRequest) (resp *types.GetSubscriptionResponse, err error) {
	// Get the subscription list
	_, data, err := l.svcCtx.SubscribeModel.FilterList(l.ctx, &subscribe.FilterParams{
		Page:            1,
		Size:            9999,
		MaxSize:         9999,
		Show:            true,
		Language:        req.Language,
		DefaultLanguage: true,
		Projection:      subscribe.FilterListProjectionPublic,
		SkipCount:       true,
	})
	if err != nil {
		l.Errorw("[Site GetSubscription]", logger.Field("err", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get subscription list error: %v", err.Error())
	}
	resp = &types.GetSubscriptionResponse{
		List: subscribeRowsFromModels(data),
	}
	return
}
