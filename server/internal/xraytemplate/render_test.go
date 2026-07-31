package xraytemplate

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestUint16ValueRejectsTruncatedPorts(t *testing.T) {
	for _, value := range []any{-1, int64(65_536), 44_000.5, json.Number("70000"), "70000"} {
		if got := uint16Value(ConfigObject{"port": value}, "port"); got != 0 {
			t.Fatalf("uint16Value(%v) = %d, want 0", value, got)
		}
	}
	if got := uint16Value(ConfigObject{"port": int64(443)}, "port"); got != 443 {
		t.Fatalf("uint16Value(443) = %d", got)
	}
}

func TestRenderBindingConfigRejectsInboundRealityWithoutTarget(t *testing.T) {
	enabled := true
	_, err := RenderBindingConfig(&node.Server{Id: 1, Name: "hk", Address: "127.0.0.1"}, &node.ServerXrayTemplate{
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Config: `{
				"tag": "hk-in",
				"port": 20003,
				"protocol": "vless",
				"settings": {"decryption": "none"},
				"streamSettings": {
					"network": "grpc",
					"security": "reality",
					"realitySettings": {
						"privateKey": "X3rxqzJ2ki0i6v22lwyM-NNQeGctkKCdv1C6lm3GjxA",
						"serverNames": ["ebay.com"],
						"shortIds": ["a3ec67"]
					}
				}
			}`,
			DefaultVariables: "{}",
			Enabled:          &enabled,
		},
		Variables: "{}",
		Enabled:   &enabled,
	}, BuildRef(&node.Server{Id: 1, Name: "hk", Address: "127.0.0.1"}, nil))
	if err == nil || !strings.Contains(err.Error(), "realitySettings.target") {
		t.Fatalf("expected target validation error, got %v", err)
	}
}

func TestRenderBindingConfigAcceptsInboundRealityWithTarget(t *testing.T) {
	enabled := true
	_, err := RenderBindingConfig(&node.Server{Id: 1, Name: "hk", Address: "127.0.0.1"}, &node.ServerXrayTemplate{
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Config: `{
				"tag": "hk-in",
				"port": 20003,
				"protocol": "vless",
				"settings": {"decryption": "none"},
				"streamSettings": {
					"network": "grpc",
					"security": "reality",
					"realitySettings": {
						"target": "ebay.com:443",
						"privateKey": "X3rxqzJ2ki0i6v22lwyM-NNQeGctkKCdv1C6lm3GjxA",
						"serverNames": ["ebay.com"],
						"shortIds": ["a3ec67"]
					}
				}
			}`,
			DefaultVariables: "{}",
			Enabled:          &enabled,
		},
		Variables: "{}",
		Enabled:   &enabled,
	}, BuildRef(&node.Server{Id: 1, Name: "hk", Address: "127.0.0.1"}, nil))
	if err != nil {
		t.Fatalf("render config: %v", err)
	}
}

func TestRenderBindingConfigNormalizesGrpcRealityFlow(t *testing.T) {
	enabled := true
	server := &node.Server{Id: 1, Name: "hk", Address: "node.example.com"}
	binding := &node.ServerXrayTemplate{
		Alias: "main-grpc",
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Config: `{
				"tag": "main-grpc",
				"port": 443,
				"protocol": "vless",
				"settings": {
					"decryption": "none",
					"clients": [{"email": "template", "flow": "xtls-rprx-vision"}]
				},
				"streamSettings": {
					"network": "grpc",
					"security": "reality",
					"realitySettings": {
						"target": "www.ebay.com:443",
						"privateKey": "X3rxqzJ2ki0i6v22lwyM-NNQeGctkKCdv1C6lm3GjxA",
						"serverNames": ["www.ebay.com"],
						"shortIds": ["0123456789abcdef"]
					},
					"grpcSettings": {"serviceName": "grpc", "authority": "www.ebay.com"}
				}
			}`,
			DefaultVariables: `{"flow":"xtls-rprx-vision"}`,
			Enabled:          &enabled,
		},
		Variables: `{"flow":"xtls-rprx-vision"}`,
		Enabled:   &enabled,
	}
	ref := BuildRef(server, []*node.ServerXrayTemplate{binding})
	rendered, err := RenderBindingConfig(server, binding, ref)
	if err != nil {
		t.Fatalf("render config: %v", err)
	}
	settings := rendered["settings"].(ConfigObject)
	clients := settings["clients"].([]any)
	client := clients[0].(ConfigObject)
	if _, ok := client["flow"]; ok {
		t.Fatalf("grpc reality runtime config must omit flow, got %v", client["flow"])
	}

	proxy, err := RenderSubscriptionProxy(server, binding, ref, UserInfo{
		UUID: "00000000-0000-0000-0000-000000000000",
	})
	if err != nil {
		t.Fatalf("render subscription proxy: %v", err)
	}
	if proxy.Flow != "none" {
		t.Fatalf("grpc reality subscription flow = %q, want none", proxy.Flow)
	}
}

func TestBindingMatchesInboundAcceptsStableAliases(t *testing.T) {
	binding := &node.ServerXrayTemplate{
		Alias: "main-grpc",
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Name: "VLESS Reality gRPC",
			Config: `{
				"tag": "vless-reality-grpc",
				"port": 443,
				"protocol": "vless",
				"streamSettings": {"network": "grpc"}
			}`,
		},
	}
	for _, inbound := range []string{
		"main-grpc",
		"VLESS Reality gRPC",
		"VLESS Reality gRPC Inbound",
		"vless-reality-grpc",
	} {
		if !BindingMatchesInbound(binding, inbound) {
			t.Fatalf("expected inbound %q to match binding aliases %#v", inbound, BindingInboundAliases(binding))
		}
	}
	if BindingMatchesInbound(binding, "vless") {
		t.Fatal("protocol name alone must not match a specific inbound binding")
	}
}

func TestBindingInboundAliasesDeduplicatesCandidates(t *testing.T) {
	binding := &node.ServerXrayTemplate{
		Alias: "main-grpc",
		Template: &node.XrayTemplate{
			Type:   node.XrayTemplateInbound,
			Name:   "main-grpc Inbound",
			Config: `{"tag":"main-grpc"}`,
		},
	}
	got := BindingInboundAliases(binding)
	want := []string{"main-grpc", "main-grpc Inbound"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BindingInboundAliases() = %#v, want %#v", got, want)
	}
}

func TestBindingInboundAliasesFromConfigUsesRenderedTag(t *testing.T) {
	binding := &node.ServerXrayTemplate{
		Alias: "main-grpc",
		Template: &node.XrayTemplate{
			Type:   node.XrayTemplateInbound,
			Name:   "VLESS",
			Config: `{"tag":"template-tag"}`,
		},
	}
	got := BindingInboundAliasesFromConfig(binding, ConfigObject{"tag": "rendered-tag"})
	want := []string{"main-grpc", "main-grpc Inbound", "VLESS", "VLESS Inbound", "rendered-tag"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BindingInboundAliasesFromConfig() = %#v, want %#v", got, want)
	}
}

