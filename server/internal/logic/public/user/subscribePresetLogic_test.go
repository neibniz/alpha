package user

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/neibniz/alpha/server/internal/model/node"
	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/subscribeprofile"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSubscribePresetUpdateReadColumnsStayFocused(t *testing.T) {
	required := []string{
		"id",
		"user_id",
		"user_subscribe_id",
		"preset_key",
		"enabled",
		"is_default",
		"created_at",
		"updated_at",
	}
	if !reflect.DeepEqual(subscribePresetUpdateReadColumns, required) {
		t.Fatalf("subscribe preset update read columns = %#v, want %#v", subscribePresetUpdateReadColumns, required)
	}
}

func TestUpdateUserSubscribePresetUpdatesOnlyMutableFields(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	if err := db.AutoMigrate(&modelSubscribe.Subscribe{}, &modelUser.Subscribe{}, &modelUser.SubscribePreset{}, &node.Server{}, &node.Node{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	if err := db.Create(&modelSubscribe.Subscribe{Id: 101, Name: "Current Plan", Nodes: "1"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&node.Server{Id: 1, Name: "server", Address: "node.example.com"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&node.Node{
		Id:           1,
		Name:         "node",
		ServerId:     1,
		Protocol:     "vless",
		InboundAlias: "vless",
		Enabled:      &enabled,
		Sort:         1,
	}).Error; err != nil {
		t.Fatal(err)
	}

	snapshot := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:  ordersnapshot.KindSubscribe,
		ID:    101,
		Name:  "Snapshot Plan",
		Nodes: []int64{1},
	})
	if err := db.Create(&modelUser.Subscribe{
		Id:              11,
		UserId:          7,
		SubscribeId:     101,
		ProductSnapshot: snapshot,
	}).Error; err != nil {
		t.Fatal(err)
	}

	createdAt := time.Unix(1700000000, 0).UTC()
	preset := modelUser.SubscribePreset{
		Id:              21,
		UserId:          7,
		UserSubscribeId: 11,
		Name:            "old",
		PresetKey:       "ps_stable",
		Mode:            modelUser.SubscribePresetModeProfile,
		ProfileKey:      "old-profile",
		Items:           "[]",
		Sort:            9,
		Enabled:         &enabled,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}
	if err := db.Create(&preset).Error; err != nil {
		t.Fatal(err)
	}

	currentUser := &modelUser.User{Id: 7}
	ctx := context.WithValue(context.Background(), constant.CtxKeyUser, currentUser)
	disabled := false
	logic := NewUpdateUserSubscribePresetLogic(ctx, &svc.ServiceContext{
		DB:             db,
		UserModel:      modelUser.NewModel(db, rds),
		SubscribeModel: modelSubscribe.NewModel(db, rds),
	})
	resp, err := logic.UpdateUserSubscribePreset(&types.UpdateUserSubscribePresetRequest{
		Id:      21,
		Name:    "New Preset",
		Enabled: &disabled,
		Items: []types.UserSubscribePresetItem{
			{NodeId: 1, InboundAlias: " vless ", DisplayName: "  Home Exit  "},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Preset.Name != "New Preset" || resp.Preset.PresetKey != "ps_stable" || resp.Preset.Mode != modelUser.SubscribePresetModeCustom || resp.Preset.ProfileKey != "" || resp.Preset.Enabled {
		t.Fatalf("response preset mismatch: %#v", resp.Preset)
	}
	if len(resp.Preset.Items) != 1 || resp.Preset.Items[0].InboundAlias != "vless" || resp.Preset.Items[0].DisplayName != "Home Exit" {
		t.Fatalf("response preset items not normalized: %#v", resp.Preset.Items)
	}

	var got modelUser.SubscribePreset
	if err := db.First(&got, 21).Error; err != nil {
		t.Fatal(err)
	}
	if got.PresetKey != "ps_stable" || got.Sort != 9 || got.UserId != 7 || got.UserSubscribeId != 11 || got.CreatedAt.Unix() != createdAt.Unix() {
		t.Fatalf("immutable preset fields changed: %#v", got)
	}
	if got.Name != "New Preset" || got.Mode != modelUser.SubscribePresetModeCustom || got.ProfileKey != "" || got.Enabled == nil || *got.Enabled {
		t.Fatalf("mutable preset fields not updated: %#v", got)
	}
	items, err := subscribeprofile.ParseItems(got.Items)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].NodeId != 1 || items[0].InboundAlias != "vless" || items[0].DisplayName != "Home Exit" {
		t.Fatalf("stored preset items = %#v", items)
	}
}
