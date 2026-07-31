package node

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFilterServerListIncludesReferenceCounts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Server{}, &Node{}, &XrayTemplate{}, &ServerXrayTemplate{}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	server := &Server{Id: 7, Name: "server", Address: "192.0.2.7"}
	template := &XrayTemplate{
		Id: 8, Name: "inbound", Type: XrayTemplateInbound,
		Config: "{}", VariablesSchema: "{}", DefaultVariables: "{}", SubscriptionMeta: "{}", Enabled: &enabled,
	}
	if err := db.Create(server).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(template).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Node{
		Name: "node", Address: "node.example.com", Port: 443, ServerId: server.Id,
		Protocol: "vless", InboundAlias: "vless", Enabled: &enabled,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&ServerXrayTemplate{
		ServerId: server.Id, TemplateId: template.Id, Alias: "vless",
		Variables: "{}", SubscriptionVariables: "{}", Enabled: &enabled, SubscriptionEnabled: &enabled,
	}).Error; err != nil {
		t.Fatal(err)
	}
	_, rows, err := NewModel(db, nil).FilterServerList(context.Background(), &FilterParams{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].NodeCount != 1 || rows[0].TemplateCount != 1 {
		t.Fatalf("reference counts = %#v, want one node and one template", rows)
	}
}