func TestContainsInboundAlias(t *testing.T) {
	aliases := []string{"main-grpc", "main-grpc Inbound"}
	if !containsInboundAlias(aliases, "main-grpc") {
		t.Fatal("containsInboundAlias should find existing alias")
	}
	if containsInboundAlias(aliases, "missing") {
		t.Fatal("containsInboundAlias should miss unknown alias")
	}
}

func TestInboundBindingIndexMatchesAliasesAndFiltersDisabled(t *testing.T) {
	enabled := true
	disabled := false
	first := &node.ServerXrayTemplate{
		Alias:               "main-grpc",
		SubscriptionEnabled: &enabled,
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Name: "VLESS Reality gRPC",
			Config: `{
				"tag": "vless-reality-grpc",
				"port": 443,
				"protocol": "vless"
			}`,
		},
	}
	second := &node.ServerXrayTemplate{
		Alias:               "main-grpc",
		SubscriptionEnabled: &enabled,
		Template: &node.XrayTemplate{
			Type:   node.XrayTemplateInbound,
			Name:   "Duplicate",
			Config: `{"tag":"duplicate"}`,
		},
	}
	disabledBinding := &node.ServerXrayTemplate{
		Alias:               "disabled",
		SubscriptionEnabled: &disabled,
		Template: &node.XrayTemplate{
			Type:   node.XrayTemplateInbound,
			Name:   "Disabled",
			Config: `{"tag":"disabled"}`,
		},
	}
	outbound := &node.ServerXrayTemplate{
		Alias: "out",
		Template: &node.XrayTemplate{
			Type:   node.XrayTemplateOutbound,
			Name:   "Outbound",
			Config: `{}`,
		},
	}

	index := NewInboundBindingIndex([]*node.ServerXrayTemplate{first, second, disabledBinding, outbound})
	if got := index.Match("vless-reality-grpc"); len(got) != 1 || got[0] != first {
		t.Fatalf("tag match = %#v, want first binding", got)
	}
	if got := index.Match("main-grpc"); len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatalf("duplicate alias match = %#v, want first and second bindings", got)
	}
	if got := index.Match("disabled"); len(got) != 0 {
		t.Fatalf("disabled binding should be filtered: %#v", got)
	}
	if got := index.Match("out"); len(got) != 0 {
		t.Fatalf("non-inbound binding should be filtered: %#v", got)
	}
}

func TestInboundBindingIndexUsesLinearPathForSmallInput(t *testing.T) {
	first := &node.ServerXrayTemplate{Alias: "first"}
	second := &node.ServerXrayTemplate{Alias: "second"}
	index := &InboundBindingIndex{}
	index.add("main", first)
	index.add("other", second)
	index.add("main", second)

	if index.byAlias != nil {
		t.Fatal("small inbound binding index should not allocate map")
	}
	got := index.Match(" main ")
	if len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatalf("Match(main) = %#v, want first and second", got)
	}
	if got := index.Match("missing"); len(got) != 0 {
		t.Fatalf("missing alias match = %#v, want empty", got)
	}
}

func TestInboundBindingIndexUpgradesLargeInputToMap(t *testing.T) {
	index := &InboundBindingIndex{capHint: smallInboundBindingAliasIndexThreshold + 20}
	binding := &node.ServerXrayTemplate{Alias: "binding"}
	for i := 1; i <= smallInboundBindingAliasIndexThreshold+1; i++ {
		index.add(fmt.Sprintf("alias-%d", i), binding)
	}

	if index.byAlias == nil {
		t.Fatal("large inbound binding index should allocate map")
	}
	lastAlias := fmt.Sprintf("alias-%d", smallInboundBindingAliasIndexThreshold+1)
	if got := index.Match(lastAlias); len(got) != 1 || got[0] != binding {
		t.Fatalf("Match(%s) = %#v, want binding", lastAlias, got)
	}
	if maxCap := len(index.rows) * 2; cap(index.rows) > maxCap {
		t.Fatalf("inbound binding index rows cap = %d, want at most %d", cap(index.rows), maxCap)
	}
	for i := 0; i < smallInboundBindingAliasIndexThreshold+20; i++ {
		index.add("alias-1", binding)
	}
	if maxCap := len(index.rows) * 2; cap(index.rows) > maxCap {
		t.Fatalf("inbound binding index rows cap after duplicates = %d, want at most %d", cap(index.rows), maxCap)
	}
}

func TestInboundBindingIndexAvoidsMapForLargeDuplicateInput(t *testing.T) {
	index := &InboundBindingIndex{capHint: smallInboundBindingAliasIndexThreshold + 20}
	binding := &node.ServerXrayTemplate{Alias: "binding"}
	for i := 0; i < smallInboundBindingAliasIndexThreshold+20; i++ {
		index.add("main", binding)
	}

	if index.byAlias != nil {
		t.Fatal("duplicate-heavy inbound binding index should not allocate map")
	}
	if len(index.rows) != 1 {
		t.Fatalf("inbound binding index rows len = %d, want 1", len(index.rows))
	}
	if cap(index.rows) != smallInboundBindingAliasIndexThreshold+1 {
		t.Fatalf("inbound binding index rows cap = %d, want %d", cap(index.rows), smallInboundBindingAliasIndexThreshold+1)
	}
	if got := index.Match("main"); len(got) != smallInboundBindingAliasIndexThreshold+20 {
		t.Fatalf("Match(main) len = %d, want duplicate binding count", len(got))
	}
}

func TestNewInboundBindingIndexDoesNotPreallocateRowsForFilteredBindings(t *testing.T) {
	disabled := false
	bindings := make([]*node.ServerXrayTemplate, 0, smallInboundBindingAliasIndexThreshold+20)
	for i := 0; i < smallInboundBindingAliasIndexThreshold+20; i++ {
		bindings = append(bindings, &node.ServerXrayTemplate{
			SubscriptionEnabled: &disabled,
			Template:            &node.XrayTemplate{Type: node.XrayTemplateInbound},
		})
	}

	index := NewInboundBindingIndex(bindings)
	if len(index.rows) != 0 {
		t.Fatalf("filtered inbound binding index rows len = %d, want 0", len(index.rows))
	}
	if cap(index.rows) != 0 {
		t.Fatalf("filtered inbound binding index rows cap = %d, want 0", cap(index.rows))
	}
	if index.byAlias != nil {
		t.Fatal("filtered inbound binding index should not allocate map")
	}
}

