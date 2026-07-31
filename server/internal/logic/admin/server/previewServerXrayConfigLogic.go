package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/internal/xraytemplate"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type PreviewServerXrayConfigLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPreviewServerXrayConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewServerXrayConfigLogic {
	return &PreviewServerXrayConfigLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreviewServerXrayConfigLogic) PreviewServerXrayConfig(req *types.PreviewServerXrayConfigRequest) (*types.PreviewServerXrayConfigResponse, error) {
	if req.ServerId == 0 {
		return nil, errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "server id is required"), "server id is required")
	}
	srv, err := l.svcCtx.NodeModel.FindOneServerForRuntime(l.ctx, req.ServerId)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query server error: %v", err)
	}

	bindings, _, err := buildServerXrayTemplateBindings(l.ctx, l.svcCtx.DB, req.ServerId, req.Bindings, xrayBindingBuildOptions{skipDisabled: true})
	if err != nil {
		return nil, err
	}

	cfg, proxies, err := buildPreviewXrayConfigResponse(srv, bindings)
	if err != nil {
		return nil, err
	}
	resp := &types.PreviewServerXrayConfigResponse{
		QueryServerXrayConfigResponse: *cfg,
	}
	resp.Variables = buildVariablePreview(srv, bindings)
	resp.SubscriptionProxies, err = rawSubscriptionProxyRows(proxies)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func rawSubscriptionProxyRows(proxies []xraytemplate.RenderedSubscriptionProxy) ([]json.RawMessage, error) {
	collector := viewutil.NewRowsCollector[json.RawMessage](len(proxies))
	for _, rendered := range proxies {
		if rendered.Proxy == nil {
			continue
		}
		rawProxy, err := xraytemplate.RawConfigValue(rendered.Proxy)
		if err != nil {
			return nil, err
		}
		collector.Add(rawProxy)
	}
	return collector.Rows(), nil
}

func buildVariablePreview(server *node.Server, bindings []*node.ServerXrayTemplate) map[string]types.XrayTemplateVariablePreview {
	rows, ref := buildVariablePreviewRows(server, bindings)
	result := make(map[string]types.XrayTemplateVariablePreview, len(rows))
	for _, row := range rows {
		binding := row.binding
		key := binding.Template.Type + "." + row.alias
		variables := row.variables
		variables.ref = ref[binding.Template.Type][row.refAlias]
		result[key] = types.XrayTemplateVariablePreview{
			Type:                        binding.Template.Type,
			Alias:                       row.alias,
			TemplateId:                  binding.TemplateId,
			TemplateName:                binding.Template.Name,
			Sort:                        binding.Sort,
			ConfigVariables:             previewRawObject(variables.config),
			DefaultVariables:            previewRawObject(variables.defaults),
			BindingVariables:            previewRawObject(variables.binding),
			MergedVariables:             previewRawObject(variables.merged),
			SubscriptionVariables:       previewRawObject(variables.subscription),
			SubscriptionMergedVariables: previewRawObject(variables.subscriptionMerged),
			Ref:                         previewRawObject(variables.ref),
		}
	}
	return result
}

func buildVariablePreviewRows(server *node.Server, bindings []*node.ServerXrayTemplate) ([]variablePreviewRow, xraytemplate.Ref) {
	collector := viewutil.NewRowsCollector[variablePreviewRow](len(bindings))
	ref := newVariablePreviewRef()
	for index, binding := range bindings {
		if binding == nil || binding.Template == nil {
			continue
		}
		alias := binding.Alias
		if alias == "" {
			alias = binding.Template.Name
		}
		refAlias := variablePreviewRefAlias(binding, index)
		variables := buildVariablePreviewData(binding)
		setVariablePreviewRef(ref, binding.Template.Type, refAlias, variablePreviewRefObject(server, refAlias, variables.merged))
		collector.Add(variablePreviewRow{
			binding:   binding,
			alias:     alias,
			refAlias:  refAlias,
			variables: variables,
		})
	}

	return collector.Rows(), ref
}

type variablePreviewRow struct {
	binding   *node.ServerXrayTemplate
	alias     string
	refAlias  string
	variables variablePreviewData
}

type variablePreviewData struct {
	config             xraytemplate.ConfigObject
	defaults           xraytemplate.ConfigObject
	binding            xraytemplate.ConfigObject
	merged             xraytemplate.ConfigObject
	subscription       xraytemplate.ConfigObject
	subscriptionMerged xraytemplate.ConfigObject
	ref                xraytemplate.ConfigObject
}

func buildVariablePreviewData(binding *node.ServerXrayTemplate) variablePreviewData {
	configVariables := xraytemplate.ConfigVariables(binding)
	defaultVariables := xraytemplate.JSONToObject(binding.Template.DefaultVariables)
	bindingVariables := xraytemplate.JSONToObject(binding.Variables)
	subscriptionVariables := xraytemplate.JSONToObject(binding.SubscriptionVariables)
	mergedVariables := mergeVariablePreviewObjects(defaultVariables, configVariables, bindingVariables)
	subscriptionMergedVariables := mergeVariablePreviewObjects(mergedVariables, subscriptionVariables)
	return variablePreviewData{
		config:             configVariables,
		defaults:           defaultVariables,
		binding:            bindingVariables,
		merged:             mergedVariables,
		subscription:       subscriptionVariables,
		subscriptionMerged: subscriptionMergedVariables,
	}
}

