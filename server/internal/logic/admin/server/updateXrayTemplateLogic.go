package server

import (
	"context"
	"strings"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/xraytemplate"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type UpdateXrayTemplateLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateXrayTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateXrayTemplateLogic {
	return &UpdateXrayTemplateLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateXrayTemplateLogic) UpdateXrayTemplate(req *types.UpdateXrayTemplateRequest) error {
	if req.Id == 0 {
		return errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "template id is required"), "template id is required")
	}
	template, err := l.svcCtx.NodeModel.FindOneXrayTemplate(l.ctx, req.Id)
	if err != nil {
		l.Errorf("[UpdateXrayTemplate] query error: %v", err.Error())
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query xray template error: %v", err)
	}

	if req.Name != nil {
		template.Name = *req.Name
	}
	if req.Type != nil {
		template.Type = strings.ToLower(strings.TrimSpace(*req.Type))
		if err := validateXrayTemplateType(template.Type); err != nil {
			return err
		}
	}
	if req.Config != nil {
		cfg, err := normalizeXrayTemplateConfig(req.Config)
		if err != nil {
			return err
		}
		template.Config = cfg
	}
	if req.ConfigTemplate != nil {
		template.ConfigTemplate = *req.ConfigTemplate
	}
	if req.VariablesSchema != nil {
		variablesSchema, err := normalizeXrayTemplateConfig(req.VariablesSchema)
		if err != nil {
			return err
		}
		template.VariablesSchema = variablesSchema
	}
	if req.DefaultVariables != nil {
		defaultVariables, err := normalizeXrayTemplateConfig(req.DefaultVariables)
		if err != nil {
			return err
		}
		template.DefaultVariables = defaultVariables
	}
	if req.SubscriptionMeta != nil {
		subscriptionMeta, err := normalizeXrayTemplateConfig(req.SubscriptionMeta)
		if err != nil {
			return err
		}
		template.SubscriptionMeta = subscriptionMeta
	}
	if req.Description != nil {
		template.Description = *req.Description
	}
	if req.Enabled != nil {
		template.Enabled = req.Enabled
	}
	name, description, err := normalizeXrayTemplateIdentity(template.Name, template.Description)
	if err != nil {
		return errors.Wrap(err, "validate xray template identity")
	}
	template.Name = name
	template.Description = description
	if err = xraytemplate.ValidateTemplateConfig(template); err != nil {
		return errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "invalid xray template config"), "validate xray template config: %v", err)
	}
	if err = l.svcCtx.NodeModel.Transaction(l.ctx, func(db *gorm.DB) error {
		return l.svcCtx.NodeModel.UpdateXrayTemplate(l.ctx, template, db)
	}); err != nil {
		l.Errorf("[UpdateXrayTemplate] update error: %v", err.Error())
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update xray template error: %v", err)
	}
	return nil
}
