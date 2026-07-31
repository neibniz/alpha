package application

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/client"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type UpdateSubscribeApplicationLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUpdateSubscribeApplicationLogic Update subscribe application
func NewUpdateSubscribeApplicationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSubscribeApplicationLogic {
	return &UpdateSubscribeApplicationLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateSubscribeApplicationLogic) UpdateSubscribeApplication(req *types.UpdateSubscribeApplicationRequest) (resp *types.SubscribeApplication, err error) {
	if err := validateUpdateSubscribeApplication(req); err != nil {
		return nil, errors.Wrap(err, "validate subscribe application")
	}
	data, err := l.svcCtx.ClientModel.FindOneForUpdate(l.ctx, req.Id)
	if err != nil {
		l.Errorf("Failed to find subscribe application with ID %d: %v", req.Id, err)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Failed to find subscribe application with ID %d", req.Id)
	}
	wasDefault := data.IsDefault
	if wasDefault && req.Enabled != nil && !*req.Enabled {
		return nil, errors.Wrap(
			xerr.NewErrCodeMsg(xerr.InvalidParams, "default subscription client must remain enabled"),
			"set another default subscription client before disabling this one",
		)
	}
	if wasDefault && req.IsDefault != nil && !*req.IsDefault {
		return nil, errors.Wrap(
			xerr.NewErrCodeMsg(xerr.InvalidParams, "default subscription client cannot be unset directly"),
			"set another subscription client as default instead",
		)
	}

	responseLink := downloadLinkFromRaw(data.DownloadLink)
	update := client.SubscribeApplicationFieldsUpdate{}
	if req.Name != nil {
		data.Name = *req.Name
		update.Name = req.Name
	}
	if req.Icon != nil {
		data.Icon = *req.Icon
		update.Icon = req.Icon
	}
	if req.Description != nil {
		data.Description = *req.Description
		update.Description = req.Description
	}
	if req.Scheme != nil {
		data.Scheme = *req.Scheme
		update.Scheme = req.Scheme
	}
	if req.UserAgent != nil {
		data.UserAgent = *req.UserAgent
		update.UserAgent = req.UserAgent
	}
	if req.Enabled != nil {
		data.Enabled = *req.Enabled
		update.Enabled = req.Enabled
	}
	if req.IsDefault != nil {
		data.IsDefault = *req.IsDefault
		update.IsDefault = req.IsDefault
		if *req.IsDefault {
			enabled := true
			data.Enabled = enabled
			update.Enabled = &enabled
		}
	}
	if req.SubscribeTemplate != nil {
		data.SubscribeTemplate = *req.SubscribeTemplate
		update.SubscribeTemplate = req.SubscribeTemplate
	}
	if req.OutputFormat != nil {
		data.OutputFormat = *req.OutputFormat
		update.OutputFormat = req.OutputFormat
	}
	if req.DownloadLink != nil {
		link := downloadLinkToModel(*req.DownloadLink)
		linkData, err := link.Marshal()
		if err != nil {
			l.Errorf("Failed to marshal download link: %v", err)
			return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), " Failed to marshal download link")
		}
		data.DownloadLink = string(linkData)
		update.DownloadLink = &data.DownloadLink
		responseLink = downloadLinkFromModel(link)
	}
	if req.IsDefault != nil && *req.IsDefault {
		err = l.svcCtx.ClientModel.Transaction(l.ctx, func(db *gorm.DB) error {
			if updateErr := db.Model(&client.SubscribeApplication{}).
				Where("id <> ? AND is_default = ?", data.Id, true).
				Update("is_default", false).Error; updateErr != nil {
				return updateErr
			}
			return client.NewSubscribeApplicationModel(db).UpdateFields(l.ctx, data.Id, update)
		})
	} else {
		err = l.svcCtx.ClientModel.UpdateFields(l.ctx, data.Id, update)
	}
	if err != nil {
		l.Errorf("Failed to update subscribe application with ID %d: %v", req.Id, err)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "Failed to update subscribe application with ID %d", req.Id)
	}
	application := subscribeApplicationFromModel(data, responseLink)
	if req.SubscribeTemplate == nil {
		application.SubscribeTemplate = ""
	}
	resp = &application
	return
}
