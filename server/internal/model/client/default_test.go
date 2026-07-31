package client

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDownloadLinkMarshalShapeAndNilReceiver(t *testing.T) {
	link := &DownloadLink{IOS: "https://ios.example.test", Linux: "https://linux.example.test"}
	data, err := link.Marshal()
	if err != nil {
		t.Fatalf("DownloadLink.Marshal() error = %v", err)
	}
	if got, want := string(data), `{"ios":"https://ios.example.test","linux":"https://linux.example.test"}`; got != want {
		t.Fatalf("DownloadLink.Marshal() = %s, want %s", got, want)
	}

	var nilLink *DownloadLink
	data, err = nilLink.Marshal()
	if err != nil {
		t.Fatalf("nil DownloadLink.Marshal() error = %v", err)
	}
	if string(data) != "{}" {
		t.Fatalf("nil DownloadLink.Marshal() = %s, want {}", data)
	}
}

func TestDownloadLinkUnmarshalClearsEmptyAndNull(t *testing.T) {
	for _, raw := range [][]byte{nil, {}, []byte("  "), []byte("null"), []byte(" \n null\t")} {
		link := DownloadLink{IOS: "https://ios.example.test"}
		if err := link.Unmarshal(raw); err != nil {
			t.Fatalf("DownloadLink.Unmarshal(%q) error = %v", raw, err)
		}
		if link != (DownloadLink{}) {
			t.Fatalf("DownloadLink.Unmarshal(%q) = %#v, want zero", raw, link)
		}
	}
}

func TestDownloadLinkUnmarshalDirectShape(t *testing.T) {
	var link DownloadLink
	if err := link.Unmarshal([]byte(`{"ios":"https://ios.example.test","android":"https://android.example.test"}`)); err != nil {
		t.Fatalf("DownloadLink.Unmarshal() error = %v", err)
	}
	if link.IOS != "https://ios.example.test" || link.Android != "https://android.example.test" {
		t.Fatalf("DownloadLink.Unmarshal() = %#v", link)
	}
}

func TestSubscribeApplicationClientColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(subscribeApplicationClientColumns))
	for _, column := range subscribeApplicationClientColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"name",
		"icon",
		"description",
		"scheme",
		"enabled",
		"is_default",
		"download_link",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("subscribe application client columns must include %q", column)
		}
	}

	forbidden := []string{
		"subscribe_template",
		"output_format",
		"user_agent",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("subscribe application client columns must not include %q", column)
		}
	}
}

func TestSubscribeApplicationAdminListColumnsStayFocused(t *testing.T) {
	columns := make(map[string]struct{}, len(subscribeApplicationAdminListColumns))
	for _, column := range subscribeApplicationAdminListColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"name",
		"icon",
		"description",
		"scheme",
		"user_agent",
		"enabled",
		"is_default",
		"output_format",
		"download_link",
		"created_at",
		"updated_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("subscribe application admin list columns must include %q", column)
		}
	}

	forbidden := []string{
		"subscribe_template",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("subscribe application admin list columns must not include %q", column)
		}
	}
}

func TestSubscribeApplicationUpdateReadColumnsSkipTemplate(t *testing.T) {
	columns := make(map[string]struct{}, len(subscribeApplicationUpdateReadColumns))
	for _, column := range subscribeApplicationUpdateReadColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"name",
		"icon",
		"description",
		"scheme",
		"user_agent",
		"enabled",
		"is_default",
		"output_format",
		"download_link",
		"created_at",
		"updated_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("subscribe application update-read columns must include %q", column)
		}
	}

	if _, ok := columns["subscribe_template"]; ok {
		t.Fatal("subscribe application update-read columns must not include subscribe_template")
	}
}

func TestSubscribeApplicationRuntimeColumnsStayFocused(t *testing.T) {
	columns := make(map[string]struct{}, len(subscribeApplicationRuntimeColumns))
	for _, column := range subscribeApplicationRuntimeColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"name",
		"user_agent",
		"is_default",
		"subscribe_template",
		"output_format",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("subscribe application runtime columns must include %q", column)
		}
	}

	forbidden := []string{
		"icon",
		"description",
		"scheme",
		"download_link",
		"created_at",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("subscribe application runtime columns must not include %q", column)
		}
	}
}

