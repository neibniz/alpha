package server

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

var (
	xrayTemplateEmptyObjectRaw = json.RawMessage("{}")
	xrayTemplateRawNull        = []byte("null")
)

const smallXrayTemplateRawCacheThreshold = 8

func normalizeXrayTemplateConfig(value json.RawMessage) (string, error) {
	raw, err := xrayTemplateConfigRaw(value)
	if err != nil {
		return "", err
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return "", errors.Wrapf(
			xerr.NewErrCodeMsg(xerr.InvalidParams, "xray template config must be a JSON object"),
			"decode xray template object: %v",
			err,
		)
	}
	return string(raw), nil
}

func normalizeXrayTemplateRaw(value json.RawMessage) (string, error) {
	raw, err := xrayTemplateConfigRaw(value)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func xrayTemplateConfigRaw(value json.RawMessage) (json.RawMessage, error) {
	data := value
	if data == nil {
		data = xrayTemplateEmptyObjectRaw
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, xrayTemplateRawNull) {
		trimmed = xrayTemplateEmptyObjectRaw
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, trimmed); err != nil {
		return nil, errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "invalid xray template config"), "compact xray template config: %v", err)
	}
	return json.RawMessage(compacted.Bytes()), nil
}

func validateXrayTemplateType(value string) error {
	switch strings.ToLower(value) {
	case node.XrayTemplateInbound, node.XrayTemplateOutbound, node.XrayTemplateDNS, node.XrayTemplateRouting, node.XrayTemplateGeodata:
		return nil
	default:
		return errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "invalid xray template type"), "invalid xray template type: %s", value)
	}
}

func normalizeXrayTemplateIdentity(name, description string) (string, string, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if count := utf8.RuneCountInString(name); count == 0 || count > 100 {
		return "", "", xerr.NewErrCodeMsg(xerr.InvalidParams, "template name must contain 1 to 100 characters")
	}
	if utf8.RuneCountInString(description) > 255 {
		return "", "", xerr.NewErrCodeMsg(xerr.InvalidParams, "template description must not exceed 255 characters")
	}
	return name, description, nil
}

func xrayTemplateToTypes(data *node.XrayTemplate, includeConfig bool) (types.XrayTemplate, error) {
	resp := xrayTemplateBaseToTypes(data)
	if !includeConfig {
		return resp, nil
	}
	return xrayTemplateToTypesWithConfigCache(data, resp, nil)
}

func xrayTemplateBaseToTypes(data *node.XrayTemplate) types.XrayTemplate {
	return types.XrayTemplate{
		Id:          data.Id,
		Name:        data.Name,
		Type:        data.Type,
		Description: data.Description,
		Enabled:     boolPtrOrDefault(data.Enabled, true),
		CreatedAt:   timex.UnixMilliOrZero(data.CreatedAt),
		UpdatedAt:   timex.UnixMilliOrZero(data.UpdatedAt),
	}
}

func xrayTemplateListItemToTypes(data *node.XrayTemplate) types.XrayTemplateListItem {
	if data == nil {
		return types.XrayTemplateListItem{}
	}
	return types.XrayTemplateListItem{
		Id:                       data.Id,
		Name:                     data.Name,
		Type:                     data.Type,
		Description:              data.Description,
		Enabled:                  boolPtrOrDefault(data.Enabled, true),
		BindingCount:             data.BindingCount,
		SubscriptionBindingCount: data.SubscriptionBindingCount,
		UpdatedAt:                timex.UnixMilliOrZero(data.UpdatedAt),
	}
}

func xrayTemplateListRowsToTypes(rows []*node.XrayTemplate) []types.XrayTemplateListItem {
	if len(rows) == 0 {
		return []types.XrayTemplateListItem{}
	}
	collector := viewutil.NewRowsCollector[types.XrayTemplateListItem](len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(xrayTemplateListItemToTypes(row))
	}
	return collector.Rows()
}

func xrayTemplateCatalogItemToTypes(data *node.XrayTemplate, rawCache *xrayTemplateRawCache) (types.XrayTemplateCatalogItem, error) {
	if data == nil {
		return types.XrayTemplateCatalogItem{}, nil
	}
	cfg, err := rawCache.Raw(data.Config)
	if err != nil {
		return types.XrayTemplateCatalogItem{}, err
	}
	variablesSchema, err := rawCache.Raw(data.VariablesSchema)
	if err != nil {
		return types.XrayTemplateCatalogItem{}, err
	}
	defaultVariables, err := rawCache.Raw(data.DefaultVariables)
	if err != nil {
		return types.XrayTemplateCatalogItem{}, err
	}
	subscriptionMeta, err := rawCache.Raw(data.SubscriptionMeta)
	if err != nil {
		return types.XrayTemplateCatalogItem{}, err
	}
	return types.XrayTemplateCatalogItem{
		Id:               data.Id,
		Name:             data.Name,
		Type:             data.Type,
		Config:           cfg,
		ConfigTemplate:   data.ConfigTemplate,
		VariablesSchema:  variablesSchema,
		DefaultVariables: defaultVariables,
		SubscriptionMeta: subscriptionMeta,
		Description:      data.Description,
		Enabled:          boolPtrOrDefault(data.Enabled, true),
	}, nil
}

