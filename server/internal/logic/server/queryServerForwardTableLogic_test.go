package server

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/xraytemplate"
)

type testStringer string

func (s testStringer) String() string {
	return string(s)
}

func TestForwardTableNodeColumnsStayLightweight(t *testing.T) {
	assertColumns(t, forwardTableNodeColumns, []string{
		"id",
		"name",
		"port",
		"address",
		"server_id",
		"protocol",
		"inbound_alias",
	}, []string{
		"tags",
		"enabled",
		"sort",
		"created_at",
		"updated_at",
	})
}

func TestForwardTableServerColumnsStayLightweight(t *testing.T) {
	assertColumns(t, forwardTableServerColumns, []string{
		"id",
		"name",
		"address",
	}, []string{
		"country",
		"city",
		"sort",
		"last_reported_at",
		"created_at",
		"updated_at",
	})
}

func TestConfigStringReadsRenderedConfigField(t *testing.T) {
	cfg := xraytemplate.ConfigObject{
		"tag":      "inbound-a",
		"stringer": testStringer("inbound-b"),
	}

	if got := configString(cfg, "tag"); got != "inbound-a" {
		t.Fatalf("tag = %q, want inbound-a", got)
	}
	if got := configString(cfg, "stringer"); got != "inbound-b" {
		t.Fatalf("stringer = %q, want inbound-b", got)
	}
	if got := configString(cfg, "missing"); got != "" {
		t.Fatalf("missing = %q, want empty", got)
	}
}