func TestVariablesPreferTemplateConfigOverDefaultVariables(t *testing.T) {
	enabled := true
	vars := Variables(&node.ServerXrayTemplate{
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Config: `{
				"tag": "from-config",
				"listen": "0.0.0.0",
				"port": 443,
				"protocol": "vless",
				"settings": {"decryption": "none"},
				"streamSettings": {"network": "tcp", "security": "none"}
			}`,
			DefaultVariables: `{"tag":"from-default","port":8443,"network":"grpc"}`,
			Enabled:          &enabled,
		},
		Variables: "{}",
		Enabled:   &enabled,
	})
	if vars["tag"] != "from-config" {
		t.Fatalf("tag = %v, want template config value", vars["tag"])
	}
	if vars["port"] != float64(443) {
		t.Fatalf("port = %v, want template config value", vars["port"])
	}
	if vars["network"] != "tcp" {
		t.Fatalf("network = %v, want template config value", vars["network"])
	}
}

func TestBuildRefReturnsVariablesByTypeAndAlias(t *testing.T) {
	ref := BuildRef(&node.Server{Id: 9, Name: "sg", Address: "node.example.com"}, []*node.ServerXrayTemplate{
		{
			Alias: "main",
			Template: &node.XrayTemplate{
				Type:             node.XrayTemplateInbound,
				Name:             "vless",
				Config:           `{"tag":"vless-main"}`,
				DefaultVariables: `{}`,
			},
			Variables: `{"port":443}`,
		},
	})

	vars := ref[node.XrayTemplateInbound]["main"]
	if vars["tag"] != "vless-main" || vars["port"].(float64) != 443 || vars["server_name"] != "sg" {
		t.Fatalf("ref variables = %#v", vars)
	}
}

func TestRenderBindingPlanReusesPlainVariablesAndBuildsRef(t *testing.T) {
	server := &node.Server{Id: 9, Name: "sg", Address: "node.example.com"}
	binding := &node.ServerXrayTemplate{
		Alias: "main",
		Template: &node.XrayTemplate{
			Type:             node.XrayTemplateInbound,
			Name:             "vless",
			Config:           `{"port":443}`,
			DefaultVariables: `{"network":"tcp"}`,
		},
		Variables: `{"port":8443}`,
	}

	plan := newRenderBindingPlan(server, []*node.ServerXrayTemplate{binding})
	if len(plan.entries) != 1 || plan.counts.Inbounds != 1 {
		t.Fatalf("plan shape = %#v, want one inbound entry", plan)
	}
	vars := plan.entries[0].variables
	if vars["port"].(float64) != 8443 || vars["network"] != "tcp" {
		t.Fatalf("render variables = %#v, want merged template and binding variables", vars)
	}
	if _, ok := vars["alias"]; ok {
		t.Fatalf("render variables should not include ref-only alias: %#v", vars)
	}
	refVars := plan.ref[node.XrayTemplateInbound]["main"]
	if refVars["alias"] != "main" || refVars["server_name"] != "sg" || refVars["tag"] != "main" {
		t.Fatalf("ref variables = %#v, want alias/server/tag values", refVars)
	}
}

