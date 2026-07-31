package application

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/client"
	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/xraytemplate"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetSubscribeApplicationReturnsTemplate(t *testing.T) {
	ctx := context.Background()
	db, model := newSubscribeApplicationTestModel(t)
	row := seedSubscribeApplicationLogicRow(t, db)

	logic := NewGetSubscribeApplicationLogic(ctx, &svc.ServiceContext{ClientModel: model})
	got, err := logic.GetSubscribeApplication(&types.GetSubscribeApplicationRequest{Id: row.Id})
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != row.Id || got.SubscribeTemplate != row.SubscribeTemplate || got.OutputFormat != row.OutputFormat {
		t.Fatalf("detail response missing full template fields: %#v", got)
	}
	if got.DownloadLink.Android == "" {
		t.Fatalf("detail response missing download link: %#v", got.DownloadLink)
	}
}

func TestUpdateSubscribeApplicationPartialPatchPreservesTemplate(t *testing.T) {
	ctx := context.Background()
	db, model := newSubscribeApplicationTestModel(t)
	row := seedSubscribeApplicationLogicRow(t, db)
	if err := db.Model(&client.SubscribeApplication{}).Where("id = ?", row.Id).Update("is_default", false).Error; err != nil {
		t.Fatal(err)
	}
	row.IsDefault = false
	enabled := false

	logic := NewUpdateSubscribeApplicationLogic(ctx, &svc.ServiceContext{ClientModel: model})
	got, err := logic.UpdateSubscribeApplication(&types.UpdateSubscribeApplicationRequest{
		Id:      row.Id,
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Enabled {
		t.Fatalf("partial update response did not include updated enabled flag: %#v", got)
	}
	if got.SubscribeTemplate != "" {
		t.Fatalf("partial update response should omit template payload: %#v", got)
	}

	var saved client.SubscribeApplication
	if err := db.First(&saved, row.Id).Error; err != nil {
		t.Fatal(err)
	}
	if saved.SubscribeTemplate != row.SubscribeTemplate {
		t.Fatalf("partial update cleared template, got %q want %q", saved.SubscribeTemplate, row.SubscribeTemplate)
	}
	if saved.Name != row.Name || saved.UserAgent != row.UserAgent || saved.OutputFormat != row.OutputFormat || saved.DownloadLink != row.DownloadLink {
		t.Fatalf("partial update modified unrelated fields: %#v", saved)
	}
	if saved.Enabled {
		t.Fatalf("partial update did not persist enabled=false: %#v", saved)
	}
}

func TestUpdateSubscribeApplicationMovesDefaultAndEnablesTarget(t *testing.T) {
	ctx := context.Background()
	db, model := newSubscribeApplicationTestModel(t)
	current := seedSubscribeApplicationLogicRow(t, db)
	target := client.SubscribeApplication{
		Id: 13, Name: "Target", UserAgent: "TargetAgent", Enabled: false,
		OutputFormat: "json", DownloadLink: "{}",
	}
	if err := db.Select("*").Create(&target).Error; err != nil {
		t.Fatal(err)
	}
	isDefault := true
	disabled := false

	got, err := NewUpdateSubscribeApplicationLogic(ctx, &svc.ServiceContext{ClientModel: model}).
		UpdateSubscribeApplication(&types.UpdateSubscribeApplicationRequest{
			Id: target.Id, IsDefault: &isDefault, Enabled: &disabled,
		})
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsDefault || !got.Enabled {
		t.Fatalf("new default was not enabled: %#v", got)
	}

	var savedCurrent, savedTarget client.SubscribeApplication
	if err := db.First(&savedCurrent, current.Id).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&savedTarget, target.Id).Error; err != nil {
		t.Fatal(err)
	}
	if savedCurrent.IsDefault || !savedTarget.IsDefault || !savedTarget.Enabled {
		t.Fatalf("default transition not persisted: current=%#v target=%#v", savedCurrent, savedTarget)
	}
}

func TestDefaultSubscribeApplicationCannotBeDisabledOrDeleted(t *testing.T) {
	ctx := context.Background()
	db, model := newSubscribeApplicationTestModel(t)
	row := seedSubscribeApplicationLogicRow(t, db)
	disabled := false

	if _, err := NewUpdateSubscribeApplicationLogic(ctx, &svc.ServiceContext{ClientModel: model}).
		UpdateSubscribeApplication(&types.UpdateSubscribeApplicationRequest{Id: row.Id, Enabled: &disabled}); err == nil {
		t.Fatal("expected disabling the default client to fail")
	}
	if err := NewDeleteSubscribeApplicationLogic(ctx, &svc.ServiceContext{ClientModel: model}).
		DeleteSubscribeApplication(&types.DeleteSubscribeApplicationRequest{Id: row.Id}); err == nil {
		t.Fatal("expected deleting the default client to fail")
	}
}

func TestCreateSubscribeApplicationKeepsSingleDefault(t *testing.T) {
	ctx := context.Background()
	db, model := newSubscribeApplicationTestModel(t)
	logic := NewCreateSubscribeApplicationLogic(ctx, &svc.ServiceContext{ClientModel: model})

	first, err := logic.CreateSubscribeApplication(&types.CreateSubscribeApplicationRequest{
		Name: "First", UserAgent: "FirstAgent", SubscribeTemplate: "template", OutputFormat: "json", DownloadLink: types.DownloadLink{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !first.IsDefault || !first.Enabled {
		t.Fatalf("first application should become enabled default: %#v", first)
	}

	second, err := logic.CreateSubscribeApplication(&types.CreateSubscribeApplicationRequest{
		Name: "Second", UserAgent: "SecondAgent", SubscribeTemplate: "template", OutputFormat: "json", IsDefault: true, DownloadLink: types.DownloadLink{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var defaults int64
	if err := db.Model(&client.SubscribeApplication{}).Where("is_default = ?", true).Count(&defaults).Error; err != nil {
		t.Fatal(err)
	}
	if !second.IsDefault || defaults != 1 {
		t.Fatalf("expected one default after replacement: second=%#v count=%d", second, defaults)
	}
}

func TestCreateSubscribeApplicationNormalizesDownloadLink(t *testing.T) {
	ctx := context.Background()
	db, model := newSubscribeApplicationTestModel(t)

	logic := NewCreateSubscribeApplicationLogic(ctx, &svc.ServiceContext{ClientModel: model})
	got, err := logic.CreateSubscribeApplication(&types.CreateSubscribeApplicationRequest{
		Name:              "Alpha Client",
		UserAgent:         "AlphaAgent",
		SubscribeTemplate: "template",
		OutputFormat:      "json",
		DownloadLink: types.DownloadLink{
			Android: " https://example.com/android ",
			Windows: " ",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.DownloadLink.Android != "https://example.com/android" || got.DownloadLink.Windows != "" {
		t.Fatalf("response download link not normalized: %#v", got.DownloadLink)
	}

	var saved client.SubscribeApplication
	if err := db.First(&saved, got.Id).Error; err != nil {
		t.Fatal(err)
	}
	if want := `{"android":"https://example.com/android"}`; saved.DownloadLink != want {
		t.Fatalf("saved download link = %q, want %q", saved.DownloadLink, want)
	}
}

func TestPreviewProxiesFromEnabledNodesDoesNotFallbackToPerServerQueries(t *testing.T) {
	ctx := context.Background()
	db := newPreviewNodeTestDB(t)
	model := node.NewModel(db, nil)
	enabled := true

	servers := []*node.Server{
		{Id: 7, Name: "server-a", Address: "a.example.com", Sort: 1},
		{Id: 8, Name: "server-b", Address: "b.example.com", Sort: 2},
	}
	if err := db.Create(&servers).Error; err != nil {
		t.Fatal(err)
	}
	nodes := []*node.Node{
		{Id: 1, Name: "node-a-1", ServerId: 7, Protocol: "vless", InboundAlias: "vless", Enabled: &enabled, Sort: 1},
		{Id: 2, Name: "node-a-2", ServerId: 7, Protocol: "vmess", InboundAlias: "vmess", Enabled: &enabled, Sort: 2},
		{Id: 3, Name: "node-b-1", ServerId: 8, Protocol: "trojan", InboundAlias: "trojan", Enabled: &enabled, Sort: 3},
	}
	if err := db.Create(&nodes).Error; err != nil {
		t.Fatal(err)
	}

	var bindingQueries int64
	callbackName := "alpha:count_server_xray_template_queries"
	if err := db.Callback().Query().After("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		sql := strings.ToLower(tx.Statement.SQL.String())
		if strings.Contains(sql, "server_xray_templates") {
			atomic.AddInt64(&bindingQueries, 1)
		}
	}); err != nil {
		t.Fatal(err)
	}

	logic := NewPreviewSubscribeTemplateLogic(ctx, &svc.ServiceContext{NodeModel: model})
	got, err := logic.previewProxiesFromEnabledNodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("preview should not render proxies without template bindings: %#v", got)
	}
	if gotQueries := atomic.LoadInt64(&bindingQueries); gotQueries != 1 {
		t.Fatalf("server xray template queries = %d, want one batch attempt", gotQueries)
	}
}

func TestPreviewInboundBindingIndexCacheUsesLinearPathForSmallInput(t *testing.T) {
	var cache previewInboundBindingIndexCache
	first := xraytemplate.NewInboundBindingIndex(nil)
	updated := xraytemplate.NewInboundBindingIndex(nil)
	cache.Set(7, first)
	cache.Set(8, xraytemplate.NewInboundBindingIndex(nil))
	cache.Set(7, updated)

	if cache.index != nil {
		t.Fatal("small preview inbound binding index cache should not allocate map")
	}
	got, ok := cache.Get(7)
	if !ok || got != updated {
		t.Fatalf("cache Get(7) = %#v/%v, want updated/true", got, ok)
	}
	if _, ok := cache.Get(9); ok {
		t.Fatal("cache should miss unknown server id")
	}
}

func TestPreviewInboundBindingIndexCacheUpgradesLargeInputToMap(t *testing.T) {
	var cache previewInboundBindingIndexCache
	for i := int64(1); i <= previewInboundBindingIndexLinearLimit+1; i++ {
		cache.Set(i, xraytemplate.NewInboundBindingIndex(nil))
	}

	if cache.index == nil {
		t.Fatal("large preview inbound binding index cache should allocate map")
	}
	if len(cache.rows) != previewInboundBindingIndexLinearLimit+1 {
		t.Fatalf("cache rows len = %d, want %d", len(cache.rows), previewInboundBindingIndexLinearLimit+1)
	}
}

func newSubscribeApplicationTestModel(t *testing.T) (*gorm.DB, client.Model) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&client.SubscribeApplication{}); err != nil {
		t.Fatal(err)
	}
	return db, client.NewSubscribeApplicationModel(db)
}

func newPreviewNodeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&node.Server{}, &node.Node{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedSubscribeApplicationLogicRow(t *testing.T, db *gorm.DB) client.SubscribeApplication {
	t.Helper()
	row := client.SubscribeApplication{
		Id:                12,
		Name:              "Alpha Client",
		Icon:              "icon",
		Description:       "description",
		Scheme:            "alpha://",
		UserAgent:         "AlphaAgent",
		Enabled:           true,
		IsDefault:         true,
		SubscribeTemplate: "large-template",
		OutputFormat:      "json",
		DownloadLink:      `{"android":"https://example.com/android"}`,
	}
	if err := db.Select("*").Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	return row
}
