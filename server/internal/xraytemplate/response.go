package xraytemplate

import (
	"encoding/json"

	"github.com/neibniz/alpha/server/internal/model/node"
)

type ConfigObject = map[string]any

type ConfigBucketCounts struct {
	Inbounds  int
	Outbounds int
	DNS       int
	Routing   int
	Geodata   int
}

func (c *ConfigBucketCounts) Add(templateType string) {
	switch templateType {
	case node.XrayTemplateInbound:
		c.Inbounds++
	case node.XrayTemplateOutbound:
		c.Outbounds++
	case node.XrayTemplateDNS:
		c.DNS++
	case node.XrayTemplateRouting:
		c.Routing++
	case node.XrayTemplateGeodata:
		c.Geodata++
	}
}

type ConfigBuckets struct {
	Inbounds  []ConfigObject
	Outbounds []ConfigObject
	DNS       []ConfigObject
	Routing   []ConfigObject
	Geodata   []ConfigObject
}

type MergedConfigBundle struct {
	Inbounds  []ConfigObject
	Outbounds []ConfigObject
	DNS       ConfigObject
	Routing   ConfigObject
	Geodata   ConfigObject
}

type RawConfigBundle struct {
	Inbounds  []json.RawMessage
	Outbounds []json.RawMessage
	DNS       json.RawMessage
	Routing   json.RawMessage
	Geodata   json.RawMessage
}

func CountConfigBuckets(list []*node.ServerXrayTemplate) ConfigBucketCounts {
	var counts ConfigBucketCounts
	for _, item := range list {
		if item == nil || item.Template == nil {
			continue
		}
		switch item.Template.Type {
		case node.XrayTemplateInbound, node.XrayTemplateOutbound, node.XrayTemplateDNS, node.XrayTemplateRouting, node.XrayTemplateGeodata:
			counts.Add(item.Template.Type)
		}
	}
	return counts
}

func NewConfigBuckets(list []*node.ServerXrayTemplate) ConfigBuckets {
	return NewConfigBucketsWithCounts(CountConfigBuckets(list))
}

func NewConfigBucketsWithCounts(counts ConfigBucketCounts) ConfigBuckets {
	return ConfigBuckets{
		Inbounds:  make([]ConfigObject, 0, counts.Inbounds),
		Outbounds: make([]ConfigObject, 0, counts.Outbounds),
		DNS:       make([]ConfigObject, 0, counts.DNS),
		Routing:   make([]ConfigObject, 0, counts.Routing),
		Geodata:   make([]ConfigObject, 0, counts.Geodata),
	}
}

func RenderConfigBuckets(server *node.Server, list []*node.ServerXrayTemplate) (ConfigBuckets, error) {
	plan := newRenderBindingPlan(server, list)
	buckets := NewConfigBucketsWithCounts(plan.counts)
	for _, entry := range plan.entries {
		cfg, err := renderBindingConfigWithVariables(server, entry.binding, plan.ref, entry.variables)
		if err != nil {
			return ConfigBuckets{}, err
		}
		buckets.Add(entry.binding.Template.Type, cfg)
	}
	return buckets, nil
}

func RenderMergedConfigBundle(server *node.Server, list []*node.ServerXrayTemplate) (MergedConfigBundle, error) {
	buckets, err := RenderConfigBuckets(server, list)
	if err != nil {
		return MergedConfigBundle{}, err
	}
	return buckets.Merged(), nil
}

func (b *ConfigBuckets) Add(templateType string, cfg ConfigObject) {
	switch templateType {
	case node.XrayTemplateInbound:
		b.Inbounds = append(b.Inbounds, cfg)
	case node.XrayTemplateOutbound:
		b.Outbounds = append(b.Outbounds, cfg)
	case node.XrayTemplateDNS:
		b.DNS = append(b.DNS, cfg)
	case node.XrayTemplateRouting:
		b.Routing = append(b.Routing, cfg)
	case node.XrayTemplateGeodata:
		b.Geodata = append(b.Geodata, cfg)
	}
}

func (b ConfigBuckets) Merged() MergedConfigBundle {
	return MergedConfigBundle{
		Inbounds:  b.Inbounds,
		Outbounds: b.Outbounds,
		DNS:       MergeDNS(b.DNS),
		Routing:   MergeRouting(b.Routing),
		Geodata:   MergeGeodata(b.Geodata),
	}
}

func (b MergedConfigBundle) ValidateRoutingReferences() error {
	return ValidateRoutingReferences(b.Outbounds, b.Routing)
}

func (b MergedConfigBundle) Raw() (RawConfigBundle, error) {
	rawInbounds, err := RawConfigList(b.Inbounds)
	if err != nil {
		return RawConfigBundle{}, err
	}
	rawOutbounds, err := RawConfigList(b.Outbounds)
	if err != nil {
		return RawConfigBundle{}, err
	}
	rawDNS, err := RawConfigValue(b.DNS)
	if err != nil {
		return RawConfigBundle{}, err
	}
	rawRouting, err := RawConfigValue(b.Routing)
	if err != nil {
		return RawConfigBundle{}, err
	}
	rawGeodata, err := RawConfigValue(b.Geodata)
	if err != nil {
		return RawConfigBundle{}, err
	}
	return RawConfigBundle{
		Inbounds:  rawInbounds,
		Outbounds: rawOutbounds,
		DNS:       rawDNS,
		Routing:   rawRouting,
		Geodata:   rawGeodata,
	}, nil
}

func RawConfigList(items []ConfigObject) ([]json.RawMessage, error) {
	if len(items) == 0 {
		return []json.RawMessage{}, nil
	}
	result := make([]json.RawMessage, len(items))
	for index, item := range items {
		raw, err := RawConfigValue(item)
		if err != nil {
			return nil, err
		}
		result[index] = raw
	}
	return result, nil
}

func RawConfigValue(value any) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}
	if raw, ok := value.(json.RawMessage); ok {
		if len(raw) == 0 {
			return nil, nil
		}
		return append(json.RawMessage(nil), raw...), nil
	}
	if data, ok := value.(ConfigObject); ok && data == nil {
		return nil, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}