func TestConfigUintReadsRenderedConfigPort(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  uint
	}{
		{name: "uint", value: uint(8443), want: 8443},
		{name: "uint16", value: uint16(443), want: 443},
		{name: "int", value: 2080, want: 2080},
		{name: "int64", value: int64(2081), want: 2081},
		{name: "float64", value: float64(2082), want: 2082},
		{name: "json number", value: json.Number("2083"), want: 2083},
		{name: "negative", value: -1, want: 0},
		{name: "invalid", value: "not-a-port", want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := configUint(xraytemplate.ConfigObject{"port": tc.value}, "port")
			if got != tc.want {
				t.Fatalf("port = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestAppendUniqueForwardIPUsesLinearDedupForSmallInput(t *testing.T) {
	var values []string
	var seen map[string]struct{}
	values, seen = appendUniqueForwardIP(values, "192.0.2.1", seen)
	values, seen = appendUniqueForwardIP(values, "192.0.2.2", seen)
	values, seen = appendUniqueForwardIP(values, "192.0.2.1", seen)
	values, seen = appendUniqueForwardIP(values, "", seen)

	if seen != nil {
		t.Fatal("small forward ip set should not allocate map")
	}
	if len(values) != 2 || values[0] != "192.0.2.1" || values[1] != "192.0.2.2" {
		t.Fatalf("forward ips = %#v, want two unique ips in first-seen order", values)
	}
}

func TestForwardInboundPortIndexUsesLinearPathForSmallInput(t *testing.T) {
	index := &forwardInboundPortIndex{}
	index.Set("VLESS", 443)
	index.Set("Trojan", 8443)
	index.Set(" VLESS ", 444)
	index.Set("", 9443)
	index.Set("zero", 0)

	if index.byAlias != nil {
		t.Fatal("small forward inbound port index should not allocate map")
	}
	if index.Len() != 2 {
		t.Fatalf("forward inbound port index len = %d, want 2", index.Len())
	}
	if got, ok := index.Get("VLESS"); !ok || got != 444 {
		t.Fatalf("forward inbound port index get(VLESS) = %d, ok=%v", got, ok)
	}
	if got, ok := index.Get("missing"); ok || got != 0 {
		t.Fatalf("forward inbound port index get(missing) = %d, ok=%v", got, ok)
	}
}

func TestForwardInboundPortIndexUpgradesLargeInputToMap(t *testing.T) {
	index := &forwardInboundPortIndex{}
	for i := 1; i <= smallForwardUniqueThreshold+1; i++ {
		index.Set("inbound-"+strconv.Itoa(i), uint(1000+i))
	}
	if index.byAlias == nil {
		t.Fatal("large forward inbound port index should allocate map")
	}
	if index.Len() != smallForwardUniqueThreshold+1 {
		t.Fatalf("forward inbound port index len = %d, want %d", index.Len(), smallForwardUniqueThreshold+1)
	}
	lastAlias := "inbound-" + strconv.Itoa(smallForwardUniqueThreshold+1)
	if got, ok := index.Get(lastAlias); !ok || got != uint(1000+smallForwardUniqueThreshold+1) {
		t.Fatalf("forward inbound port index get(last) = %d, ok=%v", got, ok)
	}
}

func TestForwardInboundPortIndexBoundsSparseUpgradeCapacity(t *testing.T) {
	index := newForwardInboundPortIndexWithHint(smallForwardUniqueThreshold + 20)
	if index.rows != nil || index.byAlias != nil {
		t.Fatalf("large hinted forward inbound port index should not allocate before first alias: %#v", index)
	}
	index.Set("inbound", 443)
	index.Set("inbound", 8443)
	if index.byAlias != nil || index.Len() != 1 {
		t.Fatalf("duplicate hinted forward inbound port index should stay linear: %#v", index)
	}
	if got, ok := index.Get("inbound"); !ok || got != 8443 {
		t.Fatalf("forward inbound port index get(inbound) = %d, ok=%v", got, ok)
	}
	for i := 2; i <= smallForwardUniqueThreshold+1; i++ {
		index.Set("inbound-"+strconv.Itoa(i), uint(1000+i))
	}
	if index.byAlias == nil {
		t.Fatal("hinted forward inbound port index should allocate map after actual unique aliases exceed threshold")
	}
	if maxCap := index.Len() * 2; cap(index.rows) > maxCap {
		t.Fatalf("forward inbound port index cap = %d, want at most %d", cap(index.rows), maxCap)
	}
}

func TestAddForwardInboundPortAliasesUsesRenderedAliasVariants(t *testing.T) {
	index := &forwardInboundPortIndex{}
	addForwardInboundPortAliases(index, &node.ServerXrayTemplate{
		Alias: "VLESS Reality gRPC",
		Template: &node.XrayTemplate{
			Name: "Main Inbound",
			Type: node.XrayTemplateInbound,
		},
	}, xraytemplate.ConfigObject{
		"tag":  "rendered-vless",
		"port": 32001,
	}, "inbound_1")

	for _, alias := range []string{
		"VLESS Reality gRPC",
		"VLESS Reality gRPC Inbound",
		"Main",
		"Main Inbound",
		"rendered-vless",
	} {
		if got, ok := index.Get(alias); !ok || got != 32001 {
			t.Fatalf("forward inbound port index get(%q) = %d, ok=%v; want 32001", alias, got, ok)
		}
	}
}

func TestAddForwardInboundPortAliasesUsesFallbackForAnonymousInbound(t *testing.T) {
	index := &forwardInboundPortIndex{}
	addForwardInboundPortAliases(index, &node.ServerXrayTemplate{
		Template: &node.XrayTemplate{Type: node.XrayTemplateInbound},
	}, xraytemplate.ConfigObject{"port": 32002}, "inbound_1")

	if got, ok := index.Get("inbound_1"); !ok || got != 32002 {
		t.Fatalf("forward inbound fallback port = %d, ok=%v; want 32002", got, ok)
	}
}

func TestForwardInboundPortCacheUsesLinearPathForSmallInput(t *testing.T) {
	var cache forwardInboundPortCache
	cache.Set(7, newForwardInboundPortIndexFromMap(map[string]uint{"vless": 443}))
	cache.Set(9, nil)
	cache.Set(7, newForwardInboundPortIndexFromMap(map[string]uint{"trojan": 8443}))

	if cache.byServer != nil {
		t.Fatal("small forward inbound port cache should not allocate map")
	}
	if cache.Len() != 2 {
		t.Fatalf("forward inbound port cache len = %d, want 2", cache.Len())
	}
	if got, ok := cache.Get(7); !ok || got == nil {
		t.Fatalf("forward inbound port cache get(7) = %#v, ok=%v", got, ok)
	} else if port, found := got.Get("trojan"); !found || port != 8443 {
		t.Fatalf("forward inbound port cache get(7).Get(trojan) = %d, found=%v", port, found)
	} else if port, found := got.Get("vless"); found || port != 0 {
		t.Fatalf("forward inbound port cache get(7).Get(vless) = %d, found=%v", port, found)
	}
	if got, ok := cache.Get(9); !ok || got != nil {
		t.Fatalf("forward inbound port cache should cache nil ports: %#v, ok=%v", got, ok)
	}
	if got, ok := cache.Get(99); ok || got != nil {
		t.Fatalf("forward inbound port cache get(missing) = %#v, ok=%v", got, ok)
	}
}

func TestForwardInboundPortCacheUpgradesLargeInputToMap(t *testing.T) {
	var cache forwardInboundPortCache
	for id := int64(1); id <= smallForwardUniqueThreshold+1; id++ {
		cache.Set(id, newForwardInboundPortIndexFromMap(map[string]uint{"inbound": uint(1000 + id)}))
	}
	if cache.byServer == nil {
		t.Fatal("large forward inbound port cache should allocate map")
	}
	if cache.Len() != smallForwardUniqueThreshold+1 {
		t.Fatalf("forward inbound port cache len = %d, want %d", cache.Len(), smallForwardUniqueThreshold+1)
	}
	lastID := int64(smallForwardUniqueThreshold + 1)
	if got, ok := cache.Get(lastID); !ok || got == nil {
		t.Fatalf("forward inbound port cache get(last) = %#v, ok=%v", got, ok)
	} else if port, found := got.Get("inbound"); !found || port != uint(1000+lastID) {
		t.Fatalf("forward inbound port cache get(last).Get(inbound) = %d, found=%v", port, found)
	}
}

func TestForwardInboundPortCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := newForwardInboundPortCache(smallForwardUniqueThreshold + 20)
	if cache.rows != nil || cache.byServer != nil {
		t.Fatalf("large hinted forward inbound port cache should not allocate before first server: %#v", cache)
	}
	cache.Set(7, newForwardInboundPortIndexFromMap(map[string]uint{"vless": 443}))
	cache.Set(7, newForwardInboundPortIndexFromMap(map[string]uint{"trojan": 8443}))
	if cache.byServer != nil || cache.Len() != 1 {
		t.Fatalf("duplicate hinted forward inbound port cache should stay linear: %#v", cache)
	}
	if got, ok := cache.Get(7); !ok || got == nil {
		t.Fatalf("forward inbound port cache get(7) = %#v, ok=%v", got, ok)
	} else if port, found := got.Get("trojan"); !found || port != 8443 {
		t.Fatalf("forward inbound port cache get(7).Get(trojan) = %d, found=%v", port, found)
	}
	for id := int64(8); id <= int64(smallForwardUniqueThreshold+7); id++ {
		cache.Set(id, newForwardInboundPortIndexFromMap(map[string]uint{"inbound": uint(1000 + id)}))
	}
	if cache.byServer == nil {
		t.Fatal("hinted forward inbound port cache should allocate map after actual unique servers exceed threshold")
	}
	if maxCap := cache.Len() * 2; cap(cache.rows) > maxCap {
		t.Fatalf("forward inbound port cache cap = %d, want at most %d", cap(cache.rows), maxCap)
	}
}

func TestForwardPortOwnerCacheUsesLinearPathForSmallInput(t *testing.T) {
	var cache forwardPortOwnerCache
	cache.Set(443, "node 1 inbound vless")
	cache.Set(8443, "node 2 inbound trojan")
	cache.Set(443, "node 1 inbound vless-updated")

	if cache.byPort != nil {
		t.Fatal("small forward port owner cache should not allocate map")
	}
	if cache.Len() != 2 {
		t.Fatalf("forward port owner cache len = %d, want 2", cache.Len())
	}
	if got, ok := cache.Get(443); !ok || got != "node 1 inbound vless-updated" {
		t.Fatalf("forward port owner cache get(443) = %q, ok=%v", got, ok)
	}
	if got, ok := cache.Get(9443); ok || got != "" {
		t.Fatalf("forward port owner cache get(missing) = %q, ok=%v", got, ok)
	}
}

func TestForwardPortOwnerCacheUpgradesLargeInputToMap(t *testing.T) {
	var cache forwardPortOwnerCache
	for i := uint(1); i <= smallForwardUniqueThreshold+1; i++ {
		cache.Set(1000+i, "owner-"+strconv.FormatUint(uint64(i), 10))
	}
	if cache.byPort == nil {
		t.Fatal("large forward port owner cache should allocate map")
	}
	if cache.Len() != smallForwardUniqueThreshold+1 {
		t.Fatalf("forward port owner cache len = %d, want %d", cache.Len(), smallForwardUniqueThreshold+1)
	}
	lastPort := uint(1000 + smallForwardUniqueThreshold + 1)
	if got, ok := cache.Get(lastPort); !ok || got != "owner-"+strconv.Itoa(smallForwardUniqueThreshold+1) {
		t.Fatalf("forward port owner cache get(last) = %q, ok=%v", got, ok)
	}
}

func TestForwardPortOwnerCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := newForwardPortOwnerCache(smallForwardUniqueThreshold + 20)
	if cache.rows != nil || cache.byPort != nil {
		t.Fatalf("large hinted forward port owner cache should not allocate before first port: %#v", cache)
	}
	cache.Set(443, "node 1 inbound vless")
	cache.Set(443, "node 1 inbound vless-updated")
	if cache.byPort != nil || cache.Len() != 1 {
		t.Fatalf("duplicate hinted forward port owner cache should stay linear: %#v", cache)
	}
	if got, ok := cache.Get(443); !ok || got != "node 1 inbound vless-updated" {
		t.Fatalf("forward port owner cache get(443) = %q, ok=%v", got, ok)
	}
	for i := uint(1); i <= smallForwardUniqueThreshold; i++ {
		cache.Set(1000+i, "owner-"+strconv.FormatUint(uint64(i), 10))
	}
	if cache.byPort == nil {
		t.Fatal("hinted forward port owner cache should allocate map after actual unique ports exceed threshold")
	}
	if maxCap := cache.Len() * 2; cap(cache.rows) > maxCap {
		t.Fatalf("forward port owner cache cap = %d, want at most %d", cap(cache.rows), maxCap)
	}
}

func TestForwardHostIPCacheUsesLinearPathForSmallInput(t *testing.T) {
	var cache forwardHostIPCache
	cache.Set("node.example.com", []string{"192.0.2.1"})
	cache.Set("server.example.com", nil)
	cache.Set("node.example.com", []string{"192.0.2.2"})

	if cache.byHost != nil {
		t.Fatal("small forward host ip cache should not allocate map")
	}
	if cache.Len() != 2 {
		t.Fatalf("forward host ip cache len = %d, want 2", cache.Len())
	}
	if got, ok := cache.Get("node.example.com"); !ok || len(got) != 1 || got[0] != "192.0.2.2" {
		t.Fatalf("forward host ip cache get(node) = %#v, ok=%v", got, ok)
	}
	if got, ok := cache.Get("server.example.com"); !ok || got != nil {
		t.Fatalf("forward host ip cache should cache nil IP list: %#v, ok=%v", got, ok)
	}
	if got, ok := cache.Get("missing.example.com"); ok || got != nil {
		t.Fatalf("forward host ip cache get(missing) = %#v, ok=%v", got, ok)
	}
}

func TestForwardHostIPCacheUpgradesLargeInputToMap(t *testing.T) {
	var cache forwardHostIPCache
	for i := 1; i <= smallForwardUniqueThreshold+1; i++ {
		host := "host-" + strconv.Itoa(i) + ".example.com"
		cache.Set(host, []string{"192.0.2." + strconv.Itoa(i)})
	}
	if cache.byHost == nil {
		t.Fatal("large forward host ip cache should allocate map")
	}
	if cache.Len() != smallForwardUniqueThreshold+1 {
		t.Fatalf("forward host ip cache len = %d, want %d", cache.Len(), smallForwardUniqueThreshold+1)
	}
	lastHost := "host-" + strconv.Itoa(smallForwardUniqueThreshold+1) + ".example.com"
	if got, ok := cache.Get(lastHost); !ok || len(got) != 1 || got[0] != "192.0.2."+strconv.Itoa(smallForwardUniqueThreshold+1) {
		t.Fatalf("forward host ip cache get(last) = %#v, ok=%v", got, ok)
	}
}

func TestForwardHostIPCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := newForwardHostIPCache(smallForwardUniqueThreshold + 20)
	if cache.rows != nil || cache.byHost != nil {
		t.Fatalf("large hinted forward host ip cache should not allocate before first host: %#v", cache)
	}
	cache.Set("node.example.com", []string{"192.0.2.1"})
	cache.Set("node.example.com", []string{"192.0.2.2"})
	if cache.byHost != nil || cache.Len() != 1 {
		t.Fatalf("duplicate hinted forward host ip cache should stay linear: %#v", cache)
	}
	if got, ok := cache.Get("node.example.com"); !ok || len(got) != 1 || got[0] != "192.0.2.2" {
		t.Fatalf("forward host ip cache get(node) = %#v, ok=%v", got, ok)
	}
	for i := 2; i <= smallForwardUniqueThreshold+1; i++ {
		host := "host-" + strconv.Itoa(i) + ".example.com"
		cache.Set(host, []string{"192.0.2." + strconv.Itoa(i)})
	}
	if cache.byHost == nil {
		t.Fatal("hinted forward host ip cache should allocate map after actual unique hosts exceed threshold")
	}
	if maxCap := cache.Len() * 2; cap(cache.rows) > maxCap {
		t.Fatalf("forward host ip cache cap = %d, want at most %d", cap(cache.rows), maxCap)
	}
}

