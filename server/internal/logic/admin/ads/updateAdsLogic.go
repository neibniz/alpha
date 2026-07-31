package ads

import (
	"context"
	"time"

	adsModel "github.com/neibniz/alpha/server/internal/model/ads"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type UpdateAdsLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Update Ads
func NewUpdateAdsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAdsLogic {
	return &UpdateAdsLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateAdsLogic) UpdateAds(req *types.UpdateAdsRequest) error {
	data, err := l.svcCtx.AdsModel.FindOneForAdmin(l.ctx, req.Id, req.Content != nil)
	if err != nil {
		l.Errorw("find ads error", logger.Field("error", err.Error()), logger.Field("id", req.Id))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find ads error: %v", err.Error())
	}
	update := adsModel.AdsFieldsUpdate{}
	if req.Title != nil {
		update.Title = req.Title
	}
	if req.Type != nil {
		update.Type = req.Type
	}
	if req.Content != nil {
		update.Content = req.Content
	}
	if req.Description != nil {
		update.Description = req.Description
	}
	if req.TargetURL != nil {
		update.TargetURL = req.TargetURL
	}
	if req.StartTime != nil {
		startTime := time.UnixMilli(*req.StartTime)
		update.StartTime = &startTime
	}
	if req.EndTime != nil {
		endTime := time.UnixMilli(*req.EndTime)
		update.EndTime = &endTime
	}
	if req.Status != nil {
		update.Status = req.Status
	}
	if err := l.svcCtx.AdsModel.UpdateFields(l.ctx, data, update); err != nil {
		l.Errorw("update ads error", logger.Field("error", err.Error()), logger.Field("req", req))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update ads error: %v", err.Error())
	}
	return nil
}
