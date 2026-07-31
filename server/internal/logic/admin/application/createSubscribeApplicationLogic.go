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

type CreateSubscribeApplicationLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewCreateSubscribeApplicationLogic Create subscribe application
func NewCreateSubscribeApplicationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSubscribeApplicationLogic {
	return &CreateSubscribeApplicationLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateSubscribeApplicationLogic) CreateSubscribeApplication(req *types.CreateSubscribeApplicationRequest) (resp *types.SubscribeApplication, err error) {
	if err := validateCreateSubscribeApplication(req); err != nil {
		return nil, errors.Wrap(err, "validate subscribe application")
	}
	link := downloadLinkToModel(req.DownloadLink)
	linkData, err := link.Marshal()
	if err != nil {
		l.Errorf("Failed to marshal download link: %v", err)
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), " Failed to marshal download link")
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	data := &client.SubscribeApplication{
		Name:              req.Name,
		Icon:              req.Icon,
		Description:       req.Description,
		Scheme:            req.Scheme,
		UserAgent:         req.UserAgent,
		Enabled:           enabled,
		IsDefault:         req.IsDefault,
		SubscribeTemplate: req.SubscribeTemplate,
		OutputFormat:      req.OutputFormat,
		DownloadLink:      string(linkData),
	}

	err = l.svcCtx.ClientModel.Transaction(l.ctx, func(db *gorm.DB) error {
		if !data.IsDefault {
			var count int64
			if countErr := db.Model(&client.SubscribeApplication{}).Count(&count).Error; countErr != nil {
				return countErr
			}
			data.IsDefault = count == 0
		}
		if data.IsDefault {
			if updateErr := db.Model(&client.SubscribeApplication{}).
				Where("is_default = ?", true).
				Update("is_default", false).Error; updateErr != nil {
				return updateErr
			}
			data.Enabled = true
		}
		return client.NewSubscribeApplicationModel(db).Insert(l.ctx, data)
	})
	if err != nil {
		l.Errorf("Failed to create subscribe application: %v", err)
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseInsertError), "Failed to create subscribe application")
	}

	application := subscribeApplicationFromModel(data, downloadLinkFromModel(link))
	resp = &application

	return
}