func TestRenderBindingPlanSkipsNilWithoutInputSizedCapacity(t *testing.T) {
	bindings := make([]*node.ServerXrayTemplate, viewutil.SmallRowsCapacityThreshold+20)
	bindings[0] = renderTestInboundBinding("main", `{}`)

	plan := newRenderBindingPlan(&node.Server{Id: 9, Name: "sg", Address: "node.example.com"}, bindings)
	if len(plan.entries) != 1 {
		t.Fatalf("render plan entries len = %d, want 1", len(plan.entries))
	}
	if cap(plan.entries) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy render plan entries cap = %d, want %d", cap(plan.entries), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestRenderBindingPlanUsesCapacityHintAfterActualThreshold(t *testing.T) {
	bindings := make([]*node.ServerXrayTemplate, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		bindings[index] = renderTestInboundBinding(fmt.Sprintf("in-%d", index), `{}`)
	}

	plan := newRenderBindingPlan(&node.Server{Id: 9, Name: "sg", Address: "node.example.com"}, bindings)
	if len(plan.entries) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("render plan entries len = %d, want %d", len(plan.entries), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(plan.entries) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("render plan entries cap = %d, want %d", cap(plan.entries), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestRenderBindingConfigListFiltersEntriesAndKeepsFullRef(t *testing.T) {
	server := &node.Server{Id: 9, Name: "sg", Address: "node.example.com"}
	enabled := true
	rendered, err := RenderBindingConfigList(server, []*node.ServerXrayTemplate{
		{
			Alias: "main-in",
			Template: &node.XrayTemplate{
				Type: node.XrayTemplateInbound,
				ConfigTemplate: `{
					"tag": "{{ .Ref.outbound.direct.tag }}",
					"port": 443,
					"protocol": "vless",
					"settings": {"clients": [], "decryption": "none"}
				}`,
				DefaultVariables: "{}",
				Enabled:          &enabled,
			},
			Variables: "{}",
			Enabled:   &enabled,
		},
		{
			Alias: "direct",
			Template: &node.XrayTemplate{
				Type:             node.XrayTemplateOutbound,
				Config:           `{"tag":"direct","protocol":"freedom"}`,
				DefaultVariables: "{}",
				Enabled:          &enabled,
			},
			Variables: "{}",
			Enabled:   &enabled,
		},
	}, RenderBindingConfigOptions{TemplateType: node.XrayTemplateInbound})
	if err != nil {
		t.Fatalf("render binding config list: %v", err)
	}
	if len(rendered) != 1 {
		t.Fatalf("rendered bindings = %d, want only inbound", len(rendered))
	}
	if rendered[0].Config["tag"] != "direct" {
		t.Fatalf("rendered inbound tag = %v, want outbound ref value", rendered[0].Config["tag"])
	}
}

func TestRenderSubscriptionProxyListUsesSelectedBindings(t *testing.T) {
	server := &node.Server{Id: 9, Name: "sg", Address: "node.example.com"}
	enabled := true
	disabled := false
	selected := &node.ServerXrayTemplate{
		Alias: "main-in",
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Config: `{
				"tag": "main-in",
				"port": 8443,
				"protocol": "vless",
				"settings": {"clients": [], "decryption": "none"}
			}`,
			DefaultVariables: "{}",
			Enabled:          &enabled,
		},
		Variables:             `{"type":"vless"}`,
		SubscriptionVariables: `{"security":"tls"}`,
		SubscriptionEnabled:   &enabled,
		Enabled:               &enabled,
	}
	disabledBinding := &node.ServerXrayTemplate{
		Alias: "disabled-in",
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Config: `{
				"tag": "disabled-in",
				"port": 9443,
				"protocol": "vless",
				"settings": {"clients": [], "decryption": "none"}
			}`,
			DefaultVariables: "{}",
			Enabled:          &enabled,
		},
		SubscriptionEnabled: &disabled,
		Enabled:             &enabled,
	}

	rendered, err := RenderSubscriptionProxyList(
		server,
		[]*node.ServerXrayTemplate{selected, disabledBinding},
		[]*node.ServerXrayTemplate{selected, disabledBinding},
		UserInfo{Password: "user-pass", UUID: "user-uuid"},
	)
	if err != nil {
		t.Fatalf("render subscription proxy list: %v", err)
	}
	if len(rendered) != 1 {
		t.Fatalf("rendered proxies = %d, want only selected enabled binding", len(rendered))
	}
	if rendered[0].Binding != selected {
		t.Fatalf("rendered binding = %#v, want selected binding", rendered[0].Binding)
	}
	if rendered[0].Proxy.Port != 8443 || rendered[0].Proxy.Security != "tls" {
		t.Fatalf("proxy = %#v, want rendered port and subscription variables", rendered[0].Proxy)
	}
}

func TestRenderSubscriptionProxyListSkipsNilProxyWithoutInputSizedCapacity(t *testing.T) {
	bindings := make([]*node.ServerXrayTemplate, viewutil.SmallRowsCapacityThreshold+20)
	for index := range bindings {
		bindings[index] = renderTestInboundBinding(fmt.Sprintf("disabled-%d", index), `{"enabled":false}`)
	}
	bindings[0] = renderTestInboundBinding("enabled", `{}`)

	rendered, err := RenderSubscriptionProxyList(
		&node.Server{Id: 9, Name: "sg", Address: "node.example.com"},
		bindings,
		bindings,
		UserInfo{Password: "user-pass", UUID: "user-uuid"},
	)
	if err != nil {
		t.Fatalf("render subscription proxy list: %v", err)
	}
	if len(rendered) != 1 {
		t.Fatalf("rendered proxies len = %d, want 1", len(rendered))
	}
	if cap(rendered) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy rendered proxies cap = %d, want %d", cap(rendered), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestSelectedBindingSetSkipsNilWithoutInputSizedCapacity(t *testing.T) {
	bindings := make([]*node.ServerXrayTemplate, smallSelectedBindingSetThreshold+20)
	bindings[0] = renderTestInboundBinding("main", `{}`)

	set := newSelectedBindingSet(bindings)
	if len(set.rows) != 1 {
		t.Fatalf("selected binding rows len = %d, want 1", len(set.rows))
	}
	if cap(set.rows) != smallSelectedBindingSetThreshold+1 {
		t.Fatalf("nil-heavy selected binding rows cap = %d, want %d", cap(set.rows), smallSelectedBindingSetThreshold+1)
	}
	if set.index != nil {
		t.Fatal("nil-heavy selected binding set should stay on linear path")
	}
}

func TestSelectedBindingSetAvoidsMapForLargeDuplicateInput(t *testing.T) {
	binding := renderTestInboundBinding("main", `{}`)
	bindings := make([]*node.ServerXrayTemplate, smallSelectedBindingSetThreshold+20)
	for index := range bindings {
		bindings[index] = binding
	}

	set := newSelectedBindingSet(bindings)
	if len(set.rows) != 1 {
		t.Fatalf("selected binding rows len = %d, want 1", len(set.rows))
	}
	if cap(set.rows) != smallSelectedBindingSetThreshold+1 {
		t.Fatalf("duplicate-heavy selected binding rows cap = %d, want %d", cap(set.rows), smallSelectedBindingSetThreshold+1)
	}
	if set.index != nil {
		t.Fatal("duplicate-heavy selected binding set should stay on linear path")
	}
}

func TestSelectedBindingSetUsesCapacityHintAfterActualThreshold(t *testing.T) {
	bindings := make([]*node.ServerXrayTemplate, smallSelectedBindingSetThreshold+20)
	for index := 0; index <= smallSelectedBindingSetThreshold; index++ {
		bindings[index] = renderTestInboundBinding(fmt.Sprintf("main-%d", index), `{}`)
	}

	set := newSelectedBindingSet(bindings)
	if len(set.rows) != smallSelectedBindingSetThreshold+1 {
		t.Fatalf("selected binding rows len = %d, want %d", len(set.rows), smallSelectedBindingSetThreshold+1)
	}
	if maxCap := len(set.rows) * 2; cap(set.rows) > maxCap {
		t.Fatalf("selected binding rows cap = %d, want at most %d", cap(set.rows), maxCap)
	}
	if set.index == nil {
		t.Fatal("large selected binding set should allocate map")
	}
	for index := 0; index < smallSelectedBindingSetThreshold+20; index++ {
		set.Add(bindings[0])
	}
	if maxCap := len(set.rows) * 2; cap(set.rows) > maxCap {
		t.Fatalf("selected binding rows cap after duplicates = %d, want at most %d", cap(set.rows), maxCap)
	}
}

func TestRenderConfigBucketsAndSubscriptionProxiesReusesRenderedConfigs(t *testing.T) {
	server := &node.Server{Id: 9, Name: "sg", Address: "node.example.com"}
	enabled := true
	disabled := false
	inbound := &node.ServerXrayTemplate{
		Alias: "main-in",
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Config: `{
				"tag": "main-in",
				"port": 8443,
				"protocol": "vless",
				"settings": {"clients": [], "decryption": "none"}
			}`,
			DefaultVariables: "{}",
			Enabled:          &enabled,
		},
		Variables:           "{}",
		SubscriptionEnabled: &enabled,
		Enabled:             &enabled,
	}
	disabledSubscription := &node.ServerXrayTemplate{
		Alias: "disabled-in",
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Config: `{
				"tag": "disabled-in",
				"port": 9443,
				"protocol": "vless",
				"settings": {"clients": [], "decryption": "none"}
			}`,
			DefaultVariables: "{}",
			Enabled:          &enabled,
		},
		Variables:           "{}",
		SubscriptionEnabled: &disabled,
		Enabled:             &enabled,
	}

	buckets, proxies, err := RenderConfigBucketsAndSubscriptionProxies(
		server,
		[]*node.ServerXrayTemplate{inbound, disabledSubscription},
		UserInfo{Password: "user-pass", UUID: "user-uuid"},
	)
	if err != nil {
		t.Fatalf("render config buckets and proxies: %v", err)
	}
	if len(buckets.Inbounds) != 2 {
		t.Fatalf("inbound configs = %d, want both config bindings", len(buckets.Inbounds))
	}
	if len(proxies) != 1 || proxies[0].Binding != inbound {
		t.Fatalf("proxies = %#v, want only subscription-enabled inbound", proxies)
	}
	if proxies[0].Proxy.Port != 8443 {
		t.Fatalf("proxy port = %d, want rendered config port", proxies[0].Proxy.Port)
	}
}

