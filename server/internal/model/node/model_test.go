package node

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestServerCacheKeyHelpersKeepStableShape(t *testing.T) {
	if got, want := serverUserListCacheKey(42), "server:user:42"; got != want {
		t.Fatalf("serverUserListCacheKey() = %q, want %q", got, want)
	}
	if got, want := serverConfigCacheScanPattern(42), "server:config:42*"; got != want {
		t.Fatalf("serverConfigCacheScanPattern() = %q, want %q", got, want)
	}
}

func TestXrayTemplateListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(xrayTemplateListColumns))
	for _, column := range xrayTemplateListColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		`"xray_templates"."id"`,
		`"xray_templates"."name"`,
		`"xray_templates"."type"`,
		`"xray_templates"."description"`,
		`"xray_templates"."enabled"`,
		`"xray_templates"."updated_at"`,
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("xray template list columns must include %q", column)
		}
	}

	selection := strings.Join(xrayTemplateListColumns, "\n")
	for _, alias := range []string{"AS binding_count", "AS subscription_binding_count"} {
		if !strings.Contains(selection, alias) {
			t.Fatalf("xray template list columns must include %q", alias)
		}
	}

	forbidden := []string{`"xray_templates"."config"`, "config_template", "variables_schema", "default_variables", "subscription_meta", "created_at"}
	for _, column := range forbidden {
		if strings.Contains(selection, column) {
			t.Fatalf("xray template list columns must not include %q", column)
		}
	}
}

func TestXrayTemplateCatalogColumnsStayFocused(t *testing.T) {
	columns := make(map[string]struct{}, len(xrayTemplateCatalogColumns))
	for _, column := range xrayTemplateCatalogColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "name", "type", "config", "config_template", "variables_schema", "default_variables", "subscription_meta", "description", "enabled"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("xray template catalog columns must include %q", column)
		}
	}

	forbidden := []string{"created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("xray template catalog columns must not include %q", column)
		}
	}
}

func TestServerXrayTemplateRuntimeColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(serverXrayTemplateRuntimeColumns))
	for _, column := range serverXrayTemplateRuntimeColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		`"server_xray_templates"."id"`,
		`"server_xray_templates"."server_id"`,
		`"server_xray_templates"."template_id"`,
		`"server_xray_templates"."alias"`,
		`"server_xray_templates"."variables"`,
		`"server_xray_templates"."subscription_enabled"`,
		`"server_xray_templates"."subscription_name"`,
		`"server_xray_templates"."subscription_variables"`,
		`"server_xray_templates"."sort"`,
		`"server_xray_templates"."enabled"`,
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("server xray template runtime columns must include %q", column)
		}
	}

	forbidden := []string{`"server_xray_templates"."created_at"`, `"server_xray_templates"."updated_at"`}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("server xray template runtime columns must not include %q", column)
		}
	}
}

func TestXrayTemplateRuntimeColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(xrayTemplateRuntimeColumns))
	for _, column := range xrayTemplateRuntimeColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "name", "type", "config", "config_template", "default_variables", "subscription_meta", "enabled"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("xray template runtime columns must include %q", column)
		}
	}

	forbidden := []string{"description", "variables_schema", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("xray template runtime columns must not include %q", column)
		}
	}
}

func TestXrayTemplateBindingSummaryColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(xrayTemplateBindingSummaryColumns))
	for _, column := range xrayTemplateBindingSummaryColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "name", "type", "config", "default_variables", "enabled"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("xray template binding summary columns must include %q", column)
		}
	}

	forbidden := []string{"config_template", "variables_schema", "subscription_meta", "description", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("xray template binding summary columns must not include %q", column)
		}
	}
}

