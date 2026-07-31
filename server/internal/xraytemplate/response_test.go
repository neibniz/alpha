package xraytemplate

import (
	"encoding/json"
	"testing"
)

func TestRawConfigListKeepsEmptyArrayShape(t *testing.T) {
	got, err := RawConfigList(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("empty raw config list = %#v, want non-nil empty slice", got)
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "[]" {
		t.Fatalf("empty raw config list JSON = %s, want []", data)
	}
}

func TestMergedConfigBundleRawMarshalsSections(t *testing.T) {
	raw, err := MergedConfigBundle{
		Inbounds:  []ConfigObject{{"tag": "inbound-main", "port": 443}},
		Outbounds: []ConfigObject{{"tag": "direct", "protocol": "freedom"}},
		DNS:       ConfigObject{"servers": []any{"1.1.1.1"}},
	}.Raw()
	if err != nil {
		t.Fatal(err)
	}
	if string(raw.Inbounds[0]) != `{"port":443,"tag":"inbound-main"}` {
		t.Fatalf("inbound raw = %s", raw.Inbounds[0])
	}
	if string(raw.Outbounds[0]) != `{"protocol":"freedom","tag":"direct"}` {
		t.Fatalf("outbound raw = %s", raw.Outbounds[0])
	}
	if string(raw.DNS) != `{"servers":["1.1.1.1"]}` {
		t.Fatalf("dns raw = %s", raw.DNS)
	}
	if raw.Routing != nil || raw.Geodata != nil {
		t.Fatalf("empty optional sections should stay nil: routing=%s geodata=%s", raw.Routing, raw.Geodata)
	}
}

func TestRawConfigValueCopiesInputRawMessage(t *testing.T) {
	source := json.RawMessage(`{"tag":"direct"}`)
	got, err := RawConfigValue(source)
	if err != nil {
		t.Fatal(err)
	}
	source[8] = 'x'
	if string(got) != `{"tag":"direct"}` {
		t.Fatalf("raw config value = %s, want independent copy", got)
	}
}

func TestMergedConfigBundleValidatesRoutingReferences(t *testing.T) {
	bundle := MergedConfigBundle{
		Outbounds: []ConfigObject{{"tag": "direct"}},
		Routing: ConfigObject{
			"rules": []any{
				ConfigObject{"type": "field", "outboundTag": "missing"},
			},
		},
	}
	if err := bundle.ValidateRoutingReferences(); err == nil {
		t.Fatal("expected missing outbound reference error")
	}
}
