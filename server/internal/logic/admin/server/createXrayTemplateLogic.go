package server

import (
	"context"
	"strings"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/xraytemplate"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type CreateXrayTemplateLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateXrayTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateXrayTemplateLogic {
	return &CreateXrayTemplateLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateXrayTemplateLogic) CreateXrayTemplate(req *types.CreateXrayTemplateRequest) error {
	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	name, description, err := normalizeXrayTemplateIdentity(req.Name, req.Description)
	if err != nil {
		return errors.Wrap(err, "validate xray template identity")
	}
	req.Name = name
	req.Description = description
	if err := validateXrayTemplateType(req.Type); err != nil {
		return err
	}
	cfg, err := normalizeXrayTemplateConfig(req.Config)
	if err != nil {
		return err
	}
	variablesSchema, err := normalizeXrayTemplateConfig(req.VariablesSchema)
	if err != nil {
		return err
	}
	defaultVariables, err := normalizeXrayTemplateConfig(req.DefaultVariables)
	if err != nil {
		return err
	}
	subscriptionMeta, err := normalizeXrayTemplateConfig(req.SubscriptionMeta)
	if err != nil {
		return err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	template := &node.XrayTemplate{
		Name:             req.Name,
		Type:             req.Type,
		Config:           cfg,
		ConfigTemplate:   req.ConfigTemplate,
		VariablesSchema:  variablesSchema,
		DefaultVariables: defaultVariables,
		SubscriptionMeta: subscriptionMeta,
		Description:      req.Description,
		Enabled:          &enabled,
	}
	if err = xraytemplate.ValidateTemplateConfig(template); err != nil {
		return errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "invalid xray template config"), "validate xray template config: %v", err)
	}
	if err = l.svcCtx.NodeModel.InsertXrayTemplate(l.ctx, template); err != nil {
		l.Errorf("[CreateXrayTemplate] insert error: %v", err.Error())
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "insert xray template error: %v", err)
	}
	return nil
}
