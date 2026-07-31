package subscribe

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	subscribeModel "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestSubscribeItemFromModelMapsAdminResponseShape(t *testing.T) {
	trueValue := true
	createdAt := time.UnixMilli(1700000000123)
	updatedAt := time.UnixMilli(1700000100456)
	item := &subscribeModel.Subscribe{
		Id:                11,
		Name:              "Pro",
		Language:          "en",
		Description:       "plan",
		UnitPrice:         1000,
		UnitTime:          "month",
		Discount:          `[{"quantity":3,"discount":90}]`,
		Replacement:       5,
		Inventory:         100,
		Traffic:           1024,
		SpeedLimit:        200,
		DeviceLimit:       3,
		Quota:             10,
		Nodes:             "101,0,102,101,-1",
		NodeTags:          "asia, premium ",
		Show:              &trueValue,
		UserVisible:       &trueValue,
		Sell:              &trueValue,
		Sort:              7,
		DeductionRatio:    80,
		AllowDeduction:    &trueValue,
		AllowRenewal:      &trueValue,
		AllowResetTraffic: &trueValue,
		ResetCycle:        2,
		RenewalReset:      &trueValue,
		ShowOriginalPrice: true,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}

	got := subscribeItemFromModel(item)
	if got.Id != item.Id || got.Name != item.Name || got.UnitPrice != item.UnitPrice || got.Sort != item.Sort || got.Traffic != item.Traffic || got.SpeedLimit != item.SpeedLimit || got.ResetCycle != item.ResetCycle {
		t.Fatalf("basic fields not mapped: %#v", got)
	}
	if !got.Show || !got.UserVisible || !got.Sell || !got.AllowRenewal {
		t.Fatalf("list boolean fields not mapped: %#v", got)
	}
	if got.CreatedAt != createdAt.UnixMilli() || got.UpdatedAt != updatedAt.UnixMilli() {
		t.Fatalf("timestamps = %d/%d, want %d/%d", got.CreatedAt, got.UpdatedAt, createdAt.UnixMilli(), updatedAt.UnixMilli())
	}
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"description", "discount", "nodes", "node_tags", "deduction_ratio", "allow_deduction", "allow_reset_traffic", "renewal_reset", "show_original_price"} {
		if _, ok := decoded[key]; ok {
			t.Fatalf("admin subscribe list item must not expose %q: %s", key, payload)
		}
	}
}

func TestSubscribeItemFromModelIgnoresDetailOnlyDiscount(t *testing.T) {
	item := &subscribeModel.Subscribe{
		Id:       12,
		Name:     "Broken discount",
		Discount: `not-json`,
	}

	got := subscribeItemFromModel(item)
	if got.Id != item.Id || got.Name != item.Name {
		t.Fatalf("fields should be preserved without parsing discount: %#v", got)
	}
}

func TestSubscribeItemRowsFromModelsSkipsNil(t *testing.T) {
	ids, items := subscribeItemRowsFromModels([]*subscribeModel.Subscribe{
		nil,
		{Id: 11, Name: "Pro", Discount: `[{"quantity":3,"discount":90}]`},
		{Id: 0, Name: "Draft", Discount: `not-json`},
	})
	if !reflect.DeepEqual(ids, []int64{11}) {
		t.Fatalf("ids = %#v, want [11]", ids)
	}
	if len(items) != 2 {
		t.Fatalf("items len = %d, want 2", len(items))
	}
	if items[0].Id != 11 || items[1].Name != "Draft" {
		t.Fatalf("items mismatch: %#v", items)
	}
}

func TestSubscribeItemRowsFromModelsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*subscribeModel.Subscribe, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &subscribeModel.Subscribe{Id: 0, Name: "draft"}
	rows[len(rows)-1] = &subscribeModel.Subscribe{Id: 2, Name: "published"}

	ids, items := subscribeItemRowsFromModels(rows)
	if !reflect.DeepEqual(ids, []int64{2}) {
		t.Fatalf("ids = %#v, want [2]", ids)
	}
	if len(items) != 2 {
		t.Fatalf("items len = %d, want 2", len(items))
	}
	if cap(ids) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy subscribe ids cap = %d, want %d", cap(ids), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(items) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy subscribe items cap = %d, want %d", cap(items), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestSubscribeItemRowsFromModelsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*subscribeModel.Subscribe, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &subscribeModel.Subscribe{Id: int64(index + 1), Name: "subscribe"}
	}

	ids, items := subscribeItemRowsFromModels(rows)
	if len(ids) != viewutil.SmallRowsCapacityThreshold+1 || len(items) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("ids/items len = %d/%d, want %d", len(ids), len(items), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(ids) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("subscribe ids cap = %d, want %d", cap(ids), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(items) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("subscribe items cap = %d, want %d", cap(items), viewutil.SmallRowsCapacityThreshold+1)
	}
	if ids[len(ids)-1] != int64(viewutil.SmallRowsCapacityThreshold+1) || items[len(items)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("subscribe rows order changed: ids=%#v items=%#v", ids, items)
	}
}

func TestSubscribeItemRowsFromModelsKeepsEmptySliceShape(t *testing.T) {
	ids, items := subscribeItemRowsFromModels(nil)
	if ids == nil || len(ids) != 0 {
		t.Fatalf("empty ids should be a non-nil empty slice: %#v", ids)
	}
	if items == nil || len(items) != 0 {
		t.Fatalf("empty items should be a non-nil empty slice: %#v", items)
	}
	payload, err := json.Marshal(struct {
		List []types.SubscribeItem `json:"list"`
	}{List: items})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"list":[]}` {
		t.Fatalf("empty subscribe items marshal = %s, want list []", payload)
	}
}

func TestSubscribeFromModelCanOmitTopology(t *testing.T) {
	item := &subscribeModel.Subscribe{Id: 13, Nodes: "1,2", NodeTags: "a,b"}

	got, err := subscribeFromModel(item, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Nodes) != 0 || len(got.NodeTags) != 0 {
		t.Fatalf("topology should be omitted: nodes=%#v tags=%#v", got.Nodes, got.NodeTags)
	}
	if got.Nodes == nil || got.NodeTags == nil {
		t.Fatalf("empty topology should keep array shape: nodes=%#v tags=%#v", got.Nodes, got.NodeTags)
	}
}

func TestSubscribeFromModelKeepsZeroTimesAtZero(t *testing.T) {
	got, err := subscribeFromModel(&subscribeModel.Subscribe{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.CreatedAt != 0 || got.UpdatedAt != 0 {
		t.Fatalf("zero times should stay zero: %#v", got)
	}
	if got.Discount == nil || got.Nodes == nil || got.NodeTags == nil {
		t.Fatalf("empty arrays should stay non-nil: discount=%#v nodes=%#v node_tags=%#v", got.Discount, got.Nodes, got.NodeTags)
	}
}