func TestRenderConfigBucketsAndSubscriptionProxiesSkipsDisabledWithoutInputSizedCapacity(t *testing.T) {
	disabled := false
	bindings := make([]*node.ServerXrayTemplate, viewutil.SmallRowsCapacityThreshold+20)
	for index := range bindings {
		binding := renderTestInboundBinding(fmt.Sprintf("disabled-%d", index), `{}`)
		binding.SubscriptionEnabled = &disabled
		bindings[index] = binding
	}
	bindings[0] = renderTestInboundBinding("enabled", `{}`)

	buckets, proxies, err := RenderConfigBucketsAndSubscriptionProxies(
		&node.Server{Id: 9, Name: "sg", Address: "node.example.com"},
		bindings,
		UserInfo{Password: "user-pass", UUID: "user-uuid"},
	)
	if err != nil {
		t.Fatalf("render config buckets and proxies: %v", err)
	}
	if len(buckets.Inbounds) != len(bindings) {
		t.Fatalf("inbound configs = %d, want %d", len(buckets.Inbounds), len(bindings))
	}
	if len(proxies) != 1 {
		t.Fatalf("rendered proxies len = %d, want 1", len(proxies))
	}
	if cap(proxies) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("disabled-heavy rendered proxies cap = %d, want %d", cap(proxies), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestRenderConfigBucketsAndSubscriptionProxiesUseCapacityHintAfterActualThreshold(t *testing.T) {
	disabled := false
	bindings := make([]*node.ServerXrayTemplate, viewutil.SmallRowsCapacityThreshold+20)
	for index := range bindings {
		binding := renderTestInboundBinding(fmt.Sprintf("disabled-%d", index), `{}`)
		binding.SubscriptionEnabled = &disabled
		bindings[index] = binding
	}
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		bindings[index] = renderTestInboundBinding(fmt.Sprintf("enabled-%d", index), `{}`)
	}

	_, proxies, err := RenderConfigBucketsAndSubscriptionProxies(
		&node.Server{Id: 9, Name: "sg", Address: "node.example.com"},
		bindings,
		UserInfo{Password: "user-pass", UUID: "user-uuid"},
	)
	if err != nil {
		t.Fatalf("render config buckets and proxies: %v", err)
	}
	if len(proxies) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("rendered proxies len = %d, want %d", len(proxies), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(proxies) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("rendered proxies cap = %d, want %d", cap(proxies), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestRenderBindingConfigAppliesDNSVariableOverrides(t *testing.T) {
	enabled := true
	rendered, err := RenderBindingConfig(&node.Server{Id: 1, Name: "hk", Address: "127.0.0.1"}, &node.ServerXrayTemplate{
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateDNS,
			Config: `{
				"servers": ["1.1.1.1"],
				"queryStrategy": "UseIP"
			}`,
			DefaultVariables: `{
				"dns_servers": ["8.8.8.8"],
				"query_strategy": "UseSystem"
			}`,
			Enabled: &enabled,
		},
		Variables: `{
			"dns_servers": ["9.9.9.9"],
			"query_strategy": "UseIPv4"
		}`,
		Enabled: &enabled,
	}, BuildRef(&node.Server{Id: 1, Name: "hk", Address: "127.0.0.1"}, nil))
	if err != nil {
		t.Fatalf("render config: %v", err)
	}
	if !reflect.DeepEqual(rendered["servers"], []any{"9.9.9.9"}) {
		t.Fatalf("servers = %#v, want binding override", rendered["servers"])
	}
	if rendered["queryStrategy"] != "UseIPv4" {
		t.Fatalf("queryStrategy = %v, want binding override", rendered["queryStrategy"])
	}
}

func TestRenderBindingConfigAppliesRoutingVariableOverrides(t *testing.T) {
	enabled := true
	rendered, err := RenderBindingConfig(&node.Server{Id: 1, Name: "hk", Address: "127.0.0.1"}, &node.ServerXrayTemplate{
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateRouting,
			Config: `{
				"domainStrategy": "AsIs",
				"rules": [{"type":"field","outboundTag":"direct"}]
			}`,
			DefaultVariables: "{}",
			Enabled:          &enabled,
		},
		Variables: `{
			"routing_domain_strategy": "IPIfNonMatch",
			"routing_rules": [{"type":"field","protocol":["bittorrent"],"outboundTag":"block"}]
		}`,
		Enabled: &enabled,
	}, BuildRef(&node.Server{Id: 1, Name: "hk", Address: "127.0.0.1"}, nil))
	if err != nil {
		t.Fatalf("render config: %v", err)
	}
	if rendered["domainStrategy"] != "IPIfNonMatch" {
		t.Fatalf("domainStrategy = %v, want binding override", rendered["domainStrategy"])
	}
	rules := rendered["rules"].([]any)
	if len(rules) != 1 || rules[0].(ConfigObject)["outboundTag"] != "block" {
		t.Fatalf("rules = %#v, want binding override", rendered["rules"])
	}
}

func TestRenderBindingConfigAppliesOutboundVariableOverrides(t *testing.T) {
	enabled := true
	rendered, err := RenderBindingConfig(&node.Server{Id: 1, Name: "hk", Address: "127.0.0.1"}, &node.ServerXrayTemplate{
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateOutbound,
			Config: `{
				"tag": "trojan-out",
				"protocol": "trojan",
				"settings": {
					"address": "old.example.com",
					"port": 443,
					"password": "old-password"
				}
			}`,
			DefaultVariables: "{}",
			Enabled:          &enabled,
		},
		Variables: `{
			"out_address": "new.example.com",
			"out_port": 8443,
			"out_password": "new-password"
		}`,
		Enabled: &enabled,
	}, BuildRef(&node.Server{Id: 1, Name: "hk", Address: "127.0.0.1"}, nil))
	if err != nil {
		t.Fatalf("render config: %v", err)
	}
	settings := rendered["settings"].(ConfigObject)
	if settings["address"] != "new.example.com" {
		t.Fatalf("address = %v, want binding override", settings["address"])
	}
	if settings["port"] != float64(8443) {
		t.Fatalf("port = %v, want binding override", settings["port"])
	}
	if settings["password"] != "new-password" {
		t.Fatalf("password = %v, want binding override", settings["password"])
	}
}

func TestRenderSubscriptionProxyDerivesRealityPublicKey(t *testing.T) {
	enabled := true
	proxy, err := RenderSubscriptionProxy(&node.Server{Id: 1, Name: "hk", Address: "node.example.com"}, &node.ServerXrayTemplate{
		Alias: "main",
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Config: `{
				"tag": "main",
				"port": 443,
				"protocol": "vless",
				"settings": {
					"decryption": "none",
					"clients": [{"flow": "xtls-rprx-vision"}]
				},
				"streamSettings": {
					"network": "tcp",
					"security": "reality",
					"realitySettings": {
						"target": "www.ebay.com:443",
						"privateKey": "X3rxqzJ2ki0i6v22lwyM-NNQeGctkKCdv1C6lm3GjxA",
						"serverNames": ["www.ebay.com"],
						"shortIds": ["0123456789abcdef"]
					}
				}
			}`,
			DefaultVariables: "{}",
			Enabled:          &enabled,
		},
		Variables: "{}",
		Enabled:   &enabled,
	}, BuildRef(&node.Server{Id: 1, Name: "hk", Address: "node.example.com"}, nil), UserInfo{
		UUID: "00000000-0000-0000-0000-000000000000",
	})
	if err != nil {
		t.Fatalf("render subscription proxy: %v", err)
	}
	if proxy == nil {
		t.Fatal("expected proxy")
	}
	if proxy.RealityPublicKey == "" {
		t.Fatal("expected derived reality public key")
	}
	if proxy.RealityPublicKey == proxy.RealityPrivateKey {
		t.Fatal("reality public key must not equal private key")
	}
}

