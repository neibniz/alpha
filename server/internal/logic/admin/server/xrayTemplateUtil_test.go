package server

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestNormalizeXrayTemplateRawCompactsAndDefaults(t *testing.T) {
	got, err := normalizeXrayTemplateRaw(json.RawMessage(`{
		"port": 443,
		"settings": {
			"clients": []
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"port":443,"settings":{"clients":[]}}` {
		t.Fatalf("normalizeXrayTemplateRaw() = %q", got)
	}

	got, err = normalizeXrayTemplateRaw(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != `{}` {
		t.Fatalf("normalizeXrayTemplateRaw(nil) = %q, want {}", got)
	}
}

func TestNormalizeXrayTemplateRawRejectsInvalidJSON(t *testing.T) {
	if _, err := normalizeXrayTemplateRaw(json.RawMessage(`{"port":`)); err == nil {
		t.Fatal("normalizeXrayTemplateRaw() expected error for invalid JSON")
	}
}

func TestNormalizeXrayTemplateConfigRequiresObject(t *testing.T) {
	if got, err := normalizeXrayTemplateConfig(json.RawMessage(`{"port":443}`)); err != nil || got != `{"port":443}` {
		t.Fatalf("normalizeXrayTemplateConfig() = %q, %v", got, err)
	}
	for _, raw := range []string{`[]`, `"value"`, `42`} {
		if _, err := normalizeXrayTemplateConfig(json.RawMessage(raw)); err == nil {
			t.Fatalf("normalizeXrayTemplateConfig(%s) expected object error", raw)
		}
	}
}

func TestNormalizeXrayTemplateIdentityTrimsAndCountsRunes(t *testing.T) {
	name, description, err := normalizeXrayTemplateIdentity("  VLESS  ", "  production  ")
	if err != nil || name != "VLESS" || description != "production" {
		t.Fatalf("normalizeXrayTemplateIdentity() = %q, %q, %v", name, description, err)
	}
	if _, _, err := normalizeXrayTemplateIdentity("   ", ""); err == nil {
		t.Fatal("blank template name should be rejected")
	}
	if _, _, err := normalizeXrayTemplateIdentity(strings.Repeat("模", 101), ""); err == nil {
		t.Fatal("template name longer than 100 runes should be rejected")
	}
}

func TestXrayTemplateToTypesKeepsRawJSON(t *testing.T) {
	enabled := true
	got, err := xrayTemplateToTypes(&node.XrayTemplate{
		Id:               7,
		Name:             "vless",
		Type:             node.XrayTemplateInbound,
		Config:           `{"protocol":"vless"}`,
		VariablesSchema:  `{"port":{"type":"number"}}`,
		DefaultVariables: `{"port":443}`,
		SubscriptionMeta: `{"type":"vless"}`,
		Description:      "template",
		Enabled:          &enabled,
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Config) != `{"protocol":"vless"}` {
		t.Fatalf("config = %s", got.Config)
	}
	if string(got.VariablesSchema) != `{"port":{"type":"number"}}` {
		t.Fatalf("variables schema = %s", got.VariablesSchema)
	}
	if string(got.DefaultVariables) != `{"port":443}` {
		t.Fatalf("default variables = %s", got.DefaultVariables)
	}
	if string(got.SubscriptionMeta) != `{"type":"vless"}` {
		t.Fatalf("subscription meta = %s", got.SubscriptionMeta)
	}
	if got.CreatedAt != 0 || got.UpdatedAt != 0 {
		t.Fatalf("zero timestamps should be encoded as 0: %#v", got)
	}
}

func TestXrayTemplateRawFromModelTrimsStoredJSON(t *testing.T) {
	got, err := xrayTemplateRawFromModel(" \n {\"protocol\":\"vless\"} \t ")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"protocol":"vless"}` {
		t.Fatalf("trimmed stored json = %q", got)
	}

	empty, err := xrayTemplateRawFromModel(" \n ")
	if err != nil {
		t.Fatal(err)
	}
	if string(empty) != `{}` {
		t.Fatalf("empty stored json = %q, want {}", empty)
	}
}

func TestXrayTemplateRawFromModelUsesSingleStringCopy(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("allocation counts include race detector instrumentation")
	}
	const value = `{"protocol":"vless","settings":{"clients":[]}}`
	allocs := testing.AllocsPerRun(1000, func() {
		got, err := xrayTemplateRawFromModel(value)
		if err != nil {
			t.Fatalf("xrayTemplateRawFromModel returned error: %v", err)
		}
		if string(got) != value {
			t.Fatalf("xrayTemplateRawFromModel = %s, want %s", got, value)
		}
	})
	if allocs > 1 {
		t.Fatalf("xrayTemplateRawFromModel allocations = %v, want <= 1", allocs)
	}
}

func TestXrayTemplateRawCacheReusesTrimmedStoredJSON(t *testing.T) {
	cache := newXrayTemplateRawCache(2)

	first, err := cache.Raw(` {"protocol":"vless"} `)
	if err != nil {
		t.Fatal(err)
	}
	second, err := cache.Raw(`{"protocol":"vless"}`)
	if err != nil {
		t.Fatal(err)
	}

	if string(first) != `{"protocol":"vless"}` || string(second) != string(first) {
		t.Fatalf("cached raw json mismatch: first=%s second=%s", first, second)
	}
	if len(cache.rows) != 1 {
		t.Fatalf("duplicate trimmed raw json should use one cache row, got %d", len(cache.rows))
	}
	if cache.index != nil {
		t.Fatal("small xray template raw cache should not allocate map")
	}
}

func TestXrayTemplateRawCacheCachesInvalidJSON(t *testing.T) {
	cache := newXrayTemplateRawCache(2)

	if raw, err := cache.Raw(`{"protocol":`); err == nil || raw != nil {
		t.Fatalf("invalid json should return nil/error, got raw=%s err=%v", raw, err)
	}
	if raw, err := cache.Raw(` {"protocol": `); err == nil || raw != nil {
		t.Fatalf("trim-equivalent invalid json should return cached nil/error, got raw=%s err=%v", raw, err)
	}
	if len(cache.rows) != 1 {
		t.Fatalf("duplicate invalid raw json should use one cache row, got %d", len(cache.rows))
	}
}

func TestXrayTemplateRawCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := newXrayTemplateRawCache(smallXrayTemplateRawCacheThreshold + 1)
	if cache == nil || cache.rows != nil || cache.index != nil {
		t.Fatalf("large xray template raw cache should not preallocate storage from hint alone: %#v", cache)
	}
	first, err := cache.Raw(` {"tag":"a"} `)
	if err != nil {
		t.Fatal(err)
	}
	second, err := cache.Raw(`{"tag":"a"}`)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) || len(cache.rows) != 1 || cache.index != nil {
		t.Fatalf("duplicate raw json should stay on linear path: rows=%#v index=%#v", cache.rows, cache.index)
	}

	cache = newXrayTemplateRawCache(smallXrayTemplateRawCacheThreshold + 1)
	for i := 0; i <= smallXrayTemplateRawCacheThreshold; i++ {
		_, err := cache.Raw(`{"tag":"` + string(rune('a'+i)) + `"}`)
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = cache.Raw(`{"tag":"a"}`)
	if err != nil {
		t.Fatal(err)
	}

	if len(cache.rows) != smallXrayTemplateRawCacheThreshold+1 || len(cache.index) != smallXrayTemplateRawCacheThreshold+1 {
		t.Fatalf("xray template raw cache rows/index = %d/%d, want %d", len(cache.rows), len(cache.index), smallXrayTemplateRawCacheThreshold+1)
	}
	if string(cache.rows[0].value) != `{"tag":"a"}` {
		t.Fatalf("xray template raw cache did not store first row: %s", cache.rows[0].value)
	}
}

func TestXrayTemplateRawCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := newXrayTemplateRawCache(smallXrayTemplateRawCacheThreshold + 20)
	for i := 0; i <= smallXrayTemplateRawCacheThreshold; i++ {
		if _, err := cache.Raw(`{"tag":"` + string(rune('a'+i)) + `"}`); err != nil {
			t.Fatal(err)
		}
	}
	if cache.index == nil {
		t.Fatal("xray template raw cache should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("xray template raw cache row cap = %d, want at most %d", cap(cache.rows), maxCap)
	}
	for i := 0; i < smallXrayTemplateRawCacheThreshold+20; i++ {
		if _, err := cache.Raw(`{"tag":"a"}`); err != nil {
			t.Fatal(err)
		}
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("xray template raw cache row cap after duplicates = %d, want at most %d", cap(cache.rows), maxCap)
	}
}

func TestXrayTemplateRowsToTypesSkipsNilAndKeepsEmptySlice(t *testing.T) {
	got, err := xrayTemplateRowsToTypes([]*node.XrayTemplate{
		nil,
		{Id: 7, Name: "inbound", Type: node.XrayTemplateInbound, Config: `{"protocol":"vless"}`},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("xray template rows len = %d, want 1", len(got))
	}
	if got[0].Id != 7 || got[0].Name != "inbound" || string(got[0].Config) != `{"protocol":"vless"}` {
		t.Fatalf("xray template row mismatch: %#v", got[0])
	}

	empty, err := xrayTemplateRowsToTypes(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if empty == nil {
		t.Fatal("empty xray template rows must return an empty slice, not nil")
	}
	payload, err := json.Marshal(struct {
		List []types.XrayTemplate `json:"list"`
	}{List: empty})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"list":[]}` {
		t.Fatalf("empty xray template rows marshal = %s, want list []", payload)
	}
}

func TestXrayTemplateListRowsToTypesKeepsPayloadLean(t *testing.T) {
	enabled := true
	now := time.UnixMilli(1700000000000)
	got := xrayTemplateListRowsToTypes([]*node.XrayTemplate{
		nil,
		{
			Id:                       7,
			Name:                     "inbound",
			Type:                     node.XrayTemplateInbound,
			Config:                   `{"protocol":"vless"}`,
			ConfigTemplate:           `{}`,
			VariablesSchema:          `{}`,
			DefaultVariables:         `{}`,
			SubscriptionMeta:         `{}`,
			Description:              "template",
			Enabled:                  &enabled,
			BindingCount:             3,
			SubscriptionBindingCount: 2,
			CreatedAt:                now.Add(-time.Hour),
			UpdatedAt:                now,
		},
	})
	if len(got) != 1 {
		t.Fatalf("xray template list rows len = %d, want 1", len(got))
	}
	if got[0].Id != 7 || got[0].Name != "inbound" || got[0].Type != node.XrayTemplateInbound || got[0].BindingCount != 3 || got[0].SubscriptionBindingCount != 2 || got[0].UpdatedAt != now.UnixMilli() {
		t.Fatalf("xray template list row mismatch: %#v", got[0])
	}
	payload, err := json.Marshal(got[0])
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"config", "config_template", "variables_schema", "default_variables", "subscription_meta", "created_at"} {
		if _, ok := decoded[key]; ok {
			t.Fatalf("xray template list item must not expose %q: %s", key, payload)
		}
	}
}

func TestXrayTemplateCatalogRowsToTypesMapsConfigPayload(t *testing.T) {
	got, err := xrayTemplateCatalogRowsToTypes([]*node.XrayTemplate{
		nil,
		{
			Id:               8,
			Name:             "routing",
			Type:             node.XrayTemplateRouting,
			Config:           `{"rules":[]}`,
			ConfigTemplate:   `{"rules":{{ .vars.rules }}}`,
			VariablesSchema:  `{"properties":{"rules":{"type":"array"}}}`,
			DefaultVariables: `{"rules":[]}`,
			SubscriptionMeta: `{"enabled":true}`,
			Description:      "catalog",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("xray template catalog rows len = %d, want 1", len(got))
	}
	if got[0].Id != 8 || string(got[0].Config) != `{"rules":[]}` || string(got[0].VariablesSchema) == "" || string(got[0].SubscriptionMeta) == "" {
		t.Fatalf("xray template catalog row mismatch: %#v", got[0])
	}
	payload, err := json.Marshal(got[0])
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"created_at", "updated_at"} {
		if _, ok := decoded[key]; ok {
			t.Fatalf("xray template catalog item must not expose %q: %s", key, payload)
		}
	}
}

func TestXrayTemplateRowsToTypesSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*node.XrayTemplate, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &node.XrayTemplate{Id: 1, Name: "first", Type: node.XrayTemplateInbound}
	rows[len(rows)-1] = &node.XrayTemplate{Id: 2, Name: "second", Type: node.XrayTemplateOutbound}

	got, err := xrayTemplateRowsToTypes(rows, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("xray template rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy xray template rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("xray template row order changed: %#v", got)
	}
}

func TestXrayTemplateRowsToTypesUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*node.XrayTemplate, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &node.XrayTemplate{Id: int64(index + 1), Name: "template", Type: node.XrayTemplateInbound}
	}

	got, err := xrayTemplateRowsToTypes(rows, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("xray template rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("xray template rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("xray template row order changed: %#v", got)
	}
}

func TestXrayTemplateToTypesDefaultsNilEnabled(t *testing.T) {
	got, err := xrayTemplateToTypes(&node.XrayTemplate{
		Id:      8,
		Name:    "dns",
		Type:    node.XrayTemplateDNS,
		Config:  `{}`,
		Enabled: nil,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled {
		t.Fatalf("nil template enabled should default true in API response: %#v", got)
	}

	summary, err := xrayTemplateBindingTemplateToTypes(&node.XrayTemplate{
		Id:               9,
		Name:             "outbound",
		Type:             node.XrayTemplateOutbound,
		Config:           `{}`,
		DefaultVariables: `{}`,
		Enabled:          nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Enabled {
		t.Fatalf("nil binding template enabled should default true in API response: %#v", summary)
	}
}

func TestServerXrayTemplateRowsToTypesSkipsNilAndIncludesTemplate(t *testing.T) {
	got, err := serverXrayTemplateRowsToTypes([]*node.ServerXrayTemplate{
		nil,
		{
			Id:                    12,
			ServerId:              7,
			TemplateId:            3,
			Variables:             `{"port":443}`,
			SubscriptionVariables: `{}`,
			Template: &node.XrayTemplate{
				Id:               3,
				Name:             "inbound",
				Type:             node.XrayTemplateInbound,
				Config:           `{"protocol":"vless"}`,
				DefaultVariables: `{"port":443}`,
			},
		},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("server xray template rows len = %d, want 1", len(got))
	}
	if got[0].Id != 12 || got[0].Template == nil || got[0].Template.Id != 3 {
		t.Fatalf("server xray template row mismatch: %#v", got[0])
	}
	if string(got[0].Variables) != `{"port":443}` || string(got[0].Template.Config) != `{"protocol":"vless"}` {
		t.Fatalf("raw json not mapped: binding=%s template=%s", got[0].Variables, got[0].Template.Config)
	}

	empty, err := serverXrayTemplateRowsToTypes(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if empty == nil {
		t.Fatal("empty server xray template rows must return an empty slice, not nil")
	}
	payload, err := json.Marshal(struct {
		List []types.ServerXrayTemplate `json:"list"`
	}{List: empty})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"list":[]}` {
		t.Fatalf("empty server xray template rows marshal = %s, want list []", payload)
	}
}

func TestServerXrayTemplateRowsToTypesSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*node.ServerXrayTemplate, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &node.ServerXrayTemplate{Id: 1, Variables: `{}`, SubscriptionVariables: `{}`}
	rows[len(rows)-1] = &node.ServerXrayTemplate{Id: 2, Variables: `{}`, SubscriptionVariables: `{}`}

	got, err := serverXrayTemplateRowsToTypes(rows, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("server xray template rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy server xray template rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("server xray template row order changed: %#v", got)
	}
}

func TestServerXrayTemplateRowsToTypesUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*node.ServerXrayTemplate, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &node.ServerXrayTemplate{Id: int64(index + 1), Variables: `{}`, SubscriptionVariables: `{}`}
	}

	got, err := serverXrayTemplateRowsToTypes(rows, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("server xray template rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("server xray template rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("server xray template row order changed: %#v", got)
	}
}

func TestServerXrayTemplateToTypesDefaultsNilBoolsAndRawObjects(t *testing.T) {
	got, err := serverXrayTemplateToTypes(&node.ServerXrayTemplate{
		Id:                    12,
		ServerId:              7,
		TemplateId:            3,
		Variables:             "",
		SubscriptionVariables: "",
		Enabled:               nil,
		SubscriptionEnabled:   nil,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled || !got.SubscriptionEnabled {
		t.Fatalf("nil binding booleans should default true in API response: %#v", got)
	}
	if string(got.Variables) != `{}` || string(got.SubscriptionVariables) != `{}` {
		t.Fatalf("empty binding JSON should return objects, got variables=%s subscription=%s", got.Variables, got.SubscriptionVariables)
	}
}
