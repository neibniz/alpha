package server

import (
	"context"
	"strings"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type xrayBindingBuildOptions struct {
	skipDisabled bool
}

var xrayTemplateBindingColumns = []string{
	"id",
	"name",
	"type",
	"config",
	"config_template",
	"variables_schema",
	"default_variables",
	"subscription_meta",
	"description",
	"enabled",
	"created_at",
	"updated_at",
}

const smallXrayTemplateBindingSetThreshold = 8

func buildServerXrayTemplateBindings(ctx context.Context, db *gorm.DB, serverID int64, reqBindings []types.ServerXrayTemplateBinding, opts xrayBindingBuildOptions) ([]*node.ServerXrayTemplate, []*node.ServerXrayTemplate, error) {
	templateIDs, err := collectXrayTemplateIDs(reqBindings)
	if err != nil {
		return nil, nil, err
	}
	templatesByID, err := queryXrayTemplatesByID(ctx, db, templateIDs)
	if err != nil {
		return nil, nil, err
	}

	bindings := viewutil.NewRowsCollector[*node.ServerXrayTemplate](len(reqBindings))
	enabledBindings := viewutil.NewRowsCollector[*node.ServerXrayTemplate](len(reqBindings))
	var aliases xrayTemplateAliasSet
	for i, item := range reqBindings {
		tpl := templatesByID[item.TemplateId]
		if tpl == nil {
			return nil, nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query xray template error: template %d not found", item.TemplateId)
		}
		enabled := true
		if item.Enabled != nil {
			enabled = *item.Enabled
		}
		if opts.skipDisabled && !enabled {
			continue
		}
		alias := strings.TrimSpace(item.Alias)
		if alias == "" {
			alias = tpl.Name
		}
		aliasKey := tpl.Type + "." + alias
		if !aliases.Add(aliasKey) {
			return nil, nil, errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "xray template alias must be unique per type"), "duplicate xray template alias: %s", aliasKey)
		}
		sort := item.Sort
		if sort == 0 {
			sort = i + 1
		}
		variables, err := normalizeXrayTemplateConfig(item.Variables)
		if err != nil {
			return nil, nil, err
		}
		subscriptionVariables, err := normalizeXrayTemplateConfig(item.SubscriptionVariables)
		if err != nil {
			return nil, nil, err
		}
		subscriptionEnabled := true
		if item.SubscriptionEnabled != nil {
			subscriptionEnabled = *item.SubscriptionEnabled
		}
		binding := &node.ServerXrayTemplate{
			ServerId:              serverID,
			TemplateId:            item.TemplateId,
			Template:              tpl,
			Alias:                 alias,
			Variables:             variables,
			SubscriptionEnabled:   &subscriptionEnabled,
			SubscriptionName:      item.SubscriptionName,
			SubscriptionVariables: subscriptionVariables,
			Sort:                  sort,
			Enabled:               &enabled,
		}
		bindings.Add(binding)
		if enabled {
			enabledBindings.Add(binding)
		}
	}
	return bindings.Rows(), enabledBindings.Rows(), nil
}

func collectXrayTemplateIDs(bindings []types.ServerXrayTemplateBinding) ([]int64, error) {
	var ids xrayTemplateIDSet
	for _, item := range bindings {
		if item.TemplateId == 0 {
			return nil, errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "template id is required"), "template id is required")
		}
		ids.Add(item.TemplateId)
	}
	return ids.Values(), nil
}

type xrayTemplateIDSet struct {
	values []int64
	index  map[int64]struct{}
}

func (s *xrayTemplateIDSet) Add(id int64) bool {
	if id <= 0 {
		return false
	}
	if s.index != nil {
		if _, ok := s.index[id]; ok {
			return false
		}
		s.index[id] = struct{}{}
		s.values = append(s.values, id)
		return true
	}
	for _, existing := range s.values {
		if existing == id {
			return false
		}
	}
	s.values = append(s.values, id)
	if len(s.values) > smallXrayTemplateBindingSetThreshold {
		s.index = make(map[int64]struct{}, len(s.values))
		for _, existing := range s.values {
			s.index[existing] = struct{}{}
		}
	}
	return true
}

func (s *xrayTemplateIDSet) Values() []int64 {
	if len(s.values) == 0 {
		return []int64{}
	}
	return s.values
}

type xrayTemplateAliasSet struct {
	values []string
	index  map[string]struct{}
}

func (s *xrayTemplateAliasSet) Add(alias string) bool {
	if alias == "" {
		return false
	}
	if s.index != nil {
		if _, ok := s.index[alias]; ok {
			return false
		}
		s.index[alias] = struct{}{}
		s.values = append(s.values, alias)
		return true
	}
	for _, existing := range s.values {
		if existing == alias {
			return false
		}
	}
	s.values = append(s.values, alias)
	if len(s.values) > smallXrayTemplateBindingSetThreshold {
		s.index = make(map[string]struct{}, len(s.values))
		for _, existing := range s.values {
			s.index[existing] = struct{}{}
		}
	}
	return true
}

func queryXrayTemplatesByID(ctx context.Context, db *gorm.DB, templateIDs []int64) (map[int64]*node.XrayTemplate, error) {
	if len(templateIDs) == 0 {
		return nil, nil
	}
	result := make(map[int64]*node.XrayTemplate, len(templateIDs))
	var templates []*node.XrayTemplate
	if err := db.WithContext(ctx).
		Model(&node.XrayTemplate{}).
		Select(xrayTemplateBindingColumns).
		Where("id IN ?", templateIDs).
		Find(&templates).Error; err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query xray template error: %v", err)
	}
	for _, tpl := range templates {
		if tpl != nil {
			result[tpl.Id] = tpl
		}
	}
	for _, id := range templateIDs {
		if _, ok := result[id]; !ok {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query xray template error: template %d not found", id)
		}
	}
	return result, nil
}