func newVariablePreviewRef() xraytemplate.Ref {
	return xraytemplate.Ref{
		node.XrayTemplateInbound:  {},
		node.XrayTemplateOutbound: {},
		node.XrayTemplateDNS:      {},
		node.XrayTemplateRouting:  {},
		node.XrayTemplateGeodata:  {},
	}
}

func variablePreviewRefAlias(binding *node.ServerXrayTemplate, index int) string {
	if binding == nil || binding.Template == nil {
		return ""
	}
	if binding.Alias != "" {
		return binding.Alias
	}
	return fmt.Sprintf("%s_%d", binding.Template.Type, index+1)
}

func setVariablePreviewRef(ref xraytemplate.Ref, templateType string, alias string, variables xraytemplate.ConfigObject) {
	if ref[templateType] == nil {
		ref[templateType] = map[string]xraytemplate.ConfigObject{}
	}
	ref[templateType][alias] = variables
}

func variablePreviewRefObject(server *node.Server, alias string, merged xraytemplate.ConfigObject) xraytemplate.ConfigObject {
	vars := mergeVariablePreviewObjects(merged)
	if _, ok := vars["tag"]; !ok && alias != "" {
		vars["tag"] = alias
	}
	if server != nil {
		vars["server_id"] = server.Id
		vars["server_name"] = server.Name
		vars["server_address"] = server.Address
	}
	vars["alias"] = alias
	return vars
}

func mergeVariablePreviewObjects(items ...xraytemplate.ConfigObject) xraytemplate.ConfigObject {
	capHint := 0
	for _, item := range items {
		capHint += len(item)
	}
	merged := make(xraytemplate.ConfigObject, capHint)
	for _, item := range items {
		for key, value := range item {
			merged[key] = value
		}
	}
	return merged
}

func previewRawObject(value xraytemplate.ConfigObject) json.RawMessage {
	raw, err := xraytemplate.RawConfigValue(value)
	if err != nil || raw == nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

func buildXrayConfigResponse(srv *node.Server, list []*node.ServerXrayTemplate) (*types.QueryServerXrayConfigResponse, error) {
	raw, err := renderValidatedRawXrayConfigBundle(srv, list)
	if err != nil {
		return nil, err
	}
	return buildXrayConfigResponseFromRaw(list, raw)
}

func buildPreviewXrayConfigResponse(srv *node.Server, list []*node.ServerXrayTemplate) (*types.QueryServerXrayConfigResponse, []xraytemplate.RenderedSubscriptionProxy, error) {
	buckets, proxies, err := xraytemplate.RenderConfigBucketsAndSubscriptionProxies(srv, list, xraytemplate.UserInfo{
		Password: "preview-user",
		UUID:     "00000000-0000-0000-0000-000000000000",
	})
	if err != nil {
		return nil, nil, err
	}
	merged := buckets.Merged()
	if err := merged.ValidateRoutingReferences(); err != nil {
		return nil, nil, errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, err.Error()), "validate routing references")
	}
	raw, err := merged.Raw()
	if err != nil {
		return nil, nil, err
	}
	resp, err := buildXrayConfigResponseFromRaw(list, raw)
	if err != nil {
		return nil, nil, err
	}
	return resp, proxies, nil
}

func buildXrayConfigResponseFromRaw(list []*node.ServerXrayTemplate, raw xraytemplate.RawConfigBundle) (*types.QueryServerXrayConfigResponse, error) {
	resp := &types.QueryServerXrayConfigResponse{
		Templates: make([]types.XrayTemplate, 0, countRenderableXrayTemplates(list)),
	}
	for _, item := range list {
		if item == nil || item.Template == nil {
			continue
		}
		template, err := xrayTemplateToTypes(item.Template, true)
		if err != nil {
			return nil, err
		}
		resp.Templates = append(resp.Templates, template)
	}
	applyRawXrayConfigBundle(resp, raw)
	return resp, nil
}

func countRenderableXrayTemplates(list []*node.ServerXrayTemplate) int {
	count := 0
	for _, item := range list {
		if item == nil || item.Template == nil {
			continue
		}
		count++
	}
	return count
}

func renderValidatedRawXrayConfigBundle(srv *node.Server, list []*node.ServerXrayTemplate) (xraytemplate.RawConfigBundle, error) {
	merged, err := xraytemplate.RenderMergedConfigBundle(srv, list)
	if err != nil {
		return xraytemplate.RawConfigBundle{}, err
	}
	if err := merged.ValidateRoutingReferences(); err != nil {
		return xraytemplate.RawConfigBundle{}, errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, err.Error()), "validate routing references")
	}
	return merged.Raw()
}

func applyRawXrayConfigBundle(resp *types.QueryServerXrayConfigResponse, raw xraytemplate.RawConfigBundle) {
	resp.Inbounds = rawConfigListOrEmpty(raw.Inbounds)
	resp.Outbounds = rawConfigListOrEmpty(raw.Outbounds)
	resp.DNS = raw.DNS
	resp.Routing = raw.Routing
	resp.Geodata = raw.Geodata
}

func rawConfigListOrEmpty(values []json.RawMessage) []json.RawMessage {
	if values == nil {
		return []json.RawMessage{}
	}
	return values
}
