package server

import (
	"context"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestQueryServerXrayTemplateListOmitsTemplateByDefault(t *testing.T) {
	ctx := context.Background()
	_, nodeModel := setupQueryServerXrayTemplateListTest(t)

	resp, err := NewQueryServerXrayTemplateListLogic(ctx, &svc.ServiceContext{NodeModel: nodeModel}).
		QueryServerXrayTemplateList(&types.QueryServerXrayTemplateListRequest{ServerId: 7})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("binding count = %d, want 1", len(resp.List))
	}
	got := resp.List[0]
	if got.Template != nil {
		t.Fatalf("default binding response should omit duplicated template payload: %#v", got.Template)
	}
	if got.TemplateId != 10 || got.Alias != "inbound-a" || got.Variables == nil || got.SubscriptionVariables == nil {
		t.Fatalf("binding response missing lightweight fields: %#v", got)
	}
	if resp.Status != nil || resp.DesiredConfigHash != "" || resp.DesiredConfigError != "" {
		t.Fatalf("default response should omit status/hash work: %#v", resp)
	}
}

func TestQueryServerXrayTemplateListIncludesTemplateSummaryOnDemand(t *testing.T) {
	ctx := context.Background()
	_, nodeModel := setupQueryServerXrayTemplateListTest(t)
	includeTemplate := true
	includeStatus := true

	resp, err := NewQueryServerXrayTemplateListLogic(ctx, &svc.ServiceContext{NodeModel: nodeModel}).
		QueryServerXrayTemplateList(&types.QueryServerXrayTemplateListRequest{
			ServerId:        7,
			IncludeTemplate: &includeTemplate,
			IncludeStatus:   &includeStatus,
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("binding count = %d, want 1", len(resp.List))
	}
	template := resp.List[0].Template
	if template == nil {
		t.Fatal("expected lightweight template summary")
	}
	if template.Id != 10 || template.Name != "vless-inbound" || template.Type != node.XrayTemplateInbound {
		t.Fatalf("template summary missing identity fields: %#v", template)
	}
	if string(template.Config) != `{"port":443,"settings":{"clients":[]}}` || string(template.DefaultVariables) != `{"port":443}` {
		t.Fatalf("template summary missing node-form fields: %#v", template)
	}
	if template.ConfigTemplate != "" || template.VariablesSchema != nil || template.SubscriptionMeta != nil || template.Description != "" || template.CreatedAt != 0 || template.UpdatedAt != 0 {
		t.Fatalf("template summary loaded editor/runtime-only fields: %#v", template)
	}
	if resp.Status == nil || resp.Status.Status != "offline" {
		t.Fatalf("include_status should return offline fallback without realtime manager: %#v", resp.Status)
	}
}

func TestQueryServerXrayTemplateListKeepsEmptyListShape(t *testing.T) {
	ctx := context.Background()
	_, nodeModel := setupQueryServerXrayTemplateListTest(t)

	resp, err := NewQueryServerXrayTemplateListLogic(ctx, &svc.ServiceContext{NodeModel: nodeModel}).
		QueryServerXrayTemplateList(&types.QueryServerXrayTemplateListRequest{ServerId: 8})
	if err != nil {
		t.Fatal(err)
	}
	if resp.List == nil || len(resp.List) != 0 {
		t.Fatalf("empty server xray template list = %#v, want non-nil empty slice", resp.List)
	}
}

func TestEnabledXrayTemplateBindingsSkipInvalidWithoutInputSizedCapacity(t *testing.T) {
	enabled := true
	list := make([]*node.ServerXrayTemplate, viewutil.SmallRowsCapacityThreshold+20)
	list[0] = &node.ServerXrayTemplate{Id: 1}
	list[1] = &node.ServerXrayTemplate{Id: 2, Template: &node.XrayTemplate{Enabled: &enabled}, Enabled: &enabled}
	list[len(list)-1] = &node.ServerXrayTemplate{Id: 3, Template: &node.XrayTemplate{Enabled: &enabled}, Enabled: &enabled}

	got := enabledXrayTemplateBindings(list)
	if len(got) != 2 {
		t.Fatalf("enabled bindings len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy enabled bindings cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 2 || got[1].Id != 3 {
		t.Fatalf("enabled binding order changed: %#v", got)
	}
}

func TestEnabledXrayTemplateBindingsUseCapacityHintAfterActualThreshold(t *testing.T) {
	enabled := true
	list := make([]*node.ServerXrayTemplate, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		list[index] = &node.ServerXrayTemplate{
			Id:       int64(index + 1),
			Enabled:  &enabled,
			Template: &node.XrayTemplate{Enabled: &enabled},
		}
	}

	got := enabledXrayTemplateBindings(list)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("enabled bindings len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("enabled bindings cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("enabled binding order changed: %#v", got)
	}
}

func setupQueryServerXrayTemplateListTest(t *testing.T) (*gorm.DB, node.Model) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&node.Server{}, &node.XrayTemplate{}, &node.ServerXrayTemplate{}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	server := node.Server{Id: 7, Name: "edge", Address: "node.example.com"}
	if err := db.Create(&server).Error; err != nil {
		t.Fatal(err)
	}
	template := node.XrayTemplate{
		Id:               10,
		Name:             "vless-inbound",
		Type:             node.XrayTemplateInbound,
		Config:           `{"port":443,"settings":{"clients":[]}}`,
		ConfigTemplate:   `{"large":"template"}`,
		VariablesSchema:  `{"port":{"type":"number"}}`,
		DefaultVariables: `{"port":443}`,
		SubscriptionMeta: `{"type":"vless"}`,
		Description:      "editor-only description",
		Enabled:          &enabled,
	}
	if err := db.Create(&template).Error; err != nil {
		t.Fatal(err)
	}
	binding := node.ServerXrayTemplate{
		Id:                    99,
		ServerId:              server.Id,
		TemplateId:            template.Id,
		Alias:                 "inbound-a",
		Variables:             `{"flow":"xtls-rprx-vision"}`,
		SubscriptionEnabled:   &enabled,
		SubscriptionName:      "{{ .Server.Name }}",
		SubscriptionVariables: `{"security":"reality"}`,
		Sort:                  1,
		Enabled:               &enabled,
	}
	if err := db.Create(&binding).Error; err != nil {
		t.Fatal(err)
	}
	return db, node.NewModel(db, nil)
}

func TestDesiredXrayConfigHashTracksAllRuntimeSections(t *testing.T) {
	enabled, disabled := true, false
	srv := &node.Server{Id: 1, Name: "test", Address: "node.example.com"}
	for _, tc := range []struct {
		name, kind, before, after string
	}{
		{"outbound settings", node.XrayTemplateOutbound, `{"tag":"proxy","protocol":"socks","settings":{"address":"old.example.com","port":1080}}`, `{"tag":"proxy","protocol":"socks","settings":{"address":"new.example.com","port":1081}}`},
		{"outbound transport", node.XrayTemplateOutbound, `{"tag":"proxy","protocol":"socks","streamSettings":{"network":"raw"}}`, `{"tag":"proxy","protocol":"socks","streamSettings":{"network":"ws","wsSettings":{"path":"/new"}}}`},
		{"inbound", node.XrayTemplateInbound, `{"tag":"main","port":443,"protocol":"vless","settings":{"clients":[]}}`, `{"tag":"main","port":8443,"protocol":"vless","settings":{"clients":[]}}`},
		{"DNS", node.XrayTemplateDNS, `{"servers":["1.1.1.1"]}`, `{"servers":["8.8.8.8"]}`},
		{"routing", node.XrayTemplateRouting, `{"rules":[]}`, `{"rules":[{"type":"field","domain":["example.com"],"outboundTag":"direct"}]}`},
		{"geodata", node.XrayTemplateGeodata, `{"cron":"0 4 * * *"}`, `{"cron":"0 5 * * *"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binding := &node.ServerXrayTemplate{
				TemplateId: 2, Alias: "main", Enabled: &enabled,
				Template: &node.XrayTemplate{Id: 2, Type: tc.kind, Config: tc.before, Enabled: &enabled},
			}
			bindings := []*node.ServerXrayTemplate{
				{TemplateId: 1, Alias: "direct", Enabled: &enabled, Template: &node.XrayTemplate{Id: 1, Type: node.XrayTemplateOutbound, Config: `{"tag":"direct","protocol":"freedom"}`, Enabled: &enabled}},
				binding,
			}
			before, err := desiredXrayConfigHash(srv, enabledXrayTemplateBindings(bindings))
			if err != nil {
				t.Fatal(err)
			}
			binding.Template.Config = tc.after
			after, err := desiredXrayConfigHash(srv, enabledXrayTemplateBindings(bindings))
			if err != nil {
				t.Fatal(err)
			}
			if before == after {
				t.Fatal("runtime config edit did not change desired hash")
			}
			binding.Template.Enabled = &disabled
			disabledHash, err := desiredXrayConfigHash(srv, enabledXrayTemplateBindings(bindings))
			if err != nil {
				t.Fatal(err)
			}
			binding.Template.Config = tc.before
			unchanged, err := desiredXrayConfigHash(srv, enabledXrayTemplateBindings(bindings))
			if err != nil {
				t.Fatal(err)
			}
			if disabledHash != unchanged {
				t.Fatal("disabled template changed runtime config")
			}
		})
	}
}
