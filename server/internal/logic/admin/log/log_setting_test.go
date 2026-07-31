package log

import (
	"context"
	"testing"

	"github.com/neibniz/alpha/server/internal/config"
	modelSystem "github.com/neibniz/alpha/server/internal/model/system"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetLogSettingMapsBoolResponse(t *testing.T) {
	db, systemModel := setupLogSettingTest(t)
	seedLogSettingRows(t, db, "true", "14")

	got, err := NewGetLogSettingLogic(context.Background(), &svc.ServiceContext{SystemModel: systemModel}).
		GetLogSetting()
	if err != nil {
		t.Fatal(err)
	}
	if !got.AutoClear || got.ClearDays != 14 {
		t.Fatalf("log setting not mapped: %#v", got)
	}
}

func TestUpdateLogSettingPersistsBoolAndRuntimeConfig(t *testing.T) {
	db, systemModel := setupLogSettingTest(t)
	seedLogSettingRows(t, db, "false", "7")
	svcCtx := &svc.ServiceContext{
		Config:      config.Config{Log: config.Log{AutoClear: false, ClearDays: 7}},
		SystemModel: systemModel,
	}

	err := NewUpdateLogSettingLogic(context.Background(), svcCtx).
		UpdateLogSetting(&types.LogSetting{AutoClear: true, ClearDays: 30})
	if err != nil {
		t.Fatal(err)
	}

	var rows []modelSystem.System
	if err := db.Where("category = ?", "log").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.Key] = row.Value
	}
	if values["AutoClear"] != "true" || values["ClearDays"] != "30" {
		t.Fatalf("log settings not persisted: %#v", values)
	}
	if !svcCtx.Config.Log.AutoClear || svcCtx.Config.Log.ClearDays != 30 {
		t.Fatalf("runtime log config not updated: %#v", svcCtx.Config.Log)
	}
}

func setupLogSettingTest(t *testing.T) (*gorm.DB, modelSystem.Model) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelSystem.System{}); err != nil {
		t.Fatal(err)
	}
	return db, modelSystem.NewModel(db, nil)
}

func seedLogSettingRows(t *testing.T, db *gorm.DB, autoClear, clearDays string) {
	t.Helper()
	rows := []modelSystem.System{
		{Category: "log", Key: "AutoClear", Value: autoClear, Type: "bool"},
		{Category: "log", Key: "ClearDays", Value: clearDays, Type: "int64"},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
}
