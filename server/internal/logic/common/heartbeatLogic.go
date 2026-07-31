package common

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type HeartbeatLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewHeartbeatLogic Heartbeat
func NewHeartbeatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HeartbeatLogic {
	return &HeartbeatLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HeartbeatLogic) Heartbeat() (resp *types.HeartbeatResponse, err error) {
	return &types.HeartbeatResponse{
		Status:    true,
		Message:   "service is alive",
		Timestamp: time.Now().Unix(),
	}, nil
}