func TestRenderSubscriptionProxyUsesVlessUserFlow(t *testing.T) {
	enabled := true
	server := &node.Server{Id: 1, Name: "hk", Address: "node.example.com"}
	binding := &node.ServerXrayTemplate{
		Alias: "main",
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Config: `{
				"tag": "main",
				"port": 443,
				"protocol": "vless",
				"settings": {
					"decryption": "none",
					"clients": [{"email": "template", "flow": "xtls-rprx-vision"}]
				},
				"streamSettings": {
					"network": "tcp",
					"security": "reality",
					"realitySettings": {
						"target": "www.ebay.com:443",
						"privateKey": "X3rxqzJ2ki0i6v22lwyM-NNQeGctkKCdv1C6lm3GjxA",
						"serverNames": ["www.ebay.com"],
						"shortIds": ["0123456789abcdef"]
					}
				}
			}`,
			DefaultVariables: "{}",
			Enabled:          &enabled,
		},
		Variables: `{"flow":"xtls-rprx-vision-udp443"}`,
		Enabled:   &enabled,
	}
	ref := BuildRef(server, []*node.ServerXrayTemplate{binding})
	rendered, err := RenderBindingConfig(server, binding, ref)
	if err != nil {
		t.Fatalf("render binding config: %v", err)
	}
	settings := rendered["settings"].(ConfigObject)
	clients := settings["clients"].([]any)
	client := clients[0].(ConfigObject)
	if client["flow"] != "xtls-rprx-vision-udp443" {
		t.Fatalf("user flow = %v, want binding flow override", client["flow"])
	}
	if _, ok := settings["flow"]; ok {
		t.Fatalf("vless inbound flow should stay on clients[0], got settings.flow=%v", settings["flow"])
	}

	proxy, err := RenderSubscriptionProxy(server, binding, ref, UserInfo{
		UUID: "00000000-0000-0000-0000-000000000000",
	})
	if err != nil {
		t.Fatalf("render subscription proxy: %v", err)
	}
	if proxy.Flow != "xtls-rprx-vision-udp443" {
		t.Fatalf("proxy flow = %q, want binding flow override", proxy.Flow)
	}
}

func TestRenderSubscriptionProxyKeepsXHTTPExtra(t *testing.T) {
	enabled := true
	proxy, err := RenderSubscriptionProxy(&node.Server{Id: 1, Name: "hk", Address: "node.example.com"}, &node.ServerXrayTemplate{
		Alias: "xhttp",
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Config: `{
				"tag": "xhttp",
				"port": 443,
				"protocol": "vless",
				"settings": {
					"decryption": "none",
					"clients": [{"email": "template"}]
				},
				"streamSettings": {
					"network": "xhttp",
					"security": "reality",
					"realitySettings": {
						"target": "www.ebay.com:443",
						"privateKey": "X3rxqzJ2ki0i6v22lwyM-NNQeGctkKCdv1C6lm3GjxA",
						"serverNames": ["www.ebay.com"],
						"shortIds": ["0123456789abcdef"]
					},
					"xhttpSettings": {
						"path": "/xhttp",
						"mode": "stream-one",
						"extra": {
							"xmux": {
								"maxConnections": 1,
								"maxConcurrency": 0,
								"cMaxReuseTimes": 0,
								"hMaxRequestTimes": 0,
								"hMaxReusableSecs": 0,
								"hKeepAlivePeriod": 0
							}
						}
					}
				}
			}`,
			DefaultVariables: "{}",
			Enabled:          &enabled,
		},
		Variables: "{}",
		Enabled:   &enabled,
	}, BuildRef(&node.Server{Id: 1, Name: "hk", Address: "node.example.com"}, nil), UserInfo{
		UUID: "00000000-0000-0000-0000-000000000000",
	})
	if err != nil {
		t.Fatalf("render subscription proxy: %v", err)
	}
	if proxy == nil {
		t.Fatal("expected proxy")
	}
	if proxy.Transport != "xhttp" {
		t.Fatalf("transport = %q, want xhttp", proxy.Transport)
	}
	if proxy.XhttpMode != "stream-one" {
		t.Fatalf("xhttp mode = %q, want stream-one", proxy.XhttpMode)
	}
	if !strings.Contains(proxy.XhttpExtra, `"maxConnections":1`) {
		t.Fatalf("xhttp extra missing maxConnections=1: %s", proxy.XhttpExtra)
	}
}

func TestRenderSubscriptionProxyUsesXHTTPBindingMode(t *testing.T) {
	enabled := true
	server := &node.Server{Id: 1, Name: "hk", Address: "node.example.com"}
	binding := &node.ServerXrayTemplate{
		Alias: "xhttp",
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Config: `{
				"tag": "xhttp",
				"port": 443,
				"protocol": "vless",
				"settings": {
					"decryption": "none",
					"clients": [{"email": "template"}]
				},
				"streamSettings": {
					"network": "xhttp",
					"security": "reality",
					"realitySettings": {
						"target": "www.ebay.com:443",
						"privateKey": "X3rxqzJ2ki0i6v22lwyM-NNQeGctkKCdv1C6lm3GjxA",
						"serverNames": ["www.ebay.com"],
						"shortIds": ["0123456789abcdef"]
					},
					"xhttpSettings": {
						"path": "/xhttp",
						"mode": "packet-up"
					}
				}
			}`,
			SubscriptionMeta: `{
				"transport": "xhttp",
				"xhttp_mode": "stream-one",
				"xhttp_extra": {
					"xmux": {
						"maxConnections": 1
					}
				}
			}`,
			DefaultVariables: "{}",
			Enabled:          &enabled,
		},
		Variables: `{"xhttp_mode":"stream-up"}`,
		Enabled:   &enabled,
	}
	ref := BuildRef(server, []*node.ServerXrayTemplate{binding})
	proxy, err := RenderSubscriptionProxy(server, binding, ref, UserInfo{
		UUID: "00000000-0000-0000-0000-000000000000",
	})
	if err != nil {
		t.Fatalf("render subscription proxy: %v", err)
	}
	if proxy.XhttpMode != "stream-up" {
		t.Fatalf("xhttp mode = %q, want binding override", proxy.XhttpMode)
	}
	if !strings.Contains(proxy.XhttpExtra, `"maxConnections":1`) {
		t.Fatalf("xhttp extra from subscription meta lost: %s", proxy.XhttpExtra)
	}
}