func TestSubscribeApplicationMatcherColumnsStayFocused(t *testing.T) {
	columns := make(map[string]struct{}, len(subscribeApplicationMatcherColumns))
	for _, column := range subscribeApplicationMatcherColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"name",
		"user_agent",
		"is_default",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("subscribe application matcher columns must include %q", column)
		}
	}

	forbidden := []string{
		"subscribe_template",
		"output_format",
		"icon",
		"description",
		"scheme",
		"download_link",
		"created_at",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("subscribe application matcher columns must not include %q", column)
		}
	}
}

func TestSubscribeApplicationPreviewColumnsStayFocused(t *testing.T) {
	columns := make(map[string]struct{}, len(subscribeApplicationPreviewColumns))
	for _, column := range subscribeApplicationPreviewColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"subscribe_template",
		"output_format",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("subscribe application preview columns must include %q", column)
		}
	}

	forbidden := []string{
		"name",
		"icon",
		"description",
		"scheme",
		"user_agent",
		"download_link",
		"created_at",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("subscribe application preview columns must not include %q", column)
		}
	}
}

func TestFindOneForPreviewSkipsDisplayFields(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeApplication{}); err != nil {
		t.Fatal(err)
	}

	row := &SubscribeApplication{
		Id:                9,
		Name:              "preview",
		Icon:              "large-icon",
		Description:       "description",
		Scheme:            "preview://",
		UserAgent:         "PreviewAgent",
		Enabled:           true,
		IsDefault:         true,
		SubscribeTemplate: "large-template",
		OutputFormat:      "json",
		DownloadLink:      `{"ios":"https://example.com/ios"}`,
	}
	if err := db.Select("*").Create(row).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewSubscribeApplicationModel(db).FindOneForPreview(ctx, row.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != row.Id || got.SubscribeTemplate != "large-template" || got.OutputFormat != "json" {
		t.Fatalf("preview fields not loaded: %#v", got)
	}
	if got.Name != "" || got.Icon != "" || got.Description != "" || got.Scheme != "" || got.UserAgent != "" || got.DownloadLink != "" || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("preview query loaded display/audit fields: %#v", got)
	}
}

func TestFindOneForUpdateSkipsTemplateField(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeApplication{}); err != nil {
		t.Fatal(err)
	}

	row := &SubscribeApplication{
		Id:                19,
		Name:              "update",
		Icon:              "icon",
		Description:       "description",
		Scheme:            "update://",
		UserAgent:         "UpdateAgent",
		Enabled:           true,
		IsDefault:         true,
		SubscribeTemplate: "large-template",
		OutputFormat:      "json",
		DownloadLink:      `{"ios":"https://example.com/ios"}`,
	}
	if err := db.Select("*").Create(row).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewSubscribeApplicationModel(db).FindOneForUpdate(ctx, row.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != row.Id || got.Name != "update" || got.UserAgent != "UpdateAgent" || got.OutputFormat != "json" || got.DownloadLink == "" {
		t.Fatalf("update-read fields not loaded: %#v", got)
	}
	if got.SubscribeTemplate != "" {
		t.Fatalf("update-read query loaded template field: %#v", got)
	}
}

