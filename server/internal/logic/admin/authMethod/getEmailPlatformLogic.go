package authMethod

import (
	"context"

	"github.com/neibniz/alpha/server/pkg/email"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type GetEmailPlatformLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get email support platform
func NewGetEmailPlatformLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetEmailPlatformLogic {
	return &GetEmailPlatformLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetEmailPlatformLogic) GetEmailPlatform() (resp *types.PlatformResponse, err error) {
	return &types.PlatformResponse{
		List: email.GetSupportedPlatforms(),
	}, nil
}