func TestServerExists(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Server{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Server{Id: 7, Name: "edge", Address: "edge.example.com"}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	exists, err := model.ServerExists(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("server 7 should exist")
	}
	exists, err = model.ServerExists(ctx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("server 8 should not exist")
	}
}

func TestEnsureModelExistsByIDSelectsOnlyID(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Server{}, &Node{}, &XrayTemplate{}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	if err := db.Create(&Server{Id: 7, Name: "edge", Address: "edge.example.com"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Node{Id: 8, Name: "node", ServerId: 7, Protocol: "vless", InboundAlias: "vless", Enabled: &enabled}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&XrayTemplate{Id: 9, Name: "template", Type: XrayTemplateInbound, Config: `{"tag":"in"}`, Enabled: &enabled}).Error; err != nil {
		t.Fatal(err)
	}

	selectsByTable := map[string][]string{}
	if err := db.Callback().Query().Before("gorm:query").Register("alpha:test_record_selects", func(tx *gorm.DB) {
		if tx.Statement.Schema == nil {
			return
		}
		table := tx.Statement.Schema.Table
		if _, ok := selectsByTable[table]; ok {
			return
		}
		switch table {
		case "servers", "nodes", "xray_templates":
			selectsByTable[table] = append([]string(nil), tx.Statement.Selects...)
		}
	}); err != nil {
		t.Fatal(err)
	}

	model := newServerModel(db, nil)
	if err := model.ensureModelExistsByID(ctx, &Server{}, 7); err != nil {
		t.Fatal(err)
	}
	if err := model.ensureModelExistsByID(ctx, &Node{}, 8); err != nil {
		t.Fatal(err)
	}
	if err := model.ensureModelExistsByID(ctx, &XrayTemplate{}, 9); err != nil {
		t.Fatal(err)
	}

	want := []string{"id"}
	for _, table := range []string{"servers", "nodes", "xray_templates"} {
		if !reflect.DeepEqual(selectsByTable[table], want) {
			t.Fatalf("%s existence selects = %#v, want %#v", table, selectsByTable[table], want)
		}
	}
}

func TestQueryServerXrayTemplatesWithOptionsSkipsTemplatePreload(t *testing.T) {
	ctx := context.Background()
	db := setupServerXrayTemplateOptionsTestDB(t)

	list, err := NewModel(db, nil).QueryServerXrayTemplatesWithOptions(ctx, 7, QueryServerXrayTemplateOptions{
		TemplatePreload: XrayTemplatePreloadNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("bindings = %d, want 2", len(list))
	}
	if list[0].Template != nil || list[1].Template != nil {
		t.Fatalf("default admin binding query should not preload templates: %#v %#v", list[0].Template, list[1].Template)
	}
	if list[0].Id != 10 || list[0].TemplateId != 1 || list[0].Alias != "vless" || list[0].Variables == "" || list[0].SubscriptionVariables == "" {
		t.Fatalf("binding query dropped required binding fields: %#v", list[0])
	}
}

func TestQueryServerXrayTemplatesWithOptionsSummaryPreload(t *testing.T) {
	ctx := context.Background()
	db := setupServerXrayTemplateOptionsTestDB(t)

	list, err := NewModel(db, nil).QueryServerXrayTemplatesWithOptions(ctx, 7, QueryServerXrayTemplateOptions{
		TemplatePreload: XrayTemplatePreloadSummary,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("bindings = %d, want 2", len(list))
	}
	got := list[0].Template
	if got == nil {
		t.Fatal("summary query should preload template summary")
	}
	if got.Id != 1 || got.Name != "vless" || got.Type != XrayTemplateInbound || got.Config == "" || got.DefaultVariables == "" || got.Enabled == nil || !*got.Enabled {
		t.Fatalf("summary template dropped required fields: %#v", got)
	}
	if got.ConfigTemplate != "" || got.VariablesSchema != "" || got.SubscriptionMeta != "" || got.Description != "" || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("summary template loaded editor/runtime-only fields: %#v", got)
	}
}

func TestQueryServerXrayTemplatesEnabledOnlySelectsRuntimeColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&XrayTemplate{}, &ServerXrayTemplate{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	enabled := true
	disabled := false
	templates := []*XrayTemplate{
		{
			Id:               1,
			Name:             "vless",
			Type:             XrayTemplateInbound,
			Config:           `{"protocol":"vless","port":443}`,
			ConfigTemplate:   `{"protocol":"{{ .Vars.protocol }}","port":443}`,
			VariablesSchema:  `{"protocol":{"type":"string"}}`,
			DefaultVariables: `{"protocol":"vless"}`,
			SubscriptionMeta: `{"type":"vless"}`,
			Description:      "admin-only description",
			Enabled:          &enabled,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			Id:               2,
			Name:             "disabled-template",
			Type:             XrayTemplateInbound,
			Config:           `{"protocol":"trojan","port":443}`,
			DefaultVariables: `{}`,
			SubscriptionMeta: `{}`,
			Enabled:          &disabled,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			Id:               3,
			Name:             "enabled-unused-template",
			Type:             XrayTemplateInbound,
			Config:           `{"protocol":"trojan","port":8443}`,
			DefaultVariables: `{}`,
			SubscriptionMeta: `{}`,
			Enabled:          &enabled,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	}
	if err := db.Create(&templates).Error; err != nil {
		t.Fatal(err)
	}
	bindings := []*ServerXrayTemplate{
		{
			Id:                    10,
			ServerId:              7,
			TemplateId:            1,
			Alias:                 "vless",
			Variables:             `{"port":443}`,
			SubscriptionEnabled:   &enabled,
			SubscriptionName:      "{{ .Server.Name }}",
			SubscriptionVariables: `{"security":"reality"}`,
			Sort:                  2,
			Enabled:               &enabled,
			CreatedAt:             now,
			UpdatedAt:             now,
		},
		{Id: 11, ServerId: 7, TemplateId: 3, Alias: "disabled-binding", Enabled: &disabled},
		{Id: 12, ServerId: 7, TemplateId: 2, Alias: "disabled-template", Enabled: &enabled},
		{Id: 13, ServerId: 8, TemplateId: 1, Alias: "other-server", Enabled: &enabled},
	}
	if err := db.Create(&bindings).Error; err != nil {
		t.Fatal(err)
	}

	list, err := NewModel(db, nil).QueryServerXrayTemplates(ctx, 7, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("runtime bindings = %d, want 1: %#v", len(list), list)
	}
	got := list[0]
	if got.Id != 10 || got.ServerId != 7 || got.TemplateId != 1 || got.Alias != "vless" || got.Sort != 2 || got.Enabled == nil || !*got.Enabled {
		t.Fatalf("runtime binding dropped required fields: %#v", got)
	}
	if got.SubscriptionEnabled == nil || !*got.SubscriptionEnabled || got.SubscriptionName == "" || got.SubscriptionVariables == "" || got.Variables == "" {
		t.Fatalf("runtime binding dropped subscription/render fields: %#v", got)
	}
	if !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("runtime binding loaded audit fields: %#v", got)
	}
	if got.Template == nil {
		t.Fatal("runtime binding did not preload template")
	}
	if got.Template.Id != 1 || got.Template.Name != "vless" || got.Template.Type != XrayTemplateInbound || got.Template.Config == "" || got.Template.ConfigTemplate == "" || got.Template.DefaultVariables == "" || got.Template.SubscriptionMeta == "" {
		t.Fatalf("runtime template dropped required fields: %#v", got.Template)
	}
	if got.Template.Description != "" || got.Template.VariablesSchema != "" || !got.Template.CreatedAt.IsZero() || !got.Template.UpdatedAt.IsZero() {
		t.Fatalf("runtime template loaded admin-only fields: %#v", got.Template)
	}
}

func TestQueryServerXrayTemplatesByServerIDsGroupsRuntimeBindings(t *testing.T) {
	ctx := context.Background()
	db := setupServerXrayTemplateOptionsTestDB(t)

	grouped, err := NewModel(db, nil).QueryServerXrayTemplatesByServerIDs(ctx, []int64{8, 7, 7, 0, 9}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(grouped) != 3 {
		t.Fatalf("grouped server count = %d, want 3: %#v", len(grouped), grouped)
	}
	if got := grouped[7]; len(got) != 2 || got[0].Id != 10 || got[1].Id != 11 {
		t.Fatalf("server 7 bindings = %#v", got)
	}
	if got := grouped[8]; len(got) != 1 || got[0].Id != 12 {
		t.Fatalf("server 8 bindings = %#v", got)
	}
	if got, ok := grouped[9]; !ok || len(got) != 0 {
		t.Fatalf("server 9 empty group = %#v, exists=%v", got, ok)
	}
	got := grouped[7][0]
	if got.Template == nil {
		t.Fatal("runtime batch query did not preload template")
	}
	if !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() || got.Template.Description != "" || got.Template.VariablesSchema != "" {
		t.Fatalf("runtime batch query loaded non-runtime fields: binding=%#v template=%#v", got, got.Template)
	}
}

func TestQueryServerXrayTemplatesByServerIDsSkipsEmptyInputMap(t *testing.T) {
	ctx := context.Background()
	db := setupServerXrayTemplateOptionsTestDB(t)

	grouped, err := NewModel(db, nil).QueryServerXrayTemplatesByServerIDs(ctx, []int64{0, -1}, true)
	if err != nil {
		t.Fatal(err)
	}
	if grouped == nil || len(grouped) != 0 {
		t.Fatalf("empty server id input grouped map = %#v, want non-nil empty map", grouped)
	}
	if got := grouped[7]; got != nil {
		t.Fatalf("empty grouped map lookup = %#v, want nil missing key", got)
	}
}

func TestGroupServerXrayTemplatesByServerIDPreallocatesLargeGroups(t *testing.T) {
	list := []*ServerXrayTemplate{
		{Id: 1, ServerId: 7},
		{Id: 2, ServerId: 8},
		{Id: 3, ServerId: 7},
		nil,
		{Id: 4, ServerId: 7},
		{Id: 5, ServerId: 8},
		{Id: 6, ServerId: 7},
		{Id: 7, ServerId: 8},
		{Id: 8, ServerId: 7},
	}

	grouped := groupServerXrayTemplatesByServerID(list, []int64{7, 8, 9})
	if gotIDs := serverXrayTemplateIDs(grouped[7]); !reflect.DeepEqual(gotIDs, []int64{1, 3, 4, 6, 8}) {
		t.Fatalf("server 7 template ids = %#v", gotIDs)
	}
	if gotIDs := serverXrayTemplateIDs(grouped[8]); !reflect.DeepEqual(gotIDs, []int64{2, 5, 7}) {
		t.Fatalf("server 8 template ids = %#v", gotIDs)
	}
	if got, ok := grouped[9]; !ok || len(got) != 0 {
		t.Fatalf("server 9 empty group = %#v, exists=%v", got, ok)
	}
	if cap(grouped[7]) != 5 || cap(grouped[8]) != 3 {
		t.Fatalf("group caps = 7:%d 8:%d, want exact counts 5 and 3", cap(grouped[7]), cap(grouped[8]))
	}
}

func TestGroupServerXrayTemplatesByServerIDIgnoresUnrequestedServers(t *testing.T) {
	small := groupServerXrayTemplatesByServerID([]*ServerXrayTemplate{
		{Id: 1, ServerId: 7},
		{Id: 2, ServerId: 99},
	}, []int64{7})
	if _, ok := small[99]; ok {
		t.Fatalf("small grouped result included unrequested server: %#v", small)
	}
	if gotIDs := serverXrayTemplateIDs(small[7]); !reflect.DeepEqual(gotIDs, []int64{1}) {
		t.Fatalf("small server 7 template ids = %#v", gotIDs)
	}

	largeList := []*ServerXrayTemplate{
		{Id: 1, ServerId: 7},
		{Id: 2, ServerId: 99},
		{Id: 3, ServerId: 7},
		{Id: 4, ServerId: 99},
		{Id: 5, ServerId: 7},
		{Id: 6, ServerId: 99},
		{Id: 7, ServerId: 7},
		{Id: 8, ServerId: 99},
		{Id: 9, ServerId: 7},
	}
	large := groupServerXrayTemplatesByServerID(largeList, []int64{7})
	if _, ok := large[99]; ok {
		t.Fatalf("large grouped result included unrequested server: %#v", large)
	}
	if gotIDs := serverXrayTemplateIDs(large[7]); !reflect.DeepEqual(gotIDs, []int64{1, 3, 5, 7, 9}) {
		t.Fatalf("large server 7 template ids = %#v", gotIDs)
	}
	if cap(large[7]) != 5 {
		t.Fatalf("large server 7 cap = %d, want exact requested count 5", cap(large[7]))
	}
}

func serverXrayTemplateIDs(list []*ServerXrayTemplate) []int64 {
	ids := make([]int64, 0, len(list))
	for _, item := range list {
		if item != nil {
			ids = append(ids, item.Id)
		}
	}
	return ids
}

func setupServerXrayTemplateOptionsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&XrayTemplate{}, &ServerXrayTemplate{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	enabled := true
	templates := []*XrayTemplate{
		{
			Id:               1,
			Name:             "vless",
			Type:             XrayTemplateInbound,
			Config:           `{"protocol":"vless","port":443}`,
			ConfigTemplate:   `{"protocol":"{{ .Vars.protocol }}","port":443}`,
			VariablesSchema:  `{"protocol":{"type":"string"}}`,
			DefaultVariables: `{"protocol":"vless"}`,
			SubscriptionMeta: `{"type":"vless"}`,
			Description:      "admin-only description",
			Enabled:          &enabled,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			Id:               2,
			Name:             "freedom",
			Type:             XrayTemplateOutbound,
			Config:           `{"protocol":"freedom"}`,
			ConfigTemplate:   `{"protocol":"freedom"}`,
			VariablesSchema:  `{}`,
			DefaultVariables: `{}`,
			SubscriptionMeta: `{}`,
			Description:      "outbound description",
			Enabled:          &enabled,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	}
	if err := db.Create(&templates).Error; err != nil {
		t.Fatal(err)
	}

	bindings := []*ServerXrayTemplate{
		{
			Id:                    10,
			ServerId:              7,
			TemplateId:            1,
			Alias:                 "vless",
			Variables:             `{"port":443}`,
			SubscriptionEnabled:   &enabled,
			SubscriptionName:      "{{ .Server.Name }}",
			SubscriptionVariables: `{"security":"reality"}`,
			Sort:                  1,
			Enabled:               &enabled,
			CreatedAt:             now,
			UpdatedAt:             now,
		},
		{
			Id:                    11,
			ServerId:              7,
			TemplateId:            2,
			Alias:                 "freedom",
			Variables:             `{}`,
			SubscriptionEnabled:   &enabled,
			SubscriptionVariables: `{}`,
			Sort:                  2,
			Enabled:               &enabled,
			CreatedAt:             now,
			UpdatedAt:             now,
		},
		{
			Id:         12,
			ServerId:   8,
			TemplateId: 1,
			Alias:      "other-server",
			Enabled:    &enabled,
		},
	}
	if err := db.Create(&bindings).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestServerListColumnsStayExplicit(t *testing.T) {
	columns := make(map[string]struct{}, len(serverListColumns))
	for _, column := range serverListColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "name", "country", "city", "address", "sort", "last_reported_at", "created_at", "updated_at"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("server list columns must include %q", column)
		}
	}
}

func TestQueryServerListReturnsEmptyWithoutDatabaseForEmptyIDs(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	list, err := NewModel(db, nil).QueryServerList(ctx, []int64{0, -1, 0})
	if err != nil {
		t.Fatal(err)
	}
	if list == nil || len(list) != 0 {
		t.Fatalf("empty query server list = %#v, want non-nil empty slice", list)
	}
}

func TestQueryServerListNormalizesIDsAndPreservesInputOrder(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Server{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create([]*Server{
		{Id: 1, Name: "one", Country: "US", City: "Denver", Address: "10.0.0.1", Sort: 1, CreatedAt: now, UpdatedAt: now},
		{Id: 2, Name: "two", Country: "JP", City: "Tokyo", Address: "10.0.0.2", Sort: 2, CreatedAt: now, UpdatedAt: now},
		{Id: 3, Name: "three", Country: "SG", City: "Singapore", Address: "10.0.0.3", Sort: 3, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}

	list, err := NewModel(db, nil).QueryServerList(ctx, []int64{3, 1, 3, 0, 2, 99})
	if err != nil {
		t.Fatal(err)
	}
	gotIDs := make([]int64, 0, len(list))
	for _, srv := range list {
		gotIDs = append(gotIDs, srv.Id)
	}
	if want := []int64{3, 1, 2}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("server ids = %#v, want %#v", gotIDs, want)
	}
}

func TestQueryServerListSkipsMissingWithoutInputSizedCapacity(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Server{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Server{Id: 7, Name: "edge"}).Error; err != nil {
		t.Fatal(err)
	}
	ids := make([]int64, smallNodeServerIDSetThreshold+20)
	ids[1] = 7
	for index := 2; index < len(ids); index++ {
		ids[index] = int64(100 + index)
	}

	list, err := NewModel(db, nil).QueryServerList(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Id != 7 {
		t.Fatalf("server list = %#v, want server 7 only", list)
	}
	if maxCap := smallNodeServerIDSetThreshold + 1; cap(list) > maxCap {
		t.Fatalf("server list cap = %d, want <= %d", cap(list), maxCap)
	}
}

func TestOrderServersByIDsSmallResultAvoidsMapAllocation(t *testing.T) {
	servers := []*Server{{Id: 3}, {Id: 1}, nil, {Id: 2}}
	got := orderServersByIDs(servers, []int64{1, 99, 3, 2})
	gotIDs := make([]int64, 0, len(got))
	for _, server := range got {
		gotIDs = append(gotIDs, server.Id)
	}
	if want := []int64{1, 3, 2}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("ordered server ids = %#v, want %#v", gotIDs, want)
	}

	allocs := testing.AllocsPerRun(1000, func() {
		_ = orderServersByIDs(servers, []int64{1, 3, 2})
	})
	if allocs > 1 {
		t.Fatalf("orderServersByIDs small result allocations = %f, want at most 1", allocs)
	}
}

func TestNormalizeServerIDsSmallInputAvoidsMapAllocation(t *testing.T) {
	ids := []int64{3, 1, 3, 0, 2, 99}
	got := normalizeServerIDs(ids)
	if want := []int64{3, 1, 2, 99}; !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeServerIDs() = %#v, want %#v", got, want)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = normalizeServerIDs(ids)
	})
	if allocs > 1 {
		t.Fatalf("normalizeServerIDs small input allocations = %f, want at most 1", allocs)
	}
}

func TestNormalizeServerFilterParamsDeduplicatesIDs(t *testing.T) {
	params := &FilterParams{
		Page:    -1,
		Size:    0,
		MaxSize: 50,
		Ids:     []int64{3, 0, 2, 3, -1},
	}

	normalizeServerFilterParams(params)

	if params.Page != 1 || params.Size != 10 {
		t.Fatalf("pagination = page %d size %d, want page 1 size 10", params.Page, params.Size)
	}
	if want := []int64{3, 2}; !reflect.DeepEqual(params.Ids, want) {
		t.Fatalf("server ids = %#v, want %#v", params.Ids, want)
	}
}

func TestNormalizeNodeFilterParamsDeduplicatesSelectors(t *testing.T) {
	params := &FilterNodeParams{
		Page:     -1,
		Size:     0,
		MaxSize:  50,
		NodeId:   []int64{7, 7, 0, 8, -1},
		ServerId: []int64{3, 0, 2, 3, -1},
		Tag:      []string{" edge ", "", "premium", "edge", "  "},
	}

	normalizeNodeFilterParams(params)

	if params.Page != 1 || params.Size != 10 {
		t.Fatalf("pagination = page %d size %d, want page 1 size 10", params.Page, params.Size)
	}
	if want := []int64{7, 8}; !reflect.DeepEqual(params.NodeId, want) {
		t.Fatalf("node ids = %#v, want %#v", params.NodeId, want)
	}
	if want := []int64{3, 2}; !reflect.DeepEqual(params.ServerId, want) {
		t.Fatalf("server ids = %#v, want %#v", params.ServerId, want)
	}
	if want := []string{"edge", "premium"}; !reflect.DeepEqual(params.Tag, want) {
		t.Fatalf("tags = %#v, want %#v", params.Tag, want)
	}
}

func TestNormalizeNodeIDsKeepsSparseInputCompact(t *testing.T) {
	ids := make([]int64, smallNodeServerIDSetThreshold+20)
	ids[2] = 7
	ids[smallNodeServerIDSetThreshold+10] = 7

	got := normalizeNodePositiveIDs(ids)
	if want := []int64{7}; !reflect.DeepEqual(got, want) {
		t.Fatalf("node ids = %#v, want %#v", got, want)
	}
	if maxCap := smallNodeServerIDSetThreshold + 1; cap(got) > maxCap {
		t.Fatalf("node ids capacity = %d, want <= %d", cap(got), maxCap)
	}

	serverIDs := normalizeServerIDs(ids)
	if want := []int64{7}; !reflect.DeepEqual(serverIDs, want) {
		t.Fatalf("server ids = %#v, want %#v", serverIDs, want)
	}
	if maxCap := smallNodeServerIDSetThreshold + 1; cap(serverIDs) > maxCap {
		t.Fatalf("server ids capacity = %d, want <= %d", cap(serverIDs), maxCap)
	}
}

func TestNormalizeNodeIDsBoundsSparseUpgradeCapacity(t *testing.T) {
	ids := make([]int64, smallNodeServerIDSetThreshold+40)
	for i := 0; i <= smallNodeServerIDSetThreshold; i++ {
		ids[i] = int64(i + 1)
	}

	got := normalizeNodePositiveIDs(ids)
	if len(got) != smallNodeServerIDSetThreshold+1 {
		t.Fatalf("node ids len = %d, want %d", len(got), smallNodeServerIDSetThreshold+1)
	}
	if maxCap := (smallNodeServerIDSetThreshold + 1) * 2; cap(got) > maxCap {
		t.Fatalf("node ids capacity = %d, want <= %d", cap(got), maxCap)
	}
}

func TestFilterListsPreallocatePageCapacity(t *testing.T) {
	ctx := context.Background()
	db := newNodeFilterListTestDB(t)
	now := time.Unix(1700000000, 0)
	enabled := true

	if err := db.Create([]*Server{
		{Id: 1, Name: "server-1", Address: "10.0.0.1", Sort: 1, CreatedAt: now, UpdatedAt: now},
		{Id: 2, Name: "server-2", Address: "10.0.0.2", Sort: 2, CreatedAt: now, UpdatedAt: now},
		{Id: 3, Name: "server-3", Address: "10.0.0.3", Sort: 3, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*Node{
		{Id: 1, Name: "node-1", Tags: "edge", Port: 443, Address: "node-1.example.com", ServerId: 1, Protocol: "vless", InboundAlias: "vless", Enabled: &enabled, Sort: 1, CreatedAt: now, UpdatedAt: now},
		{Id: 2, Name: "node-2", Tags: "edge", Port: 8443, Address: "node-2.example.com", ServerId: 2, Protocol: "trojan", InboundAlias: "trojan", Enabled: &enabled, Sort: 2, CreatedAt: now, UpdatedAt: now},
		{Id: 3, Name: "node-3", Tags: "core", Port: 9443, Address: "node-3.example.com", ServerId: 3, Protocol: "hysteria2", InboundAlias: "hysteria2", Enabled: &enabled, Sort: 3, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*XrayTemplate{
		{Id: 1, Name: "inbound", Type: XrayTemplateInbound, Config: `{}`, ConfigTemplate: `{}`, VariablesSchema: `{}`, DefaultVariables: `{}`, SubscriptionMeta: `{}`, Enabled: &enabled, CreatedAt: now, UpdatedAt: now},
		{Id: 2, Name: "outbound", Type: XrayTemplateOutbound, Config: `{}`, ConfigTemplate: `{}`, VariablesSchema: `{}`, DefaultVariables: `{}`, SubscriptionMeta: `{}`, Enabled: &enabled, CreatedAt: now, UpdatedAt: now},
		{Id: 3, Name: "routing", Type: XrayTemplateRouting, Config: `{}`, ConfigTemplate: `{}`, VariablesSchema: `{}`, DefaultVariables: `{}`, SubscriptionMeta: `{}`, Enabled: &enabled, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	serverTotal, servers, err := model.FilterServerList(ctx, &FilterParams{Page: 1, Size: 2})
	if err != nil {
		t.Fatal(err)
	}
	if serverTotal != 3 || len(servers) != 2 || cap(servers) != 2 {
		t.Fatalf("servers total/len/cap = %d/%d/%d, want 3/2/2", serverTotal, len(servers), cap(servers))
	}

	nodeTotal, nodes, err := model.FilterNodeList(ctx, &FilterNodeParams{Page: 1, Size: 2})
	if err != nil {
		t.Fatal(err)
	}
	if nodeTotal != 3 || len(nodes) != 2 || cap(nodes) != 2 {
		t.Fatalf("nodes total/len/cap = %d/%d/%d, want 3/2/2", nodeTotal, len(nodes), cap(nodes))
	}

	templateTotal, templates, err := model.FilterXrayTemplateList(ctx, &FilterXrayTemplateParams{Page: 1, Size: 2})
	if err != nil {
		t.Fatal(err)
	}
	if templateTotal != 3 || len(templates) != 2 || cap(templates) != 2 {
		t.Fatalf("templates total/len/cap = %d/%d/%d, want 3/2/2", templateTotal, len(templates), cap(templates))
	}
}

func TestNodeListMethodsReturnNonNilEmptySlices(t *testing.T) {
	ctx := context.Background()
	db := newNodeFilterListTestDB(t)

	model := NewModel(db, nil)
	serverTotal, servers, err := model.FilterServerList(ctx, &FilterParams{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if serverTotal != 0 || servers == nil || len(servers) != 0 {
		t.Fatalf("server list = total %d list %#v, want total 0 and non-nil empty slice", serverTotal, servers)
	}

	nodeTotal, nodes, err := model.FilterNodeList(ctx, &FilterNodeParams{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if nodeTotal != 0 || nodes == nil || len(nodes) != 0 {
		t.Fatalf("node list = total %d list %#v, want total 0 and non-nil empty slice", nodeTotal, nodes)
	}

	templateTotal, templates, err := model.FilterXrayTemplateList(ctx, &FilterXrayTemplateParams{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if templateTotal != 0 || templates == nil || len(templates) != 0 {
		t.Fatalf("xray template list = total %d list %#v, want total 0 and non-nil empty slice", templateTotal, templates)
	}

	ips, err := model.QueryServerIPList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ips == nil || len(ips) != 0 {
		t.Fatalf("server ip list = %#v, want non-nil empty slice", ips)
	}

	deployServers, err := model.QueryDeployServerList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if deployServers == nil || len(deployServers) != 0 {
		t.Fatalf("deploy server list = %#v, want non-nil empty slice", deployServers)
	}

	serverTemplates, err := model.QueryServerXrayTemplatesWithOptions(ctx, 7, QueryServerXrayTemplateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if serverTemplates == nil || len(serverTemplates) != 0 {
		t.Fatalf("server xray templates = %#v, want non-nil empty slice", serverTemplates)
	}

	grouped, err := model.QueryServerXrayTemplatesByServerIDs(ctx, []int64{7}, false)
	if err != nil {
		t.Fatal(err)
	}
	if grouped == nil || grouped[7] == nil || len(grouped[7]) != 0 {
		t.Fatalf("grouped server xray templates = %#v, want key 7 with non-nil empty slice", grouped)
	}

	emptyGrouped, err := model.QueryServerXrayTemplatesByServerIDs(ctx, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if emptyGrouped == nil || len(emptyGrouped) != 0 {
		t.Fatalf("empty grouped server xray templates = %#v, want non-nil empty map", emptyGrouped)
	}
}

func TestQueryXrayTemplateCatalogLoadsConfigWithoutTimestamps(t *testing.T) {
	ctx := context.Background()
	db := newNodeFilterListTestDB(t)
	enabled := true
	now := time.Unix(1700000000, 0)
	if err := db.Create(&XrayTemplate{
		Id:               7,
		Name:             "inbound",
		Type:             XrayTemplateInbound,
		Config:           `{"protocol":"vless"}`,
		ConfigTemplate:   `{"tag":"{{ .vars.tag }}"}`,
		VariablesSchema:  `{"properties":{"tag":{"type":"string"}}}`,
		DefaultVariables: `{"tag":"proxy"}`,
		SubscriptionMeta: `{"host":"example.com"}`,
		Description:      "template",
		Enabled:          &enabled,
		CreatedAt:        now,
		UpdatedAt:        now.Add(time.Minute),
	}).Error; err != nil {
		t.Fatal(err)
	}

	total, list, err := NewModel(db, nil).QueryXrayTemplateCatalog(ctx, &FilterXrayTemplateParams{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("catalog = total %d len %d, want 1/1", total, len(list))
	}
	got := list[0]
	if got.Config == "" || got.ConfigTemplate == "" || got.VariablesSchema == "" || got.DefaultVariables == "" || got.SubscriptionMeta == "" {
		t.Fatalf("catalog did not load config fields: %#v", got)
	}
	if !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("catalog loaded timestamp fields: created=%v updated=%v", got.CreatedAt, got.UpdatedAt)
	}
}

func TestFilterXrayTemplateListIncludesBindingSummaries(t *testing.T) {
	ctx := context.Background()
	db := newNodeFilterListTestDB(t)
	enabled := true
	disabled := false
	now := time.Unix(1700000000, 0)
	if err := db.Create(&XrayTemplate{
		Id:               7,
		Name:             "inbound",
		Type:             XrayTemplateInbound,
		Config:           `{}`,
		ConfigTemplate:   `{}`,
		VariablesSchema:  `{}`,
		DefaultVariables: `{}`,
		SubscriptionMeta: `{}`,
		Enabled:          &enabled,
		CreatedAt:        now,
		UpdatedAt:        now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*ServerXrayTemplate{
		{ServerId: 1, TemplateId: 7, Enabled: &enabled, SubscriptionEnabled: &enabled},
		{ServerId: 2, TemplateId: 7, Enabled: &enabled, SubscriptionEnabled: &disabled},
		{ServerId: 3, TemplateId: 7, Enabled: &disabled, SubscriptionEnabled: &enabled},
	}).Error; err != nil {
		t.Fatal(err)
	}

	total, list, err := NewModel(db, nil).FilterXrayTemplateList(ctx, &FilterXrayTemplateParams{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("template list = total %d len %d, want 1/1", total, len(list))
	}
	if list[0].BindingCount != 2 || list[0].SubscriptionBindingCount != 1 {
		t.Fatalf("binding summary = %d/%d, want 2/1", list[0].BindingCount, list[0].SubscriptionBindingCount)
	}
}

func TestServerDeployListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(serverDeployListColumns))
	for _, column := range serverDeployListColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "name", "address"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("server deploy list columns must include %q", column)
		}
	}

	forbidden := []string{"country", "city", "sort", "last_reported_at", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("server deploy list columns must not include %q", column)
		}
	}
}

func TestServerRuntimeColumnsStayLightweight(t *testing.T) {
	want := []string{"id", "name", "address"}
	if !reflect.DeepEqual(serverRuntimeColumns, want) {
		t.Fatalf("server runtime columns = %#v, want %#v", serverRuntimeColumns, want)
	}
}

func TestFindOneServerForRuntimeSelectsOnlyRuntimeColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Server{}); err != nil {
		t.Fatal(err)
	}

	reportedAt := time.Unix(1700000000, 0)
	if err := db.Create(&Server{
		Id:             7,
		Name:           "edge",
		Country:        "US",
		City:           "Denver",
		Address:        "edge.example.com",
		Sort:           3,
		LastReportedAt: &reportedAt,
		CreatedAt:      reportedAt,
		UpdatedAt:      reportedAt,
	}).Error; err != nil {
		t.Fatal(err)
	}

	srv, err := NewModel(db, nil).FindOneServerForRuntime(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if srv.Id != 7 || srv.Name != "edge" || srv.Address != "edge.example.com" {
		t.Fatalf("runtime server query dropped required fields: %#v", srv)
	}
	if srv.Country != "" || srv.City != "" || srv.Sort != 0 || srv.LastReportedAt != nil || !srv.CreatedAt.IsZero() || !srv.UpdatedAt.IsZero() {
		t.Fatalf("runtime server query loaded non-runtime fields: %#v", srv)
	}
}

func TestQueryDeployServerListSelectsOnlyResponseColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Server{}); err != nil {
		t.Fatal(err)
	}

	reportedAt := time.Unix(1700000000, 0)
	if err := db.Create([]*Server{
		{Id: 1, Name: "later", Country: "US", City: "Denver", Address: "10.0.0.2", Sort: 2, LastReportedAt: &reportedAt, CreatedAt: reportedAt, UpdatedAt: reportedAt},
		{Id: 2, Name: "empty", Country: "US", City: "Austin", Address: "", Sort: 1, LastReportedAt: &reportedAt, CreatedAt: reportedAt, UpdatedAt: reportedAt},
		{Id: 3, Name: "first", Country: "JP", City: "Tokyo", Address: "10.0.0.1", Sort: 1, LastReportedAt: &reportedAt, CreatedAt: reportedAt, UpdatedAt: reportedAt},
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	list, err := model.QueryDeployServerList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("deploy server list = %d, want 2", len(list))
	}
	gotIDs := []int64{list[0].Id, list[1].Id}
	if want := []int64{3, 1}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("deploy server ids = %#v, want %#v", gotIDs, want)
	}
	for _, srv := range list {
		if srv.Name == "" || srv.Address == "" {
			t.Fatalf("deploy server response fields missing: %#v", srv)
		}
		if srv.Country != "" || srv.City != "" || srv.Sort != 0 || srv.LastReportedAt != nil || !srv.CreatedAt.IsZero() || !srv.UpdatedAt.IsZero() {
			t.Fatalf("deploy server query loaded non-response fields: %#v", srv)
		}
	}
}

func TestReorderSortWithServerUsesSortableRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Server{}); err != nil {
		t.Fatal(err)
	}

	reportedAt := time.Unix(1700000000, 0)
	if err := db.Create([]*Server{
		{Id: 1, Name: "one", Country: "US", City: "Denver", Address: "10.0.0.1", Sort: 2, LastReportedAt: &reportedAt, CreatedAt: reportedAt, UpdatedAt: reportedAt},
		{Id: 2, Name: "two", Country: "JP", City: "Tokyo", Address: "10.0.0.2", Sort: 2, LastReportedAt: &reportedAt, CreatedAt: reportedAt, UpdatedAt: reportedAt},
		{Id: 3, Name: "three", Country: "SG", City: "Singapore", Address: "10.0.0.3", Sort: 5, LastReportedAt: &reportedAt, CreatedAt: reportedAt, UpdatedAt: reportedAt},
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := reorderSortWithServer(db); err != nil {
		t.Fatal(err)
	}

	var rows []sortRow
	if err := db.Model(&Server{}).Select("id", "sort").Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	want := []sortRow{{Id: 1, Sort: 1}, {Id: 2, Sort: 2}, {Id: 3, Sort: 3}}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("server sort rows = %#v, want %#v", rows, want)
	}
}

func TestNodeListColumnsStayExplicit(t *testing.T) {
	columns := make(map[string]struct{}, len(nodeListColumns))
	for _, column := range nodeListColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "name", "tags", "port", "address", "server_id", "protocol", "inbound_alias", "enabled", "sort", "created_at", "updated_at"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("node list columns must include %q", column)
		}
	}
}

func TestReorderSortWithNodeUsesSortableRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Node{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	enabled := true
	if err := db.Create([]*Node{
		{Id: 1, Name: "one", Tags: "a", Port: 443, Address: "node-1.example.com", ServerId: 10, Protocol: "vless", InboundAlias: "vless", Enabled: &enabled, Sort: 2, CreatedAt: now, UpdatedAt: now},
		{Id: 2, Name: "two", Tags: "b", Port: 8443, Address: "node-2.example.com", ServerId: 10, Protocol: "trojan", InboundAlias: "trojan", Enabled: &enabled, Sort: 4, CreatedAt: now, UpdatedAt: now},
		{Id: 3, Name: "three", Tags: "c", Port: 9443, Address: "node-3.example.com", ServerId: 11, Protocol: "hysteria", InboundAlias: "hysteria", Enabled: &enabled, Sort: 9, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := reorderSortWithNode(db); err != nil {
		t.Fatal(err)
	}

	var rows []sortRow
	if err := db.Model(&Node{}).Select("id", "sort").Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	want := []sortRow{{Id: 1, Sort: 1}, {Id: 2, Sort: 2}, {Id: 3, Sort: 3}}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("node sort rows = %#v, want %#v", rows, want)
	}
}

func TestUpdateServerUsesSingleTargetedStatement(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Server{}); err != nil {
		t.Fatal(err)
	}
	createdAt := time.Unix(1_700_000_000, 0).UTC()
	reportedAt := createdAt.Add(time.Hour)
	if err := db.Create(&Server{
		Id:             1,
		Name:           "old",
		Country:        "US",
		City:           "Denver",
		Address:        "old.example.com",
		Sort:           7,
		LastReportedAt: &reportedAt,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}).Error; err != nil {
		t.Fatal(err)
	}

	queries := 0
	updates := 0
	if err := db.Callback().Query().Before("gorm:query").Register("test:update_server_query_count", func(*gorm.DB) {
		queries++
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Update().Before("gorm:update").Register("test:update_server_update_count", func(*gorm.DB) {
		updates++
	}); err != nil {
		t.Fatal(err)
	}

	model := newServerModel(db, nil)
	input := &Server{Id: 1, Name: "new", Country: "JP", City: "Tokyo", Address: "new.example.com", Sort: 99}
	if err := model.UpdateServer(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if queries != 0 || updates != 1 {
		t.Fatalf("UpdateServer callbacks = queries:%d updates:%d, want 0/1", queries, updates)
	}

	var got Server
	if err := db.First(&got, 1).Error; err != nil {
		t.Fatal(err)
	}
	if got.Name != "new" || got.Country != "JP" || got.City != "Tokyo" || got.Address != "new.example.com" {
		t.Fatalf("mutable server fields were not updated: %#v", got)
	}
	if got.Sort != 7 || got.LastReportedAt == nil || !got.LastReportedAt.Equal(reportedAt) || !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("UpdateServer overwrote immutable/runtime fields: %#v", got)
	}
	if !got.UpdatedAt.Equal(input.UpdatedAt) || !got.UpdatedAt.After(createdAt) {
		t.Fatalf("updated_at = %v input=%v, want a new shared timestamp", got.UpdatedAt, input.UpdatedAt)
	}
	if err := model.UpdateServer(context.Background(), &Server{Id: 99}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("UpdateServer missing row error = %v, want gorm.ErrRecordNotFound", err)
	}
}

func TestUpdateNodeUsesSingleTargetedStatement(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Node{}); err != nil {
		t.Fatal(err)
	}
	createdAt := time.Unix(1_700_000_000, 0).UTC()
	enabled := true
	if err := db.Create(&Node{
		Id: 1, Name: "old", Tags: "old", Port: 443, Address: "old.example.com", ServerId: 2,
		Protocol: "vless", InboundAlias: "old", Enabled: &enabled, Sort: 7, CreatedAt: createdAt, UpdatedAt: createdAt,
	}).Error; err != nil {
		t.Fatal(err)
	}

	queries := 0
	updates := 0
	if err := db.Callback().Query().Before("gorm:query").Register("test:update_node_query_count", func(*gorm.DB) {
		queries++
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Update().Before("gorm:update").Register("test:update_node_update_count", func(*gorm.DB) {
		updates++
	}); err != nil {
		t.Fatal(err)
	}

	disabled := false
	model := newServerModel(db, nil)
	input := &Node{
		Id: 1, Name: "new", Tags: "edge", Port: 8443, Address: "new.example.com", ServerId: 3,
		Protocol: "trojan", InboundAlias: "main", Enabled: &disabled, Sort: 99,
	}
	if err := model.UpdateNode(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if queries != 0 || updates != 1 {
		t.Fatalf("UpdateNode callbacks = queries:%d updates:%d, want 0/1", queries, updates)
	}

	var got Node
	if err := db.First(&got, 1).Error; err != nil {
		t.Fatal(err)
	}
	if got.Name != "new" || got.Tags != "edge" || got.Port != 8443 || got.Address != "new.example.com" || got.ServerId != 3 || got.Protocol != "trojan" || got.InboundAlias != "main" || got.Enabled == nil || *got.Enabled {
		t.Fatalf("mutable node fields were not updated: %#v", got)
	}
	if got.Sort != 7 || !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("UpdateNode overwrote immutable fields: %#v", got)
	}
	if !got.UpdatedAt.Equal(input.UpdatedAt) || !got.UpdatedAt.After(createdAt) {
		t.Fatalf("updated_at = %v input=%v, want a new shared timestamp", got.UpdatedAt, input.UpdatedAt)
	}
	if err := model.UpdateNode(context.Background(), &Node{Id: 99}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("UpdateNode missing row error = %v, want gorm.ErrRecordNotFound", err)
	}
}

func TestNodeCacheInvalidationColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(nodeCacheInvalidationColumns))
	for _, column := range nodeCacheInvalidationColumns {
		columns[column] = struct{}{}
	}

	required := []string{"server_id", "protocol", "inbound_alias"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("node cache invalidation columns must include %q", column)
		}
	}

	forbidden := []string{"id", "name", "tags", "port", "address", "enabled", "sort", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("node cache invalidation columns must not include %q", column)
		}
	}
}

func TestBuildNodeCacheInvalidationPlanDeduplicatesServerScans(t *testing.T) {
	plan := buildNodeCacheInvalidationPlan([]*Node{
		nil,
		{ServerId: 10, Protocol: "vless"},
		{ServerId: 10, Protocol: "trojan"},
		{ServerId: 11},
		{ServerId: 12, InboundAlias: "hysteria2"},
		{ServerId: 12, Protocol: "hysteria2"},
		{ServerId: 0, Protocol: "ignored"},
	})

	if want := []int64{10, 11, 12}; !reflect.DeepEqual(plan.userListServerIDs, want) {
		t.Fatalf("userListServerIDs = %#v, want %#v", plan.userListServerIDs, want)
	}
	if want := []int64{10, 12}; !reflect.DeepEqual(plan.configServerIDs, want) {
		t.Fatalf("configServerIDs = %#v, want %#v", plan.configServerIDs, want)
	}
}

func TestBuildNodeCacheInvalidationPlanSmallInputAvoidsMapAllocation(t *testing.T) {
	nodes := []*Node{
		{ServerId: 10, Protocol: "vless"},
		{ServerId: 10, Protocol: "trojan"},
		{ServerId: 11},
		{ServerId: 12, InboundAlias: "hysteria2"},
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = buildNodeCacheInvalidationPlan(nodes)
	})
	if allocs > 2 {
		t.Fatalf("buildNodeCacheInvalidationPlan small input allocations = %f, want at most 2", allocs)
	}
}

func TestBuildNodeCacheInvalidationPlanAvoidsInputSizedCapacityForLargeDuplicateInput(t *testing.T) {
	nodes := make([]*Node, smallNodeServerIDSetThreshold+20)
	nodes[1] = &Node{ServerId: 10}
	nodes[2] = &Node{ServerId: 10, Protocol: "vless"}
	nodes[smallNodeServerIDSetThreshold+10] = &Node{ServerId: 10, Protocol: "trojan"}

	plan := buildNodeCacheInvalidationPlan(nodes)
	if want := []int64{10}; !reflect.DeepEqual(plan.userListServerIDs, want) {
		t.Fatalf("user list server ids = %#v, want %#v", plan.userListServerIDs, want)
	}
	if want := []int64{10}; !reflect.DeepEqual(plan.configServerIDs, want) {
		t.Fatalf("config server ids = %#v, want %#v", plan.configServerIDs, want)
	}
	if cap(plan.userListServerIDs) != smallNodeServerIDSetThreshold+1 {
		t.Fatalf("user list id cap = %d, want small initial cap", cap(plan.userListServerIDs))
	}
	if cap(plan.configServerIDs) != smallNodeServerIDSetThreshold+1 {
		t.Fatalf("config id cap = %d, want small initial cap", cap(plan.configServerIDs))
	}
}

func TestAppendUniqueNodeServerIDUsesCapacityHintForLargeInput(t *testing.T) {
	ids := make([]int64, 0, smallNodeServerIDSetThreshold+1)
	var seen map[int64]struct{}
	ids, seen = appendUniqueNodeServerID(ids, 10, seen)
	if seen != nil {
		t.Fatal("large hinted node server id set should not allocate map for one unique id")
	}
	ids, seen = appendUniqueNodeServerID(ids, 10, seen)
	if len(ids) != 1 || seen != nil {
		t.Fatalf("node server ids = %#v seen=%#v, want one unique id and no map", ids, seen)
	}
	for id := int64(11); id <= int64(10+smallNodeServerIDSetThreshold); id++ {
		ids, seen = appendUniqueNodeServerID(ids, id, seen)
	}
	if seen == nil {
		t.Fatal("node server id set should allocate map after actual unique ids exceed threshold")
	}
	if len(ids) != smallNodeServerIDSetThreshold+1 {
		t.Fatalf("node server ids = %#v, want %d unique ids", ids, smallNodeServerIDSetThreshold+1)
	}
}

func TestUniqueNodeCacheKeysKeepsOrderAndSkipsEmpty(t *testing.T) {
	got := uniqueNodeCacheKeys([]string{"server:user:1", "", "server:config:1:a", "server:user:1", "server:config:1:a", "server:user:2"})
	want := []string{"server:user:1", "server:config:1:a", "server:user:2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueNodeCacheKeys() = %#v, want %#v", got, want)
	}
}

func TestUniqueNodeCacheKeysSmallInputAvoidsMapAllocation(t *testing.T) {
	keys := []string{"server:user:1", "server:config:1:a", "server:user:1"}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = uniqueNodeCacheKeys(keys)
	})
	if allocs > 1 {
		t.Fatalf("uniqueNodeCacheKeys small input allocations = %f, want at most 1", allocs)
	}
}

func TestUniqueNodeCacheKeysLargeDuplicateAvoidsInputSizedCapacity(t *testing.T) {
	keys := make([]string, smallNodeServerIDSetThreshold+20)
	for i := range keys {
		keys[i] = "server:user:1"
	}

	got := uniqueNodeCacheKeys(keys)
	if want := []string{"server:user:1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unique node cache keys = %#v, want %#v", got, want)
	}
	if cap(got) != smallNodeServerIDSetThreshold+1 {
		t.Fatalf("unique node cache key cap = %d, want small initial cap", cap(got))
	}
}

func TestUniqueNodeCacheKeysBoundsSparseUpgradeCapacity(t *testing.T) {
	keys := make([]string, smallNodeServerIDSetThreshold+40)
	for i := 0; i <= smallNodeServerIDSetThreshold; i++ {
		keys[i] = "server:user:" + strconv.Itoa(i+1)
	}
	for i := smallNodeServerIDSetThreshold + 1; i < len(keys); i++ {
		keys[i] = "server:user:1"
	}

	got := uniqueNodeCacheKeys(keys)
	if len(got) != smallNodeServerIDSetThreshold+1 {
		t.Fatalf("unique node cache key len = %d, want %d", len(got), smallNodeServerIDSetThreshold+1)
	}
	if maxCap := (smallNodeServerIDSetThreshold + 1) * 2; cap(got) > maxCap {
		t.Fatalf("unique node cache key cap = %d, want <= %d", cap(got), maxCap)
	}
}

func TestClearServerCacheSkipsInvalidServerID(t *testing.T) {
	if err := (&customServerModel{}).ClearServerCache(context.Background(), 0); err != nil {
		t.Fatalf("ClearServerCache(0) error = %v", err)
	}
	if err := (&customServerModel{}).ClearServerCache(context.Background(), -1); err != nil {
		t.Fatalf("ClearServerCache(-1) error = %v", err)
	}
}

func newNodeFilterListTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Server{}, &Node{}, &XrayTemplate{}, &ServerXrayTemplate{}); err != nil {
		t.Fatal(err)
	}
	return db
}
