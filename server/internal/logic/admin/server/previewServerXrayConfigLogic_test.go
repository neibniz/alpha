package server

import (
	"encoding/json"
	"testing"

	"github.com/neibniz/alpha/server/adapter"
	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/internal/xraytemplate"
)

func TestPreviewXrayConfigBucketsPreallocateByTemplateType(t *testing.T) {
	list := []*node.ServerXrayTemplate{
		{Template: &node.XrayTemplate{Type: node.XrayTemplateInbound}},
		{Template: &node.XrayTemplate{Type: node.XrayTemplateOutbound}},
		{Template: &node.XrayTemplate{Type: node.XrayTemplateOutbound}},
		{Template: &node.XrayTemplate{Type: node.XrayTemplateRouting}},
		{Template: &node.XrayTemplate{Type: "unknown"}},
		nil,
		{Template: nil},
	}

	templateCount := countRenderableXrayTemplates(list)
	if templateCount != 5 {
		t.Fatalf("preview xray template count = %d, want 5", templateCount)
	}
	counts := xraytemplate.CountConfigBuckets(list)
	if counts.Inbounds != 1 || counts.Outbounds != 2 || counts.Routing != 1 {
		t.Fatalf("preview xray config bucket counts = %#v", counts)
	}
	if counts.DNS != 0 || counts.Geodata != 0 {
		t.Fatalf("unused preview xray config bucket counts = %#v", counts)
	}
	buckets := xraytemplate.NewConfigBucketsWithCounts(counts)
	if cap(buckets.Inbounds) != 1 || cap(buckets.Outbounds) != 2 || cap(buckets.Routing) != 1 {
		t.Fatalf("preview bucket caps = in:%d out:%d routing:%d", cap(buckets.Inbounds), cap(buckets.Outbounds), cap(buckets.Routing))
	}
	if cap(buckets.DNS) != 0 || cap(buckets.Geodata) != 0 {
		t.Fatalf("unused preview bucket caps = dns:%d geodata:%d", cap(buckets.DNS), cap(buckets.Geodata))
	}
}

func TestApplyRawXrayConfigBundleKeepsEmptyArrayShape(t *testing.T) {
	resp := &types.QueryServerXrayConfigResponse{}
	applyRawXrayConfigBundle(resp, xraytemplate.RawConfigBundle{})
	if resp.Inbounds == nil || resp.Outbounds == nil {
		t.Fatalf("preview raw config arrays should be non-nil: in=%#v out=%#v", resp.Inbounds, resp.Outbounds)
	}
}

func TestRawSubscriptionProxyRowsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	proxies := make([]xraytemplate.RenderedSubscriptionProxy, viewutil.SmallRowsCapacityThreshold+20)
	proxies[0] = xraytemplate.RenderedSubscriptionProxy{Proxy: &adapter.Proxy{Name: "one", Server: "node.example.com", Type: "vless"}}

	got, err := rawSubscriptionProxyRows(proxies)
	if err != nil {
		t.Fatalf("raw subscription proxy rows error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("raw subscription proxy rows len = %d, want 1", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy raw subscription proxy rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestRawSubscriptionProxyRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	proxies := make([]xraytemplate.RenderedSubscriptionProxy, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		proxies[index] = xraytemplate.RenderedSubscriptionProxy{Proxy: &adapter.Proxy{Name: "one", Server: "node.example.com", Type: "vless"}}
	}

	got, err := rawSubscriptionProxyRows(proxies)
	if err != nil {
		t.Fatalf("raw subscription proxy rows error = %v", err)
	}
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("raw subscription proxy rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("raw subscription proxy rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestBuildVariablePreviewMergesVariableLayersOnce(t *testing.T) {
	binding := &node.ServerXrayTemplate{
		Alias:      "main",
		TemplateId: 7,
		Sort:       3,
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Name: "vless",
			Config: `{
				"tag": "from-config",
				"port": 443,
				"protocol": "vless",
				"streamSettings": {"network": "grpc"}
			}`,
			DefaultVariables: `{
				"tag": "from-default",
				"port": 80,
				"network": "tcp",
				"default_only": "yes"
			}`,
		},
		Variables:             `{"port": 8443, "binding_only": true}`,
		SubscriptionVariables: `{"security": "tls", "sub_only": "ok"}`,
	}
	preview := buildVariablePreview(&node.Server{Id: 9, Name: "edge", Address: "node.example.com"}, []*node.ServerXrayTemplate{binding})
	got, ok := preview[node.XrayTemplateInbound+".main"]
	if !ok {
		t.Fatalf("preview keys = %#v, want inbound.main", preview)
	}

	merged := decodePreviewRaw(t, got.MergedVariables)
	if merged["tag"] != "from-config" || merged["network"] != "grpc" || merged["port"] != float64(8443) {
		t.Fatalf("merged variables = %#v, want config values overridden by binding values", merged)
	}
	if merged["default_only"] != "yes" || merged["binding_only"] != true {
		t.Fatalf("merged variables missing layer-specific values: %#v", merged)
	}

	subscriptionMerged := decodePreviewRaw(t, got.SubscriptionMergedVariables)
	if subscriptionMerged["security"] != "tls" || subscriptionMerged["sub_only"] != "ok" || subscriptionMerged["port"] != float64(8443) {
		t.Fatalf("subscription merged variables = %#v", subscriptionMerged)
	}

	refVars := decodePreviewRaw(t, got.Ref)
	if refVars["alias"] != "main" || refVars["server_name"] != "edge" || refVars["tag"] != "from-config" {
		t.Fatalf("ref variables = %#v", refVars)
	}
}

func TestBuildVariablePreviewRowsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	bindings := make([]*node.ServerXrayTemplate, viewutil.SmallRowsCapacityThreshold+20)
	bindings[0] = previewVariableBinding()

	got, _ := buildVariablePreviewRows(&node.Server{Id: 9, Name: "edge", Address: "node.example.com"}, bindings)
	if len(got) != 1 {
		t.Fatalf("variable preview rows len = %d, want 1", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy variable preview rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestBuildVariablePreviewRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	bindings := make([]*node.ServerXrayTemplate, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		bindings[index] = previewVariableBinding()
	}

	got, _ := buildVariablePreviewRows(&node.Server{Id: 9, Name: "edge", Address: "node.example.com"}, bindings)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("variable preview rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("variable preview rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestBuildVariablePreviewUsesRenderAliasForDefaultAliasRef(t *testing.T) {
	binding := &node.ServerXrayTemplate{
		Template: &node.XrayTemplate{
			Type:             node.XrayTemplateInbound,
			Name:             "vless",
			Config:           `{"tag":"from-config","port":443}`,
			DefaultVariables: `{}`,
		},
		Variables: "{}",
	}

	preview := buildVariablePreview(&node.Server{Id: 9, Name: "edge", Address: "node.example.com"}, []*node.ServerXrayTemplate{binding})
	got, ok := preview[node.XrayTemplateInbound+".vless"]
	if !ok {
		t.Fatalf("preview keys = %#v, want inbound.vless", preview)
	}
	refVars := decodePreviewRaw(t, got.Ref)
	if refVars["alias"] != "inbound_1" || refVars["server_name"] != "edge" || refVars["tag"] != "from-config" {
		t.Fatalf("default binding alias ref variables = %#v", refVars)
	}
}

func previewVariableBinding() *node.ServerXrayTemplate {
	return &node.ServerXrayTemplate{
		Template: &node.XrayTemplate{
			Type:             node.XrayTemplateInbound,
			Name:             "vless",
			Config:           `{"tag":"from-config","port":443}`,
			DefaultVariables: `{}`,
		},
		Variables: "{}",
	}
}

func decodePreviewRaw(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode preview raw %s: %v", raw, err)
	}
	return decoded
}