func TestSubscribeApplicationListMethodsReturnNonNilEmptySlices(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeApplication{}); err != nil {
		t.Fatal(err)
	}

	model := NewSubscribeApplicationModel(db)
	total, list, err := model.List(ctx, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || list == nil || len(list) != 0 {
		t.Fatalf("empty list = total %d list %#v, want total 0 and non-nil empty slice", total, list)
	}

	enabled, err := model.ListEnabled(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if enabled == nil || len(enabled) != 0 {
		t.Fatalf("empty enabled list = %#v, want non-nil empty slice", enabled)
	}

	userAgents, err := model.ListEnabledUserAgents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if userAgents == nil || len(userAgents) != 0 {
		t.Fatalf("empty enabled user agents = %#v, want non-nil empty slice", userAgents)
	}

	matchers, err := model.ListEnabledMatchers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if matchers == nil || len(matchers) != 0 {
		t.Fatalf("empty enabled matchers = %#v, want non-nil empty slice", matchers)
	}

	runtime, err := model.ListEnabledRuntime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if runtime == nil || len(runtime) != 0 {
		t.Fatalf("empty enabled runtime list = %#v, want non-nil empty slice", runtime)
	}

	summary, err := model.ListEnabledSummary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if summary == nil || len(summary) != 0 {
		t.Fatalf("empty enabled summary list = %#v, want non-nil empty slice", summary)
	}
}

func TestListSkipsTemplateField(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeApplication{}); err != nil {
		t.Fatal(err)
	}

	rows := []SubscribeApplication{
		{
			Id:                2,
			Name:              "second",
			Icon:              "icon-two",
			Description:       "second description",
			Scheme:            "two://",
			UserAgent:         "SecondAgent",
			Enabled:           false,
			IsDefault:         false,
			SubscribeTemplate: "large-template-two",
			OutputFormat:      "yaml",
			DownloadLink:      `{"ios":"https://example.com/ios"}`,
		},
		{
			Id:                1,
			Name:              "first",
			Icon:              "icon-one",
			Description:       "first description",
			Scheme:            "one://",
			UserAgent:         "FirstAgent",
			Enabled:           true,
			IsDefault:         true,
			SubscribeTemplate: "large-template-one",
			OutputFormat:      "json",
			DownloadLink:      `{"android":"https://example.com/android"}`,
		},
	}
	if err := db.Select("*").Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	total, list, err := NewSubscribeApplicationModel(db).List(ctx, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("admin list total = %d, want 2", total)
	}
	if len(list) != 2 {
		t.Fatalf("admin list count = %d, want 2", len(list))
	}
	if cap(list) != 2 {
		t.Fatalf("admin list capacity = %d, want 2", cap(list))
	}
	if list[0].Id != 1 || list[1].Id != 2 {
		t.Fatalf("admin list order = [%d %d], want [1 2]", list[0].Id, list[1].Id)
	}
	if list[0].Name != "first" || list[0].UserAgent != "FirstAgent" || list[0].OutputFormat != "json" || list[0].DownloadLink == "" {
		t.Fatalf("admin list missing metadata fields: %#v", list[0])
	}
	if list[0].SubscribeTemplate != "" || list[1].SubscribeTemplate != "" {
		t.Fatalf("admin list loaded template fields: %#v %#v", list[0], list[1])
	}
}

func TestListPaginatesAndKeepsTotal(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeApplication{}); err != nil {
		t.Fatal(err)
	}

	rows := []SubscribeApplication{
		{Id: 1, Name: "first", SubscribeTemplate: "large-template-one"},
		{Id: 2, Name: "second", SubscribeTemplate: "large-template-two"},
		{Id: 3, Name: "third", SubscribeTemplate: "large-template-three"},
	}
	if err := db.Select("*").Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	total, list, err := NewSubscribeApplicationModel(db).List(ctx, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Fatalf("admin list total = %d, want 3", total)
	}
	if len(list) != 1 {
		t.Fatalf("admin list page count = %d, want 1", len(list))
	}
	if cap(list) != 1 {
		t.Fatalf("admin list page capacity = %d, want 1", cap(list))
	}
	if list[0].Id != 2 || list[0].Name != "second" {
		t.Fatalf("admin list page item = %#v, want second row", list[0])
	}
	if list[0].SubscribeTemplate != "" {
		t.Fatalf("admin list page loaded template field: %#v", list[0])
	}
}

func TestFilterSubscribeApplicationsBySearchAndStatus(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeApplication{}); err != nil {
		t.Fatal(err)
	}

	rows := []SubscribeApplication{
		{Id: 1, Name: "Clash", UserAgent: "Mihomo", Enabled: true, OutputFormat: "yaml"},
		{Id: 2, Name: "sing-box", UserAgent: "sing-box", Enabled: true, OutputFormat: "json"},
		{Id: 3, Name: "Legacy", UserAgent: "Shadowrocket", Enabled: false, OutputFormat: "base64"},
	}
	if err := db.Select("*").Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&SubscribeApplication{}).Where("id = ?", 3).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}

	enabled := true
	total, list, err := NewSubscribeApplicationModel(db).Filter(ctx, 1, 10, SubscribeApplicationListFilter{
		Search:  "JSON",
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].Id != 2 {
		t.Fatalf("filtered applications = total %d list %#v, want sing-box only", total, list)
	}

	disabled := false
	total, list, err = NewSubscribeApplicationModel(db).Filter(ctx, 1, 10, SubscribeApplicationListFilter{
		Search:  "shadow",
		Enabled: &disabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].Id != 3 {
		t.Fatalf("disabled applications = total %d list %#v, want legacy only", total, list)
	}
}

func TestListEnabledSkipsTemplateField(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeApplication{}); err != nil {
		t.Fatal(err)
	}

	rows := []SubscribeApplication{
		{
			Id:                1,
			Name:              "enabled",
			Icon:              "icon-one",
			Description:       "enabled description",
			Scheme:            "enabled://",
			UserAgent:         "EnabledAgent",
			Enabled:           true,
			IsDefault:         true,
			SubscribeTemplate: "large-template-one",
			OutputFormat:      "json",
			DownloadLink:      `{"android":"https://example.com/android"}`,
		},
		{
			Id:                2,
			Name:              "disabled",
			Icon:              "icon-two",
			Description:       "disabled description",
			Scheme:            "disabled://",
			UserAgent:         "DisabledAgent",
			Enabled:           false,
			IsDefault:         false,
			SubscribeTemplate: "large-template-two",
			OutputFormat:      "yaml",
			DownloadLink:      `{"ios":"https://example.com/ios"}`,
		},
	}
	if err := db.Select("*").Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&SubscribeApplication{}).Where("id = ?", 2).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}

	list, err := NewSubscribeApplicationModel(db).ListEnabled(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("enabled admin list count = %d, want 1", len(list))
	}
	if list[0].Id != 1 || list[0].Name != "enabled" || list[0].UserAgent != "EnabledAgent" || list[0].OutputFormat != "json" {
		t.Fatalf("enabled admin list missing metadata fields: %#v", list[0])
	}
	if list[0].SubscribeTemplate != "" {
		t.Fatalf("enabled admin list loaded template field: %#v", list[0])
	}
}

