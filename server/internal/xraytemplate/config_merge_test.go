package xraytemplate

import (
	"reflect"
	"testing"
)

func TestJSONHelpersFallback(t *testing.T) {
	if got := JSONToObject("[]"); !reflect.DeepEqual(got, ConfigObject{}) {
		t.Fatalf("JSONToObject array input = %#v, want empty object", got)
	}
}

func TestMergeDNS(t *testing.T) {
	got := MergeDNS([]ConfigObject{
		ConfigObject{
			"servers": []any{"1.1.1.1"},
			"hosts": ConfigObject{
				"one.example": "1.1.1.1",
			},
		},
		ConfigObject{
			"servers": []any{"8.8.8.8"},
			"hosts": ConfigObject{
				"two.example": "8.8.8.8",
			},
		},
	})

	if !reflect.DeepEqual(got["servers"], []any{"1.1.1.1", "8.8.8.8"}) {
		t.Fatalf("servers = %#v", got["servers"])
	}
	hosts := got["hosts"].(ConfigObject)
	if hosts["one.example"] != "1.1.1.1" || hosts["two.example"] != "8.8.8.8" {
		t.Fatalf("hosts = %#v", hosts)
	}
}

func TestMergeRoutingDeduplicatesLists(t *testing.T) {
	rule := ConfigObject{"type": "field", "outboundTag": "direct"}
	got := MergeRouting([]ConfigObject{
		ConfigObject{"rules": []any{rule}},
		ConfigObject{"rules": []any{rule}},
	})

	rules := got["rules"].([]any)
	if len(rules) != 1 {
		t.Fatalf("len(rules) = %d, want 1", len(rules))
	}
}

func TestMergeGeodata(t *testing.T) {
	got := MergeGeodata([]ConfigObject{
		ConfigObject{"assets": []any{"geoip.dat"}},
		ConfigObject{"assets": []any{"geosite.dat"}},
	})

	if !reflect.DeepEqual(got["assets"], []any{"geoip.dat", "geosite.dat"}) {
		t.Fatalf("assets = %#v", got["assets"])
	}
}