func xrayTemplateCatalogRowsToTypes(rows []*node.XrayTemplate) ([]types.XrayTemplateCatalogItem, error) {
	if len(rows) == 0 {
		return []types.XrayTemplateCatalogItem{}, nil
	}
	collector := viewutil.NewRowsCollector[types.XrayTemplateCatalogItem](len(rows))
	rawCache := newXrayTemplateRawCache(len(rows) * 4)
	for _, row := range rows {
		if row == nil {
			continue
		}
		item, err := xrayTemplateCatalogItemToTypes(row, rawCache)
		if err != nil {
			return nil, err
		}
		collector.Add(item)
	}
	return collector.Rows(), nil
}

func xrayTemplateRowsToTypes(rows []*node.XrayTemplate, includeConfig bool) ([]types.XrayTemplate, error) {
	if len(rows) == 0 {
		return []types.XrayTemplate{}, nil
	}
	collector := viewutil.NewRowsCollector[types.XrayTemplate](len(rows))
	var rawCache *xrayTemplateRawCache
	if includeConfig {
		rawCache = newXrayTemplateRawCache(len(rows) * 4)
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		item, err := xrayTemplateToTypesWithCache(row, includeConfig, rawCache)
		if err != nil {
			return nil, err
		}
		collector.Add(item)
	}
	return collector.Rows(), nil
}

func xrayTemplateToTypesWithCache(data *node.XrayTemplate, includeConfig bool, rawCache *xrayTemplateRawCache) (types.XrayTemplate, error) {
	resp := xrayTemplateBaseToTypes(data)
	if !includeConfig {
		return resp, nil
	}
	return xrayTemplateToTypesWithConfigCache(data, resp, rawCache)
}

func xrayTemplateToTypesWithConfig(data *node.XrayTemplate, resp types.XrayTemplate) (types.XrayTemplate, error) {
	return xrayTemplateToTypesWithConfigCache(data, resp, nil)
}

func xrayTemplateToTypesWithConfigCache(data *node.XrayTemplate, resp types.XrayTemplate, rawCache *xrayTemplateRawCache) (types.XrayTemplate, error) {
	cfg, err := rawCache.Raw(data.Config)
	if err != nil {
		return types.XrayTemplate{}, err
	}
	variablesSchema, err := rawCache.Raw(data.VariablesSchema)
	if err != nil {
		return types.XrayTemplate{}, err
	}
	defaultVariables, err := rawCache.Raw(data.DefaultVariables)
	if err != nil {
		return types.XrayTemplate{}, err
	}
	subscriptionMeta, err := rawCache.Raw(data.SubscriptionMeta)
	if err != nil {
		return types.XrayTemplate{}, err
	}
	resp.Config = cfg
	resp.ConfigTemplate = data.ConfigTemplate
	resp.VariablesSchema = variablesSchema
	resp.DefaultVariables = defaultVariables
	resp.SubscriptionMeta = subscriptionMeta
	return resp, nil
}

func xrayTemplateBindingTemplateToTypes(data *node.XrayTemplate) (types.XrayTemplate, error) {
	return xrayTemplateBindingTemplateToTypesWithCache(data, nil)
}

func xrayTemplateBindingTemplateToTypesWithCache(data *node.XrayTemplate, rawCache *xrayTemplateRawCache) (types.XrayTemplate, error) {
	cfg, err := rawCache.Raw(data.Config)
	if err != nil {
		return types.XrayTemplate{}, err
	}
	defaultVariables, err := rawCache.Raw(data.DefaultVariables)
	if err != nil {
		return types.XrayTemplate{}, err
	}
	return types.XrayTemplate{
		Id:               data.Id,
		Name:             data.Name,
		Type:             data.Type,
		Config:           cfg,
		DefaultVariables: defaultVariables,
		Enabled:          boolPtrOrDefault(data.Enabled, true),
	}, nil
}

func serverXrayTemplateRowsToTypes(rows []*node.ServerXrayTemplate, includeTemplate bool) ([]types.ServerXrayTemplate, error) {
	if len(rows) == 0 {
		return []types.ServerXrayTemplate{}, nil
	}
	collector := viewutil.NewRowsCollector[types.ServerXrayTemplate](len(rows))
	rawCache := newXrayTemplateRawCache(len(rows) * serverXrayTemplateRawFieldsPerRow(includeTemplate))
	for _, row := range rows {
		if row == nil {
			continue
		}
		var template *types.XrayTemplate
		if includeTemplate && row.Template != nil {
			templateInfo, err := xrayTemplateBindingTemplateToTypesWithCache(row.Template, rawCache)
			if err != nil {
				return nil, err
			}
			template = &templateInfo
		}
		binding, err := serverXrayTemplateToTypesWithCache(row, template, rawCache)
		if err != nil {
			return nil, err
		}
		collector.Add(binding)
	}
	return collector.Rows(), nil
}

