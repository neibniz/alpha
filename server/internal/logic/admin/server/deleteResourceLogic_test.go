package server

import (
	"context"
	"errors"
	"testing"

	modelNode "github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func deleteResourceTestContext(t *testing.T) (*gorm.DB, *svc.ServiceContext) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&modelNode.Server{},
		&modelNode.Node{},
		&modelNode.XrayTemplate{},
		&modelNode.ServerXrayTemplate{},
	); err != nil {
		t.Fatal(err)
	}
	return db, &svc.ServiceContext{NodeModel: modelNode.NewModel(db, nil)}
}

func requireCodeError(t *testing.T, err error, code uint32) {
	t.Helper()
	var codeErr *xerr.CodeError
	if !errors.As(err, &codeErr) || codeErr.GetErrCode() != code {
		t.Fatalf("error = %v, want code %d", err, code)
	}
}

func TestDeleteServerRejectsNodeAndTemplateReferences(t *testing.T) {
	db, service := deleteResourceTestContext(t)
	ctx := context.Background()
	enabled := true
	server := &modelNode.Server{Id: 7, Name: "server", Address: "192.0.2.7"}
	if err := db.Create(server).Error; err != nil {
		t.Fatal(err)
	}
	node := &modelNode.Node{
		Id: 8, Name: "node", Address: "node.example.com", Port: 443,
		ServerId: server.Id, Protocol: "vless", InboundAlias: "vless", Enabled: &enabled,
	}
	if err := db.Create(node).Error; err != nil {
		t.Fatal(err)
	}
	logic := NewDeleteServerLogic(ctx, service)
	requireCodeError(t, logic.DeleteServer(&types.DeleteServerRequest{Id: server.Id}), xerr.ServerReferenced)
	if _, err := service.NodeModel.FindOneServer(ctx, server.Id); err != nil {
		t.Fatalf("referenced server was deleted: %v", err)
	}
	if err := service.NodeModel.DeleteNode(ctx, node.Id); err != nil {
		t.Fatal(err)
	}
	template := &modelNode.XrayTemplate{
		Id: 9, Name: "inbound", Type: modelNode.XrayTemplateInbound,
		Config: "{}", VariablesSchema: "{}", DefaultVariables: "{}", SubscriptionMeta: "{}", Enabled: &enabled,
	}
	if err := db.Create(template).Error; err != nil {
		t.Fatal(err)
	}
	binding := &modelNode.ServerXrayTemplate{
		ServerId: server.Id, TemplateId: template.Id, Alias: "vless",
		Variables: "{}", SubscriptionVariables: "{}", Enabled: &enabled, SubscriptionEnabled: &enabled,
	}
	if err := db.Create(binding).Error; err != nil {
		t.Fatal(err)
	}
	requireCodeError(t, logic.DeleteServer(&types.DeleteServerRequest{Id: server.Id}), xerr.ServerReferenced)
	if err := db.Delete(binding).Error; err != nil {
		t.Fatal(err)
	}
	if err := logic.DeleteServer(&types.DeleteServerRequest{Id: server.Id}); err != nil {
		t.Fatalf("unreferenced server delete failed: %v", err)
	}
	if _, err := service.NodeModel.FindOneServer(ctx, server.Id); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("server still exists after delete: %v", err)
	}
}

func TestDeleteMissingResourcesReturnStableErrors(t *testing.T) {
	_, service := deleteResourceTestContext(t)
	ctx := context.Background()
	requireCodeError(
		t,
		NewDeleteServerLogic(ctx, service).DeleteServer(&types.DeleteServerRequest{Id: 404}),
		xerr.ServerNotExist,
	)
	requireCodeError(
		t,
		NewDeleteNodeLogic(ctx, service).DeleteNode(&types.DeleteNodeRequest{Id: 404}),
		xerr.NodeNotExist,
	)
}
