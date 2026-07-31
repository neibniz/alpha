package server

import (
	"context"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFilterXrayTemplateListKeepsEmptyListShape(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&node.XrayTemplate{}); err != nil {
		t.Fatal(err)
	}

	resp, err := NewFilterXrayTemplateListLogic(context.Background(), &svc.ServiceContext{NodeModel: node.NewModel(db, nil)}).
		FilterXrayTemplateList(&types.FilterXrayTemplateListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.List == nil || len(resp.List) != 0 {
		t.Fatalf("empty xray template filter list = %#v, want non-nil empty slice", resp.List)
	}
	if resp.Total != 0 {
		t.Fatalf("empty xray template total = %d, want 0", resp.Total)
	}
}

func TestQueryXrayTemplateCatalogKeepsEmptyListShape(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&node.XrayTemplate{}); err != nil {
		t.Fatal(err)
	}

	resp, err := NewQueryXrayTemplateCatalogLogic(context.Background(), &svc.ServiceContext{NodeModel: node.NewModel(db, nil)}).
		QueryXrayTemplateCatalog(&types.QueryXrayTemplateCatalogRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.List == nil || len(resp.List) != 0 {
		t.Fatalf("empty xray template catalog list = %#v, want non-nil empty slice", resp.List)
	}
	if resp.Total != 0 {
		t.Fatalf("empty xray template catalog total = %d, want 0", resp.Total)
	}
}
