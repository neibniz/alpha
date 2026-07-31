package server

import (
	"context"
	"reflect"
	"testing"

	modelNode "github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFilterNodeListNormalizesResponseTags(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelNode.Node{}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	if err := db.Create(&modelNode.Node{
		Name:         "node-a",
		Tags:         " edge, premium,,edge ",
		Port:         443,
		Address:      "node.example.com",
		ServerId:     1,
		Protocol:     "vless",
		InboundAlias: "VLESS",
		Enabled:      &enabled,
		Sort:         1,
	}).Error; err != nil {
		t.Fatal(err)
	}

	logic := NewFilterNodeListLogic(context.Background(), &svc.ServiceContext{
		NodeModel: modelNode.NewModel(db, nil),
	})
	resp, err := logic.FilterNodeList(&types.FilterNodeListRequest{
		Page: 1,
		Size: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("response = %#v, want one node", resp)
	}
	if want := []string{"edge", "premium"}; !reflect.DeepEqual(resp.List[0].Tags, want) {
		t.Fatalf("tags = %#v, want %#v", resp.List[0].Tags, want)
	}
}