func TestUpdateSubscribeApplicationWritesZeroValues(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeApplication{}); err != nil {
		t.Fatal(err)
	}

	row := &SubscribeApplication{
		Id:                10,
		Name:              "existing",
		Icon:              "icon",
		Description:       "description",
		Scheme:            "alpha://",
		UserAgent:         "AlphaAgent",
		Enabled:           true,
		IsDefault:         true,
		SubscribeTemplate: "template",
		OutputFormat:      "yaml",
		DownloadLink:      `{"ios":"https://example.com/ios"}`,
	}
	if err := db.Select("*").Create(row).Error; err != nil {
		t.Fatal(err)
	}

	update := &SubscribeApplication{
		Id:                row.Id,
		Name:              "",
		Icon:              "",
		Description:       "",
		Scheme:            "",
		UserAgent:         "",
		Enabled:           false,
		IsDefault:         false,
		SubscribeTemplate: "",
		OutputFormat:      "",
		DownloadLink:      "",
	}
	if err := NewSubscribeApplicationModel(db).Update(ctx, update); err != nil {
		t.Fatal(err)
	}

	var got SubscribeApplication
	if err := db.First(&got, row.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.Name != "" || got.Icon != "" || got.Description != "" || got.Scheme != "" || got.UserAgent != "" || got.SubscribeTemplate != "" || got.OutputFormat != "" || got.DownloadLink != "" {
		t.Fatalf("zero string fields were not persisted: %#v", got)
	}
	if got.Enabled || got.IsDefault {
		t.Fatalf("zero bool fields were not persisted: %#v", got)
	}
}

func TestUpdateSubscribeApplicationFieldsUpdatesOnlyRequestedColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeApplication{}); err != nil {
		t.Fatal(err)
	}

	row := &SubscribeApplication{
		Id:                11,
		Name:              "existing",
		Icon:              "icon",
		Description:       "description",
		Scheme:            "alpha://",
		UserAgent:         "AlphaAgent",
		Enabled:           true,
		IsDefault:         true,
		SubscribeTemplate: "template",
		OutputFormat:      "yaml",
		DownloadLink:      `{"ios":"https://example.com/ios"}`,
	}
	if err := db.Select("*").Create(row).Error; err != nil {
		t.Fatal(err)
	}

	enabled := false
	name := ""
	if err := NewSubscribeApplicationModel(db).UpdateFields(ctx, row.Id, SubscribeApplicationFieldsUpdate{
		Name:    &name,
		Enabled: &enabled,
	}); err != nil {
		t.Fatal(err)
	}

	var got SubscribeApplication
	if err := db.First(&got, row.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.Name != "" || got.Enabled {
		t.Fatalf("requested fields were not updated: %#v", got)
	}
	if got.Icon != row.Icon || got.Description != row.Description || got.Scheme != row.Scheme || got.UserAgent != row.UserAgent || got.SubscribeTemplate != row.SubscribeTemplate || got.OutputFormat != row.OutputFormat || got.DownloadLink != row.DownloadLink || !got.IsDefault {
		t.Fatalf("unrequested fields changed: %#v", got)
	}
}

