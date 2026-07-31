package server

import (
	"encoding/json"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/xraytemplate"
)

func TestXrayConfigBucketsPreallocatesByTemplateType(t *testing.T) {
	list := []*node.ServerXrayTemplate{
		{Template: &node.XrayTemplate{Type: node.XrayTemplateInbound}},
		{Template: &node.XrayTemplate{Type: node.XrayTemplateInbound}},
		{Template: &node.XrayTemplate{Type: node.XrayTemplateOutbound}},
		{Template: &node.XrayTemplate{Type: node.XrayTemplateDNS}},
		nil,
		{Template: nil},
		{Template: &node.XrayTemplate{Type: "unknown"}},
	}

	buckets := xraytemplate.NewConfigBuckets(list)
	if cap(buckets.Inbounds) != 2 || cap(buckets.Outbounds) != 1 || cap(buckets.DNS) != 1 {
		t.Fatalf("bucket caps = in:%d out:%d dns:%d", cap(buckets.Inbounds), cap(buckets.Outbounds), cap(buckets.DNS))
	}
	if cap(buckets.Routing) != 0 || cap(buckets.Geodata) != 0 {
		t.Fatalf("unused bucket caps = routing:%d geodata:%d", cap(buckets.Routing), cap(buckets.Geodata))
	}
}

func TestBuildRawXrayConfigResponseMarshalsNestedJSON(t *testing.T) {
	raw, err := xraytemplate.MergedConfigBundle{
		Inbounds:  []xraytemplate.ConfigObject{{"tag": "inbound-main", "port": 443}},
		Outbounds: []xraytemplate.ConfigObject{{"tag": "direct", "protocol": "freedom"}},
		DNS:       xraytemplate.ConfigObject{"servers": []interface{}{"1.1.1.1"}},
	}.Raw()
	if err != nil {
		t.Fatal(err)
	}
	resp := buildRawXrayConfigResponse(raw)

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Inbounds  []map[string]interface{} `json:"inbounds"`
		Outbounds []map[string]interface{} `json:"outbounds"`
		DNS       map[string]interface{}   `json:"dns"`
		Templates []map[string]interface{} `json:"templates"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("response should marshal as nested JSON: %v\n%s", err, data)
	}
	if decoded.Inbounds[0]["tag"] != "inbound-main" || decoded.Outbounds[0]["tag"] != "direct" {
		t.Fatalf("decoded response = %#v", decoded)
	}
	if decoded.DNS["servers"].([]interface{})[0] != "1.1.1.1" {
		t.Fatalf("dns = %#v", decoded.DNS)
	}
	if decoded.Templates == nil || len(decoded.Templates) != 0 {
		t.Fatalf("templates should be a stable empty array: %#v\n%s", decoded.Templates, data)
	}
}

func TestBuildRawXrayConfigResponseKeepsEmptyArrayShape(t *testing.T) {
	resp := buildRawXrayConfigResponse(xraytemplate.RawConfigBundle{})
	if resp.Inbounds == nil || resp.Outbounds == nil || resp.Templates == nil {
		t.Fatalf("xray config response arrays should be non-nil: in=%#v out=%#v templates=%#v", resp.Inbounds, resp.Outbounds, resp.Templates)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("response should marshal as JSON object: %v\n%s", err, data)
	}
	for _, key := range []string{"inbounds", "outbounds", "templates"} {
		list, ok := decoded[key].([]any)
		if !ok || len(list) != 0 {
			t.Fatalf("xray config response must include stable empty %s array: %s", key, data)
		}
	}
}
