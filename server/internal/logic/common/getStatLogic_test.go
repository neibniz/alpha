package common

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func boolPtr(value bool) *bool {
	return &value
}

func TestQueryCommonStatCountsUsesSingleAggregatedQuery(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&user.User{}, &node.Node{}, &node.Server{}); err != nil {
		t.Fatalf("migrate stat models: %v", err)
	}

	users := []user.User{
		{Id: 1, Password: "x", Enable: boolPtr(true)},
		{Id: 2, Password: "x", Enable: boolPtr(true)},
		{Id: 3, Password: "x", Enable: boolPtr(false)},
		{Id: 4, Password: "x", Enable: boolPtr(true)},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&users).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := db.Delete(&users[3]).Error; err != nil {
		t.Fatalf("soft delete user: %v", err)
	}

	nodes := []node.Node{
		{Id: 1, Protocol: "vless", Enabled: boolPtr(true), Sort: 1},
		{Id: 2, Protocol: "vmess", Enabled: boolPtr(true), Sort: 2},
		{Id: 3, Protocol: "trojan", Enabled: boolPtr(false), Sort: 3},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&nodes).Error; err != nil {
		t.Fatalf("create nodes: %v", err)
	}

	servers := []node.Server{
		{Id: 1, Country: "US", Sort: 1},
		{Id: 2, Country: "US", Sort: 2},
		{Id: 3, Country: "JP", Sort: 3},
		{Id: 4, Country: "", Sort: 4},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&servers).Error; err != nil {
		t.Fatalf("create servers: %v", err)
	}

	counts, err := queryCommonStatCounts(context.Background(), db)
	if err != nil {
		t.Fatalf("query common stat counts: %v", err)
	}
	if counts.UserCount != 2 || counts.NodeCount != 2 || counts.CountryCount != 2 {
		t.Fatalf("counts = %#v", counts)
	}
}

func TestRoundedPublicUserCount(t *testing.T) {
	tests := map[int64]int64{
		0:   1,
		5:   1,
		10:  1,
		11:  10,
		105: 100,
		250: 200,
	}
	for input, want := range tests {
		if got := roundedPublicUserCount(input); got != want {
			t.Fatalf("roundedPublicUserCount(%d) = %d, want %d", input, got, want)
		}
	}
}

func TestQueryEnabledProtocolsUsesDatabaseDistinct(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&node.Node{}); err != nil {
		t.Fatalf("migrate nodes: %v", err)
	}

	enabled := true
	disabled := false
	rows := []node.Node{
		{Id: 1, Protocol: "vless", InboundAlias: "", Enabled: &enabled, Sort: 1},
		{Id: 2, Protocol: "vless", InboundAlias: "", Enabled: &enabled, Sort: 2},
		{Id: 3, Protocol: "vmess", InboundAlias: "shared-inbound", Enabled: &enabled, Sort: 3},
		{Id: 4, Protocol: "trojan", InboundAlias: "shared-inbound", Enabled: &enabled, Sort: 4},
		{Id: 5, Protocol: "ss", InboundAlias: "", Enabled: &disabled, Sort: 5},
		{Id: 6, Protocol: "", InboundAlias: "", Enabled: &enabled, Sort: 6},
	}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&rows).Error; err != nil {
		t.Fatalf("create nodes: %v", err)
	}

	protocols, err := queryEnabledProtocols(context.Background(), db)
	if err != nil {
		t.Fatalf("query enabled protocols: %v", err)
	}

	got := make(map[string]int, len(protocols))
	for _, protocol := range protocols {
		got[protocol]++
	}
	if len(got) != 2 || got["vless"] != 1 || got["shared-inbound"] != 1 {
		t.Fatalf("protocols = %#v", protocols)
	}
}

func TestQueryEnabledProtocolsKeepsEmptyArrayShape(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&node.Node{}); err != nil {
		t.Fatalf("migrate nodes: %v", err)
	}

	protocols, err := queryEnabledProtocols(context.Background(), db)
	if err != nil {
		t.Fatalf("query enabled protocols: %v", err)
	}
	if protocols == nil || len(protocols) != 0 {
		t.Fatalf("empty protocols should be a non-nil empty slice: %#v", protocols)
	}

	payload, err := json.Marshal(types.GetStatResponse{Protocol: protocols})
	if err != nil {
		t.Fatalf("marshal stat response: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal stat response: %v", err)
	}
	list, ok := got["protocol"].([]any)
	if !ok || len(list) != 0 {
		t.Fatalf("stat protocol should marshal as empty array: %s", payload)
	}
}

func TestNormalizeStatResponseKeepsCachedProtocolArrayShape(t *testing.T) {
	resp := &types.GetStatResponse{}
	normalizeStatResponse(resp)
	if resp.Protocol == nil || len(resp.Protocol) != 0 {
		t.Fatalf("cached empty stat protocol should be non-nil empty slice: %#v", resp.Protocol)
	}

	payload, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal normalized stat response: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal normalized stat response: %v", err)
	}
	list, ok := got["protocol"].([]any)
	if !ok || len(list) != 0 {
		t.Fatalf("normalized cached stat protocol should marshal as empty array: %s", payload)
	}
}