func TestUpdateSubscribeApplicationReturnsNotFoundWithoutPreRead(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeApplication{}); err != nil {
		t.Fatal(err)
	}

	err = NewSubscribeApplicationModel(db).Update(ctx, &SubscribeApplication{Id: 404})
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Update missing row error = %v, want %v", err, gorm.ErrRecordNotFound)
	}
}

func TestListEnabledSummarySkipsTemplateFields(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeApplication{}); err != nil {
		t.Fatal(err)
	}

	rows := []SubscribeApplication{
		{
			Id:                2,
			Name:              "enabled-two",
			Icon:              "icon-two",
			Description:       "second",
			Scheme:            "two://",
			UserAgent:         "HeavyAgent",
			Enabled:           true,
			IsDefault:         false,
			SubscribeTemplate: "large-template-two",
			OutputFormat:      "yaml",
			DownloadLink:      `{"ios":"https://example.com/ios"}`,
		},
		{
			Id:                1,
			Name:              "enabled-one",
			Icon:              "icon-one",
			Description:       "first",
			Scheme:            "one://",
			UserAgent:         "OtherAgent",
			Enabled:           true,
			IsDefault:         true,
			SubscribeTemplate: "large-template-one",
			OutputFormat:      "json",
			DownloadLink:      `{"android":"https://example.com/android"}`,
		},
		{
			Id:                3,
			Name:              "disabled",
			Icon:              "disabled-icon",
			Description:       "disabled",
			Scheme:            "disabled://",
			UserAgent:         "DisabledAgent",
			Enabled:           false,
			IsDefault:         false,
			SubscribeTemplate: "disabled-template",
			OutputFormat:      "base64",
			DownloadLink:      "{}",
		},
	}
	if err := db.Select("*").Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&SubscribeApplication{}).Where("id = ?", 3).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}

	model := NewSubscribeApplicationModel(db)
	list, err := model.ListEnabledSummary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("enabled summary count = %d, want 2", len(list))
	}
	if list[0].Id != 1 || list[1].Id != 2 {
		t.Fatalf("enabled summary order = [%d %d], want [1 2]", list[0].Id, list[1].Id)
	}
	if list[0].Name != "enabled-one" || list[0].Scheme != "one://" || list[0].DownloadLink == "" {
		t.Fatalf("enabled summary missing response fields: %#v", list[0])
	}
	if list[0].SubscribeTemplate != "" || list[0].OutputFormat != "" || list[0].UserAgent != "" {
		t.Fatalf("enabled summary loaded template-only fields: %#v", list[0])
	}
}

func TestListEnabledUserAgentsPlucksOnlyEnabledAgents(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeApplication{}); err != nil {
		t.Fatal(err)
	}

	rows := []SubscribeApplication{
		{
			Id:                2,
			Name:              "enabled-two",
			Icon:              "icon-two",
			Description:       "second",
			Scheme:            "two://",
			UserAgent:         "HeavyAgent",
			Enabled:           true,
			IsDefault:         false,
			SubscribeTemplate: "large-template-two",
			OutputFormat:      "yaml",
			DownloadLink:      `{"ios":"https://example.com/ios"}`,
		},
		{
			Id:                1,
			Name:              "enabled-one",
			Icon:              "icon-one",
			Description:       "first",
			Scheme:            "one://",
			UserAgent:         "OtherAgent",
			Enabled:           true,
			IsDefault:         true,
			SubscribeTemplate: "large-template-one",
			OutputFormat:      "json",
			DownloadLink:      `{"android":"https://example.com/android"}`,
		},
		{
			Id:                3,
			Name:              "disabled",
			Icon:              "disabled-icon",
			Description:       "disabled",
			Scheme:            "disabled://",
			UserAgent:         "DisabledAgent",
			Enabled:           false,
			IsDefault:         false,
			SubscribeTemplate: "disabled-template",
			OutputFormat:      "base64",
			DownloadLink:      "{}",
		},
	}
	if err := db.Select("*").Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&SubscribeApplication{}).Where("id = ?", 3).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewSubscribeApplicationModel(db).ListEnabledUserAgents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("enabled user agents count = %d, want 2", len(got))
	}
	if got[0] != "OtherAgent" || got[1] != "HeavyAgent" {
		t.Fatalf("enabled user agents = %#v, want [OtherAgent HeavyAgent]", got)
	}
}

