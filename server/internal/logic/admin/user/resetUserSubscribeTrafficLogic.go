package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type ResetUserSubscribeTrafficLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewResetUserSubscribeTrafficLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetUserSubscribeTrafficLogic {
	return &ResetUserSubscribeTrafficLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResetUserSubscribeTrafficLogic) ResetUserSubscribeTraffic(req *types.ResetUserSubscribeTrafficRequest) error {
	logic := NewBatchResetUserSubscribeTrafficLogic(l.ctx, l.svcCtx)
	resp, err := logic.Reset(&types.BatchResetUserSubscribeTrafficRequest{
		Scope:            batchTrafficResetScopeSubscribes,
		UserSubscribeIds: []int64{req.UserSubscribeId},
	})
	if err != nil {
		return err
	}
	if resp.ResetCount == 0 {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "subscription not found")
	}
	return nil
}