func TestRenderSubscriptionProxyKeepsGrpcSettings(t *testing.T) {
	enabled := true
	proxy, err := RenderSubscriptionProxy(&node.Server{Id: 1, Name: "hk", Address: "node.example.com"}, &node.ServerXrayTemplate{
		Alias: "grpc",
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Config: `{
				"tag": "grpc",
				"port": 443,
				"protocol": "vless",
				"settings": {
					"decryption": "none",
					"clients": [{"email": "template"}]
				},
				"streamSettings": {
					"network": "grpc",
					"security": "reality",
					"realitySettings": {
						"target": "www.ebay.com:443",
						"privateKey": "X3rxqzJ2ki0i6v22lwyM-NNQeGctkKCdv1C6lm3GjxA",
						"serverNames": ["www.ebay.com"],
						"shortIds": ["0123456789abcdef"]
					},
					"grpcSettings": {
						"serviceName": "shop",
						"authority": "www.ebay.com",
						"multiMode": true,
						"user_agent": "alpha-test",
						"idle_timeout": 60,
						"health_check_timeout": 20,
						"permit_without_stream": true,
						"initial_windows_size": 65536
					}
				}
			}`,
			DefaultVariables: "{}",
			Enabled:          &enabled,
		},
		Variables: "{}",
		Enabled:   &enabled,
	}, BuildRef(&node.Server{Id: 1, Name: "hk", Address: "node.example.com"}, nil), UserInfo{
		UUID: "00000000-0000-0000-0000-000000000000",
	})
	if err != nil {
		t.Fatalf("render subscription proxy: %v", err)
	}
	if proxy == nil {
		t.Fatal("expected proxy")
	}
	if proxy.Transport != "grpc" {
		t.Fatalf("transport = %q, want grpc", proxy.Transport)
	}
	if proxy.ServiceName != "shop" {
		t.Fatalf("service name = %q, want shop", proxy.ServiceName)
	}
	if proxy.GrpcAuthority != "www.ebay.com" {
		t.Fatalf("grpc authority = %q, want www.ebay.com", proxy.GrpcAuthority)
	}
	if !proxy.GrpcMultiMode {
		t.Fatal("expected grpc multi mode")
	}
	if proxy.GrpcUserAgent != "alpha-test" {
		t.Fatalf("grpc user agent = %q, want alpha-test", proxy.GrpcUserAgent)
	}
	if proxy.GrpcIdleTimeout != 60 {
		t.Fatalf("grpc idle timeout = %d, want 60", proxy.GrpcIdleTimeout)
	}
	if proxy.GrpcHealthCheckTimeout != 20 {
		t.Fatalf("grpc health check timeout = %d, want 20", proxy.GrpcHealthCheckTimeout)
	}
	if !proxy.GrpcPermitWithoutStream {
		t.Fatal("expected grpc permit without stream")
	}
	if proxy.GrpcInitialWindowsSize != 65536 {
		t.Fatalf("grpc initial windows size = %d, want 65536", proxy.GrpcInitialWindowsSize)
	}
}

func TestRenderSubscriptionProxySupportsHysteria2(t *testing.T) {
	enabled := true
	proxy, err := RenderSubscriptionProxy(&node.Server{Id: 1, Name: "sg", Address: "node.example.com"}, &node.ServerXrayTemplate{
		Alias: "hy2",
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Config: `{
				"tag": "hy2",
				"port": 443,
				"protocol": "hysteria",
				"settings": {
					"version": 2,
					"users": [{"auth": "template", "level": 0, "email": "template"}]
				},
				"streamSettings": {
					"network": "hysteria",
					"security": "tls",
					"hysteriaSettings": {
						"version": 2,
						"udpIdleTimeout": 60
					}
				}
			}`,
			DefaultVariables: "{}",
			Enabled:          &enabled,
		},
		Variables: "{}",
		Enabled:   &enabled,
	}, BuildRef(&node.Server{Id: 1, Name: "sg", Address: "node.example.com"}, nil), UserInfo{
		UUID:     "00000000-0000-0000-0000-000000000000",
		Password: "hy2-password",
	})
	if err != nil {
		t.Fatalf("render subscription proxy: %v", err)
	}
	if proxy == nil {
		t.Fatal("expected proxy")
	}
	if proxy.Type != "hysteria2" {
		t.Fatalf("type = %q, want hysteria2", proxy.Type)
	}
	if proxy.Transport != "hysteria" {
		t.Fatalf("transport = %q, want hysteria", proxy.Transport)
	}
	if proxy.ServerKey != "hy2-password" {
		t.Fatalf("server key = %q, want hy2-password", proxy.ServerKey)
	}
}

func TestValidateTemplateConfigRejectsInvalidConfigTemplate(t *testing.T) {
	err := ValidateTemplateConfig(&node.XrayTemplate{
		Name:             "bad-template",
		Type:             node.XrayTemplateOutbound,
		Config:           "{}",
		ConfigTemplate:   `{"tag": {{ .Vars.missing }}}`,
		DefaultVariables: "{}",
	})
	if err == nil {
		t.Fatal("expected config template validation error")
	}
}

func TestValidateTemplateConfigRejectsInvalidRoutingShape(t *testing.T) {
	err := ValidateTemplateConfig(&node.XrayTemplate{
		Name: "bad-routing",
		Type: node.XrayTemplateRouting,
		Config: `{
			"domainStrategy": "AsIs",
			"rules": {"type": "field", "outboundTag": "block"}
		}`,
		DefaultVariables: "{}",
	})
	if err == nil || !strings.Contains(err.Error(), "routing.rules must be an array") {
		t.Fatalf("expected routing rules validation error, got %v", err)
	}
}

func TestValidateTemplateConfigRejectsInvalidHysteriaTransportVersion(t *testing.T) {
	err := normalizeAndValidateXrayConfig(ConfigObject{
		"protocol": "hysteria",
		"settings": ConfigObject{
			"version": 2,
			"clients": []any{
				ConfigObject{"auth": "template", "level": 0, "email": "template"},
			},
		},
		"streamSettings": ConfigObject{
			"network":  "hysteria",
			"security": "tls",
			"hysteriaSettings": ConfigObject{
				"version": 1,
			},
		},
	}, node.XrayTemplateInbound)
	if err == nil || !strings.Contains(err.Error(), "hysteriaSettings.version must be 2") {
		t.Fatalf("expected hysteria transport version validation error, got %v", err)
	}
}

func TestValidateTemplateConfigRejectsHysteriaTransportWithoutTLS(t *testing.T) {
	err := normalizeAndValidateXrayConfig(ConfigObject{
		"protocol": "vless",
		"settings": ConfigObject{
			"clients":    []any{},
			"decryption": "none",
		},
		"streamSettings": ConfigObject{
			"network":  "hysteria",
			"security": "none",
			"hysteriaSettings": ConfigObject{
				"version": 2,
			},
		},
	}, node.XrayTemplateInbound)
	if err == nil || !strings.Contains(err.Error(), "security to be tls") {
		t.Fatalf("expected hysteria TLS validation error, got %v", err)
	}
}