func serverXrayTemplateRawFieldsPerRow(includeTemplate bool) int {
	if includeTemplate {
		return 4
	}
	return 2
}

func serverXrayTemplateToTypes(data *node.ServerXrayTemplate, template *types.XrayTemplate) (types.ServerXrayTemplate, error) {
	return serverXrayTemplateToTypesWithCache(data, template, nil)
}

func serverXrayTemplateToTypesWithCache(data *node.ServerXrayTemplate, template *types.XrayTemplate, rawCache *xrayTemplateRawCache) (types.ServerXrayTemplate, error) {
	variables, err := rawCache.Raw(data.Variables)
	if err != nil {
		return types.ServerXrayTemplate{}, err
	}
	subscriptionVariables, err := rawCache.Raw(data.SubscriptionVariables)
	if err != nil {
		return types.ServerXrayTemplate{}, err
	}
	return types.ServerXrayTemplate{
		Id:                    data.Id,
		ServerId:              data.ServerId,
		TemplateId:            data.TemplateId,
		Alias:                 data.Alias,
		Variables:             variables,
		SubscriptionEnabled:   boolPtrOrDefault(data.SubscriptionEnabled, true),
		SubscriptionName:      data.SubscriptionName,
		SubscriptionVariables: subscriptionVariables,
		Sort:                  data.Sort,
		Enabled:               boolPtrOrDefault(data.Enabled, true),
		Template:              template,
		CreatedAt:             timex.UnixMilliOrZero(data.CreatedAt),
		UpdatedAt:             timex.UnixMilliOrZero(data.UpdatedAt),
	}, nil
}

func xrayTemplateRawFromModel(value string) (json.RawMessage, error) {
	return xrayTemplateRawFromTrimmedModel(strings.TrimSpace(value))
}

func xrayTemplateRawFromTrimmedModel(value string) (json.RawMessage, error) {
	if value == "" {
		return xrayTemplateEmptyObjectRaw, nil
	}
	raw := []byte(value)
	if !json.Valid(raw) {
		return nil, errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "invalid xray template config"), "invalid stored xray template JSON")
	}
	return json.RawMessage(raw), nil
}

type xrayTemplateRawCacheEntry struct {
	raw   string
	value json.RawMessage
	err   error
}

type xrayTemplateRawCache struct {
	rows    []xrayTemplateRawCacheEntry
	index   map[string]int
	capHint int
}

func newXrayTemplateRawCache(capHint int) *xrayTemplateRawCache {
	if capHint <= 1 {
		return nil
	}
	return &xrayTemplateRawCache{capHint: capHint}
}

func (c *xrayTemplateRawCache) Raw(value string) (json.RawMessage, error) {
	key := strings.TrimSpace(value)
	if c == nil {
		return xrayTemplateRawFromTrimmedModel(key)
	}
	if c.index != nil {
		if pos, ok := c.index[key]; ok {
			entry := c.rows[pos]
			return entry.value, entry.err
		}
		return c.add(key)
	}
	for _, row := range c.rows {
		if row.raw == key {
			return row.value, row.err
		}
	}
	return c.add(key)
}

func (c *xrayTemplateRawCache) add(key string) (json.RawMessage, error) {
	value, err := xrayTemplateRawFromTrimmedModel(key)
	c.ensureRows()
	c.rows = append(c.rows, xrayTemplateRawCacheEntry{raw: key, value: value, err: err})
	if c.index != nil {
		c.index[key] = len(c.rows) - 1
	} else if len(c.rows) > smallXrayTemplateRawCacheThreshold {
		nextCap := adminServerLookupCap(len(c.rows), c.capHint)
		c.index = make(map[string]int, nextCap)
		if nextCap > cap(c.rows) {
			rows := make([]xrayTemplateRawCacheEntry, len(c.rows), nextCap)
			copy(rows, c.rows)
			c.rows = rows
		}
		for i, row := range c.rows {
			c.index[row.raw] = i
		}
	}
	return value, err
}

func (c *xrayTemplateRawCache) ensureRows() {
	if c.rows != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = smallXrayTemplateRawCacheThreshold
	} else if capHint > smallXrayTemplateRawCacheThreshold+1 {
		capHint = smallXrayTemplateRawCacheThreshold + 1
	}
	c.rows = make([]xrayTemplateRawCacheEntry, 0, capHint)
}

func boolPtrOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