func TestAppendUniqueForwardIPUpgradesLargeInputToMap(t *testing.T) {
	var values []string
	var seen map[string]struct{}
	for i := 1; i <= smallForwardUniqueThreshold+1; i++ {
		values, seen = appendUniqueForwardIP(values, "192.0.2."+strconv.Itoa(i), seen)
	}
	values, seen = appendUniqueForwardIP(values, "192.0.2.1", seen)

	if seen == nil {
		t.Fatal("large forward ip set should upgrade to map")
	}
	if len(values) != smallForwardUniqueThreshold+1 {
		t.Fatalf("forward ips = %#v, want %d unique ips", values, smallForwardUniqueThreshold+1)
	}
}

func TestAppendUniqueForwardIPUpgradesAfterActualUniqueThreshold(t *testing.T) {
	values := make([]string, 0, smallForwardUniqueThreshold+1)
	var seen map[string]struct{}
	values, seen = appendUniqueForwardIP(values, "192.0.2.1", seen)
	if seen != nil {
		t.Fatal("large hinted forward ip set should not allocate map for one unique value")
	}
	values, seen = appendUniqueForwardIP(values, "192.0.2.1", seen)
	if len(values) != 1 || seen != nil {
		t.Fatalf("duplicate hinted forward ip set should stay linear: values=%#v seen=%#v", values, seen)
	}
	for i := 2; i <= smallForwardUniqueThreshold+1; i++ {
		values, seen = appendUniqueForwardIP(values, "192.0.2."+strconv.Itoa(i), seen)
	}
	if seen == nil {
		t.Fatal("hinted forward ip set should allocate map after actual unique values exceed threshold")
	}
}

func TestAppendUniqueForwardIPSmallInputAvoidsMapAllocation(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		values := make([]string, 0, 3)
		var seen map[string]struct{}
		values, seen = appendUniqueForwardIP(values, "192.0.2.1", seen)
		values, seen = appendUniqueForwardIP(values, "192.0.2.2", seen)
		values, seen = appendUniqueForwardIP(values, "192.0.2.1", seen)
		_ = values
		_ = seen
	})
	if allocs > 1 {
		t.Fatalf("appendUniqueForwardIP small input allocations = %f, want at most 1", allocs)
	}
}