func TestListEnabledMatchersSkipsTemplateFields(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeApplication{}); err != nil {
		t.Fatal(err)
	}

	rows := []SubscribeApplication{
		{
			Id:                2,
			Name:              "enabled-two",
			Icon:              "icon-two",
			Description:       "second",
			Scheme:            "two://",
			UserAgent:         "HeavyAgent",
			Enabled:           true,
			IsDefault:         false,
			SubscribeTemplate: "large-template-two",
			OutputFormat:      "yaml",
			DownloadLink:      `{"ios":"https://example.com/ios"}`,
		},
		{
			Id:                1,
			Name:              "enabled-one",
			Icon:              "icon-one",
			Description:       "first",
			Scheme:            "one://",
			UserAgent:         "OtherAgent",
			Enabled:           true,
			IsDefault:         true,
			SubscribeTemplate: "large-template-one",
			OutputFormat:      "json",
			DownloadLink:      `{"android":"https://example.com/android"}`,
		},
		{
			Id:                3,
			Name:              "disabled",
			Icon:              "disabled-icon",
			Description:       "disabled",
			Scheme:            "disabled://",
			UserAgent:         "DisabledAgent",
			Enabled:           false,
			IsDefault:         false,
			SubscribeTemplate: "disabled-template",
			OutputFormat:      "base64",
			DownloadLink:      "{}",
		},
	}
	if err := db.Select("*").Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&SubscribeApplication{}).Where("id = ?", 3).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}

	list, err := NewSubscribeApplicationModel(db).ListEnabledMatchers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("enabled matcher count = %d, want 2", len(list))
	}
	if list[0].Id != 1 || list[1].Id != 2 {
		t.Fatalf("enabled matcher order = [%d %d], want [1 2]", list[0].Id, list[1].Id)
	}
	if list[0].Name != "enabled-one" || list[0].UserAgent != "OtherAgent" || !list[0].IsDefault {
		t.Fatalf("enabled matcher missing match fields: %#v", list[0])
	}
	if list[0].SubscribeTemplate != "" || list[0].OutputFormat != "" || list[0].Icon != "" || list[0].Description != "" || list[0].Scheme != "" || list[0].DownloadLink != "" || !list[0].CreatedAt.IsZero() || !list[0].UpdatedAt.IsZero() {
		t.Fatalf("enabled matcher loaded generation/display fields: %#v", list[0])
	}
}

func TestListEnabledRuntimeSkipsDisplayFields(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeApplication{}); err != nil {
		t.Fatal(err)
	}

	rows := []SubscribeApplication{
		{
			Id:                2,
			Name:              "enabled-two",
			Icon:              "icon-two",
			Description:       "second",
			Scheme:            "two://",
			UserAgent:         "HeavyAgent",
			Enabled:           true,
			IsDefault:         false,
			SubscribeTemplate: "large-template-two",
			OutputFormat:      "yaml",
			DownloadLink:      `{"ios":"https://example.com/ios"}`,
		},
		{
			Id:                1,
			Name:              "enabled-one",
			Icon:              "icon-one",
			Description:       "first",
			Scheme:            "one://",
			UserAgent:         "OtherAgent",
			Enabled:           true,
			IsDefault:         true,
			SubscribeTemplate: "large-template-one",
			OutputFormat:      "json",
			DownloadLink:      `{"android":"https://example.com/android"}`,
		},
		{
			Id:                3,
			Name:              "disabled",
			Icon:              "disabled-icon",
			Description:       "disabled",
			Scheme:            "disabled://",
			UserAgent:         "DisabledAgent",
			Enabled:           false,
			IsDefault:         false,
			SubscribeTemplate: "disabled-template",
			OutputFormat:      "base64",
			DownloadLink:      "{}",
		},
	}
	if err := db.Select("*").Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&SubscribeApplication{}).Where("id = ?", 3).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}

	model := NewSubscribeApplicationModel(db)
	list, err := model.ListEnabledRuntime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("enabled runtime count = %d, want 2", len(list))
	}
	if list[0].Id != 1 || list[1].Id != 2 {
		t.Fatalf("enabled runtime order = [%d %d], want [1 2]", list[0].Id, list[1].Id)
	}
	if list[0].Name != "enabled-one" || list[0].UserAgent != "OtherAgent" || list[0].SubscribeTemplate != "large-template-one" || list[0].OutputFormat != "json" || !list[0].IsDefault {
		t.Fatalf("enabled runtime missing generation fields: %#v", list[0])
	}
	if list[0].Icon != "" || list[0].Description != "" || list[0].Scheme != "" || list[0].DownloadLink != "" || !list[0].CreatedAt.IsZero() || !list[0].UpdatedAt.IsZero() {
		t.Fatalf("enabled runtime loaded display/audit fields: %#v", list[0])
	}
}
