package server

import (
	"context"
	"testing"

	"github.com/neibniz/alpha/server/internal/config"
	modelNode "github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/nodeauth"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestQueryServerNodeTokenReturnsBoundTokenOnlyForExistingServer(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelNode.Server{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&modelNode.Server{Id: 7, Name: "server-a", Address: "192.0.2.7"}).Error; err != nil {
		t.Fatal(err)
	}
	const masterSecret = "node-master-secret-that-is-long-enough-for-tests"
	logic := NewQueryServerNodeTokenLogic(context.Background(), &svc.ServiceContext{
		Config:    config.Config{Node: config.NodeConfig{NodeSecret: masterSecret}},
		NodeModel: modelNode.NewModel(db, nil),
	})

	resp, err := logic.QueryServerNodeToken(&types.QueryServerNodeTokenRequest{Id: 7})
	if err != nil {
		t.Fatal(err)
	}
	want, err := nodeauth.DeriveToken(masterSecret, 7)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != want {
		t.Fatalf("token = %q, want derived token", resp.Token)
	}
	if _, err := logic.QueryServerNodeToken(&types.QueryServerNodeTokenRequest{Id: 8}); err == nil {
		t.Fatal("missing server should not receive a token")
	}
}
