package common

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/ads"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type GetAdsLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get Ads
func NewGetAdsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAdsLogic {
	return &GetAdsLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAdsLogic) GetAds(req *types.GetAdsRequest) (resp *types.GetAdsResponse, err error) {
	// todo: add ads position and device
	status := 1
	_, data, err := l.svcCtx.AdsModel.GetAdsListByPage(l.ctx, 1, 200, ads.Filter{
		Status: &status,
	})
	if err != nil {
		return nil, err
	}
	resp = &types.GetAdsResponse{
		List: adsRows(data),
	}
	return
}
