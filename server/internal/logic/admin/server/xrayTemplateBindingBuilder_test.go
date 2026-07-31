package server

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestXrayTemplateBindingColumnsStayComplete(t *testing.T) {
	columns := make(map[string]struct{}, len(xrayTemplateBindingColumns))
	for _, column := range xrayTemplateBindingColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"name",
		"type",
		"config",
		"config_template",
		"variables_schema",
		"default_variables",
		"subscription_meta",
		"description",
		"enabled",
		"created_at",
		"updated_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("xray template binding columns must include %q", column)
		}
	}
}

func TestCollectXrayTemplateIDsDeduplicatesAndValidates(t *testing.T) {
	got, err := collectXrayTemplateIDs([]types.ServerXrayTemplateBinding{
		{TemplateId: 2},
		{TemplateId: 1},
		{TemplateId: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []int64{2, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("collectXrayTemplateIDs() = %#v, want %#v", got, want)
	}

	if _, err = collectXrayTemplateIDs([]types.ServerXrayTemplateBinding{{TemplateId: 0}}); err == nil {
		t.Fatal("collectXrayTemplateIDs() expected error for missing template id")
	}
}

func TestCollectXrayTemplateIDsKeepsEmptySliceShape(t *testing.T) {
	got, err := collectXrayTemplateIDs(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("empty template ids = %#v, want non-nil empty slice", got)
	}
}

func TestXrayTemplateIDSetUsesLinearPathForSmallInput(t *testing.T) {
	var set xrayTemplateIDSet
	if !set.Add(2) || !set.Add(1) || set.Add(2) || set.Add(0) {
		t.Fatalf("xray template id set add results invalid: %#v", set)
	}
	if set.index != nil {
		t.Fatal("small xray template id set should not allocate map")
	}
	if want := []int64{2, 1}; !reflect.DeepEqual(set.Values(), want) {
		t.Fatalf("xray template id set values = %#v, want %#v", set.Values(), want)
	}
}

func TestXrayTemplateIDSetUpgradesLargeInputToMap(t *testing.T) {
	var set xrayTemplateIDSet
	for i := int64(1); i <= smallXrayTemplateBindingSetThreshold+1; i++ {
		set.Add(i)
	}
	if set.index == nil {
		t.Fatal("large xray template id set should allocate map")
	}
	if len(set.Values()) != smallXrayTemplateBindingSetThreshold+1 {
		t.Fatalf("xray template id set values len = %d, want %d", len(set.Values()), smallXrayTemplateBindingSetThreshold+1)
	}
}

func TestXrayTemplateAliasSetUsesLinearPathForSmallInput(t *testing.T) {
	var set xrayTemplateAliasSet
	if !set.Add("inbound.main") || !set.Add("outbound.main") || set.Add("inbound.main") || set.Add("") {
		t.Fatalf("xray template alias set add results invalid: %#v", set)
	}
	if set.index != nil {
		t.Fatal("small xray template alias set should not allocate map")
	}
}

func TestXrayTemplateAliasSetUpgradesLargeInputToMap(t *testing.T) {
	var set xrayTemplateAliasSet
	for i := 0; i <= smallXrayTemplateBindingSetThreshold; i++ {
		set.Add(string(rune('a' + i)))
	}
	if set.index == nil {
		t.Fatal("large xray template alias set should allocate map")
	}
	if set.Add("a") {
		t.Fatal("large xray template alias set should reject duplicate alias")
	}
}

func TestBuildServerXrayTemplateBindingsUsesLoadedTemplates(t *testing.T) {
	ctx := context.Background()
	db := setupXrayTemplateBindingTestDB(t)

	enabled := true
	disabled := false
	subscriptionDisabled := false
	bindings, enabledBindings, err := buildServerXrayTemplateBindings(ctx, db, 42, []types.ServerXrayTemplateBinding{
		{
			TemplateId:          1,
			Variables:           json.RawMessage(`{"port":443}`),
			SubscriptionEnabled: &subscriptionDisabled,
		},
		{
			TemplateId: 1,
			Alias:      "backup",
			Enabled:    &disabled,
		},
		{
			TemplateId: 2,
			Alias:      "direct",
			Enabled:    &enabled,
			Sort:       9,
		},
	}, xrayBindingBuildOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(bindings) != 3 {
		t.Fatalf("bindings length = %d, want 3", len(bindings))
	}
	if len(enabledBindings) != 2 {
		t.Fatalf("enabled bindings length = %d, want 2", len(enabledBindings))
	}
	if bindings[0].ServerId != 42 || bindings[0].Alias != "vless" || bindings[0].Sort != 1 {
		t.Fatalf("first binding = %#v", bindings[0])
	}
	if bindings[0].Template != bindings[1].Template {
		t.Fatal("duplicate template id should reuse the loaded template object")
	}
	if *bindings[0].SubscriptionEnabled {
		t.Fatal("subscription enabled should preserve explicit false")
	}
	if bindings[0].Variables != `{"port":443}` {
		t.Fatalf("variables = %q", bindings[0].Variables)
	}
	if bindings[2].Sort != 9 || bindings[2].Template.Type != node.XrayTemplateOutbound {
		t.Fatalf("third binding = %#v", bindings[2])
	}
}

func TestBuildServerXrayTemplateBindingsSkipDisabledPreviewBehavior(t *testing.T) {
	ctx := context.Background()
	db := setupXrayTemplateBindingTestDB(t)
	disabled := false

	_, _, err := buildServerXrayTemplateBindings(ctx, db, 42, []types.ServerXrayTemplateBinding{
		{TemplateId: 1, Alias: "same", Enabled: &disabled},
		{TemplateId: 1, Alias: "same"},
	}, xrayBindingBuildOptions{})
	if err == nil {
		t.Fatal("buildServerXrayTemplateBindings() expected duplicate alias error when disabled bindings are persisted")
	}

	bindings, enabledBindings, err := buildServerXrayTemplateBindings(ctx, db, 42, []types.ServerXrayTemplateBinding{
		{TemplateId: 1, Alias: "same", Enabled: &disabled},
		{TemplateId: 1, Alias: "same"},
	}, xrayBindingBuildOptions{skipDisabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 1 || len(enabledBindings) != 1 {
		t.Fatalf("preview bindings lengths = (%d, %d), want (1, 1)", len(bindings), len(enabledBindings))
	}
}

func TestBuildServerXrayTemplateBindingsSkipDisabledWithoutInputSizedCapacity(t *testing.T) {
	ctx := context.Background()
	db := setupXrayTemplateBindingTestDB(t)
	disabled := false
	req := make([]types.ServerXrayTemplateBinding, viewutil.SmallRowsCapacityThreshold+20)
	for index := range req {
		req[index] = types.ServerXrayTemplateBinding{
			TemplateId: 1,
			Alias:      fmt.Sprintf("disabled-%d", index),
			Enabled:    &disabled,
		}
	}
	req[0] = types.ServerXrayTemplateBinding{TemplateId: 1, Alias: "enabled"}

	bindings, enabledBindings, err := buildServerXrayTemplateBindings(ctx, db, 42, req, xrayBindingBuildOptions{skipDisabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 1 || len(enabledBindings) != 1 {
		t.Fatalf("preview bindings lengths = (%d, %d), want (1, 1)", len(bindings), len(enabledBindings))
	}
	if cap(bindings) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("preview bindings cap = %d, want %d", cap(bindings), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(enabledBindings) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("preview enabled bindings cap = %d, want %d", cap(enabledBindings), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestBuildServerXrayTemplateBindingsEnabledRowsSkipDisabledWithoutInputSizedCapacity(t *testing.T) {
	ctx := context.Background()
	db := setupXrayTemplateBindingTestDB(t)
	disabled := false
	req := make([]types.ServerXrayTemplateBinding, viewutil.SmallRowsCapacityThreshold+20)
	req[0] = types.ServerXrayTemplateBinding{TemplateId: 1, Alias: "enabled"}
	for index := 1; index < len(req); index++ {
		req[index] = types.ServerXrayTemplateBinding{
			TemplateId: 1,
			Alias:      fmt.Sprintf("disabled-%d", index),
			Enabled:    &disabled,
		}
	}

	bindings, enabledBindings, err := buildServerXrayTemplateBindings(ctx, db, 42, req, xrayBindingBuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(bindings) != len(req) {
		t.Fatalf("bindings length = %d, want %d", len(bindings), len(req))
	}
	if cap(bindings) != len(req) {
		t.Fatalf("bindings cap = %d, want %d", cap(bindings), len(req))
	}
	if len(enabledBindings) != 1 {
		t.Fatalf("enabled bindings length = %d, want 1", len(enabledBindings))
	}
	if cap(enabledBindings) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("enabled bindings cap = %d, want %d", cap(enabledBindings), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestQueryXrayTemplatesByIDRequiresAllTemplates(t *testing.T) {
	_, err := queryXrayTemplatesByID(context.Background(), setupXrayTemplateBindingTestDB(t), []int64{1, 99})
	if err == nil {
		t.Fatal("queryXrayTemplatesByID() expected missing template error")
	}
}

func TestQueryXrayTemplatesByIDSkipsEmptyLookup(t *testing.T) {
	got, err := queryXrayTemplatesByID(context.Background(), setupXrayTemplateBindingTestDB(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("empty template lookup = %#v, want nil map", got)
	}
}

func setupXrayTemplateBindingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&node.XrayTemplate{}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	templates := []*node.XrayTemplate{
		{
			Id:               1,
			Name:             "vless",
			Type:             node.XrayTemplateInbound,
			Config:           `{"listen":"0.0.0.0"}`,
			ConfigTemplate:   `{"listen":"{{ .Server.Address }}"}`,
			VariablesSchema:  `{}`,
			DefaultVariables: `{}`,
			SubscriptionMeta: `{}`,
			Description:      "inbound",
			Enabled:          &enabled,
		},
		{
			Id:               2,
			Name:             "freedom",
			Type:             node.XrayTemplateOutbound,
			Config:           `{"protocol":"freedom"}`,
			ConfigTemplate:   `{"protocol":"freedom"}`,
			VariablesSchema:  `{}`,
			DefaultVariables: `{}`,
			SubscriptionMeta: `{}`,
			Description:      "outbound",
			Enabled:          &enabled,
		},
	}
	if err := db.Create(&templates).Error; err != nil {
		t.Fatal(err)
	}
	return db
}