func TestValidateRoutingReferencesAcceptsKnownOutboundTag(t *testing.T) {
	err := ValidateRoutingReferences([]ConfigObject{
		ConfigObject{"tag": "direct", "protocol": "freedom"},
		ConfigObject{"tag": "block", "protocol": "blackhole"},
	}, ConfigObject{
		"rules": []any{
			ConfigObject{
				"type":        "field",
				"protocol":    []any{"bittorrent"},
				"outboundTag": "block",
			},
		},
	})
	if err != nil {
		t.Fatalf("validate routing references: %v", err)
	}
}

func TestValidateRoutingReferencesRejectsMissingOutboundTag(t *testing.T) {
	err := ValidateRoutingReferences([]ConfigObject{
		ConfigObject{"tag": "direct", "protocol": "freedom"},
	}, ConfigObject{
		"rules": []any{
			ConfigObject{
				"type":        "field",
				"outboundTag": "block",
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), `missing outboundTag "block"`) {
		t.Fatalf("expected missing outbound tag error, got %v", err)
	}
}

func TestValidateRoutingReferencesRejectsMissingNestedBalancerTag(t *testing.T) {
	err := ValidateRoutingReferences([]ConfigObject{
		ConfigObject{"tag": "direct", "protocol": "freedom"},
	}, ConfigObject{
		"rules": []any{
			ConfigObject{
				"type": "logical",
				"mode": "and",
				"rules": []any{
					ConfigObject{
						"type":        "field",
						"balancerTag": "auto",
					},
				},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), `missing balancerTag "auto"`) {
		t.Fatalf("expected missing balancer tag error, got %v", err)
	}
}

func TestCollectRoutingBalancerTagsAvoidsAllocationForMissingList(t *testing.T) {
	if got := collectRoutingBalancerTags(nil); len(got.values) != 0 || got.index != nil {
		t.Fatalf("collectRoutingBalancerTags(nil) = %#v, want empty linear set", got)
	}
	if got := collectRoutingBalancerTags(ConfigObject{"tag": "auto"}); len(got.values) != 0 || got.index != nil {
		t.Fatalf("collectRoutingBalancerTags(non-list) = %#v, want empty linear set", got)
	}
}

func TestRoutingStringSetUsesLinearPathForSmallInput(t *testing.T) {
	var set routingStringSet
	set.Add("direct")
	set.Add("direct")
	set.Add("block")

	if !set.Contains("direct") || !set.Contains("block") {
		t.Fatalf("set missing expected values: %#v", set)
	}
	if set.Contains("missing") {
		t.Fatalf("set unexpectedly contains missing value: %#v", set)
	}
	if len(set.values) != 2 {
		t.Fatalf("set values = %#v, want 2 unique values", set.values)
	}
	if set.index != nil {
		t.Fatal("small routing string set should not allocate map")
	}
}

func TestRoutingStringSetUpgradesLargeInputToMap(t *testing.T) {
	var set routingStringSet
	for i := 0; i <= smallRoutingStringSetThreshold; i++ {
		set.Add(fmt.Sprintf("tag-%d", i))
	}

	if set.index == nil {
		t.Fatal("large routing string set should allocate map")
	}
	if !set.Contains("tag-0") || !set.Contains(fmt.Sprintf("tag-%d", smallRoutingStringSetThreshold)) {
		t.Fatalf("set missing upgraded values: %#v", set)
	}
}

func TestCollectOutboundTagsUsesLinearPathForSmallInput(t *testing.T) {
	got := collectOutboundTags([]ConfigObject{
		ConfigObject{"tag": "direct"},
		ConfigObject{"tag": "block"},
		ConfigObject{"tag": "direct"},
		ConfigObject{"tag": ""},
		nil,
	})

	if !got.Contains("direct") || !got.Contains("block") {
		t.Fatalf("outbound tags missing expected values: %#v", got)
	}
	if len(got.values) != 2 {
		t.Fatalf("outbound tag values = %#v, want 2 unique values", got.values)
	}
	if got.index != nil {
		t.Fatal("small outbound tag set should not allocate map")
	}
}

func TestAppendUniqueConfigList(t *testing.T) {
	result := AppendUniqueConfigList([]any{
		ConfigObject{"type": "field", "outboundTag": "block"},
	}, []ConfigObject{
		ConfigObject{"outboundTag": "block", "type": "field"},
		ConfigObject{"outboundTag": "direct", "type": "field"},
	})
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	if cap(result) != 3 {
		t.Fatalf("cap(result) = %d, want 3", cap(result))
	}
}

func TestUniqueConfigListSkipsDuplicatesWithoutInputSizedCapacity(t *testing.T) {
	items := make([]any, viewutil.SmallRowsCapacityThreshold+20)
	for index := range items {
		items[index] = ConfigObject{"tag": "direct", "index": 1}
	}

	got := uniqueConfigList(items)
	if len(got) != 1 {
		t.Fatalf("unique config list len = %d, want 1", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("duplicate-heavy unique config list cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestStringListValueSkipsEmptyWithoutInputSizedCapacity(t *testing.T) {
	items := make([]string, viewutil.SmallRowsCapacityThreshold+20)
	items[0] = "one"

	got := stringListValue(items)
	if len(got) != 1 || got[0] != "one" {
		t.Fatalf("string list value = %#v, want [one]", got)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("empty-heavy string list cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestSetStringListIfVarSkipsEmptyWithoutInputSizedCapacity(t *testing.T) {
	items := make([]string, viewutil.SmallRowsCapacityThreshold+20)
	items[0] = "one"
	data := ConfigObject{}

	setStringListIfVar(data, ConfigObject{"tags": items}, "tags", "tags")

	got, ok := data["tags"].([]any)
	if !ok {
		t.Fatalf("tags = %#v, want []any", data["tags"])
	}
	if len(got) != 1 || got[0] != "one" {
		t.Fatalf("tags = %#v, want [one]", got)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("empty-heavy tags cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func renderTestInboundBinding(alias string, subscriptionVariables string) *node.ServerXrayTemplate {
	enabled := true
	if subscriptionVariables == "" {
		subscriptionVariables = "{}"
	}
	return &node.ServerXrayTemplate{
		Alias: alias,
		Template: &node.XrayTemplate{
			Type: node.XrayTemplateInbound,
			Config: `{
				"tag": "` + alias + `",
				"port": 8443,
				"protocol": "vless",
				"settings": {"clients": [], "decryption": "none"}
			}`,
			DefaultVariables: "{}",
			Enabled:          &enabled,
		},
		Variables:             `{"type":"vless"}`,
		SubscriptionVariables: subscriptionVariables,
		SubscriptionEnabled:   &enabled,
		Enabled:               &enabled,
	}
}
