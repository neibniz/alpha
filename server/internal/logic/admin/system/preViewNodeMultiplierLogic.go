package system

import (
	"context"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"time"
)

type PreViewNodeMultiplierLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// PreView Node Multiplier
func NewPreViewNodeMultiplierLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreViewNodeMultiplierLogic {
	return &PreViewNodeMultiplierLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreViewNodeMultiplierLogic) PreViewNodeMultiplier() (resp *types.PreViewNodeMultiplierResponse, err error) {
	now := time.Now()
	ratio := l.svcCtx.NodeMultiplierManager().GetMultiplier(now)
	return &types.PreViewNodeMultiplierResponse{
		Ratio:       ratio,
		CurrentTime: now.Format("2006-01-02 15:04:05"),
	}, nil
}
