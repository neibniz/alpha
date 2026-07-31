package coupon

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	couponModel "github.com/neibniz/alpha/server/internal/model/coupon"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestCouponFromCreateRequestMapsExplicitFields(t *testing.T) {
	enabled := true
	req := &types.CreateCouponRequest{
		Name:       "launch",
		Code:       "ALPHA",
		Count:      100,
		Type:       1,
		Discount:   20,
		StartTime:  1700000000,
		ExpireTime: 1700600000,
		UserLimit:  2,
		Subscribe:  []int64{11, 0, 12, 11, -1},
		UsedCount:  3,
		Enable:     &enabled,
	}

	got := couponFromCreateRequest(req)
	if got.Name != req.Name || got.Code != req.Code || got.Count != req.Count || got.Type != req.Type || got.Discount != req.Discount {
		t.Fatalf("basic coupon fields not mapped: %#v", got)
	}
	if got.StartTime != req.StartTime || got.ExpireTime != req.ExpireTime || got.UserLimit != req.UserLimit || got.UsedCount != req.UsedCount {
		t.Fatalf("limit/time fields not mapped: %#v", got)
	}
	if got.Subscribe != "11,12" {
		t.Fatalf("subscribe = %q, want 11,12", got.Subscribe)
	}
	if got.Enable == nil || *got.Enable != enabled {
		t.Fatalf("enable not mapped: %#v", got.Enable)
	}
}

func TestCouponFromUpdateRequestMapsIDAndExplicitFields(t *testing.T) {
	enabled := false
	req := &types.UpdateCouponRequest{
		Id:         99,
		Name:       "renewal",
		Code:       "RENEW",
		Count:      10,
		Type:       2,
		Discount:   500,
		StartTime:  1700000000,
		ExpireTime: 1700600000,
		UserLimit:  1,
		Subscribe:  []int64{7, 7, 0, -1},
		UsedCount:  4,
		Enable:     &enabled,
	}

	got := couponFromUpdateRequest(req)
	if got.Id != req.Id || got.Name != req.Name || got.Code != req.Code || got.Count != req.Count || got.Type != req.Type || got.Discount != req.Discount {
		t.Fatalf("update coupon fields not mapped: %#v", got)
	}
	if got.Subscribe != "7" || got.Enable == nil || *got.Enable != enabled {
		t.Fatalf("update subscribe/enable not mapped: %#v", got)
	}
}

func TestCouponResponseFromModelMapsResponseShape(t *testing.T) {
	createdAt := time.UnixMilli(1700000000123)
	updatedAt := time.UnixMilli(1700000100456)
	enabled := true
	item := &couponModel.Coupon{
		Id:         7,
		Name:       "public",
		Code:       "PUBLIC",
		Count:      25,
		Type:       1,
		Discount:   10,
		StartTime:  1700000000,
		ExpireTime: 1700600000,
		UserLimit:  3,
		Subscribe:  "101,0,102,101,-1",
		UsedCount:  5,
		Enable:     &enabled,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}

	got := couponResponseFromModel(item)
	if got.Id != item.Id || got.Name != item.Name || got.Code != item.Code || got.Count != item.Count || got.Type != item.Type || got.Discount != item.Discount {
		t.Fatalf("response basic fields not mapped: %#v", got)
	}
	if !reflect.DeepEqual(got.Subscribe, []int64{101, 102}) {
		t.Fatalf("subscribe = %#v, want [101 102]", got.Subscribe)
	}
	if !got.Enable {
		t.Fatalf("enable = false, want true")
	}
	if got.CreatedAt != createdAt.UnixMilli() || got.UpdatedAt != updatedAt.UnixMilli() {
		t.Fatalf("timestamps = %d/%d, want %d/%d", got.CreatedAt, got.UpdatedAt, createdAt.UnixMilli(), updatedAt.UnixMilli())
	}
}

func TestCouponRowsFromModelsSkipsNilAndKeepsEmptySlice(t *testing.T) {
	enabled := true
	got := couponRowsFromModels([]*couponModel.Coupon{
		nil,
		{Id: 7, Name: "launch", Subscribe: "9,0,9,10", Enable: &enabled},
	})
	if len(got) != 1 {
		t.Fatalf("coupon rows len = %d, want 1", len(got))
	}
	if got[0].Id != 7 || got[0].Name != "launch" || !reflect.DeepEqual(got[0].Subscribe, []int64{9, 10}) || !got[0].Enable {
		t.Fatalf("coupon row mismatch: %#v", got[0])
	}

	empty := couponRowsFromModels(nil)
	if empty == nil {
		t.Fatal("empty coupon rows must return an empty slice, not nil")
	}
	payload, err := json.Marshal(struct {
		List []types.Coupon `json:"list"`
	}{List: empty})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"list":[]}` {
		t.Fatalf("empty coupon rows marshal = %s, want list []", payload)
	}
}

func TestCouponListRowsFromModelsOmitSubscribeAndTimestamps(t *testing.T) {
	enabled := true
	got := couponListRowsFromModels([]*couponModel.Coupon{
		nil,
		{
			Id:         7,
			Name:       "launch",
			Code:       "LAUNCH",
			Count:      100,
			Type:       1,
			Discount:   20,
			StartTime:  1700000000,
			ExpireTime: 1700600000,
			UserLimit:  2,
			Subscribe:  "9,10",
			UsedCount:  5,
			Enable:     &enabled,
			CreatedAt:  time.UnixMilli(1700000000123),
			UpdatedAt:  time.UnixMilli(1700000100456),
		},
	})
	if len(got) != 1 {
		t.Fatalf("coupon list rows len = %d, want 1", len(got))
	}
	if got[0].Id != 7 || got[0].Name != "launch" || got[0].Code != "LAUNCH" || !got[0].Enable {
		t.Fatalf("coupon list row mismatch: %#v", got[0])
	}
	payload, err := json.Marshal(struct {
		List []types.CouponListItem `json:"list"`
	}{List: got})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string][]map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	row := decoded["list"][0]
	for _, key := range []string{"subscribe", "created_at", "updated_at"} {
		if _, ok := row[key]; ok {
			t.Fatalf("coupon list item must not expose %q: %s", key, payload)
		}
	}
}

func TestCouponRowsFromModelsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*couponModel.Coupon, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &couponModel.Coupon{Id: 1, Name: "first"}
	rows[len(rows)-1] = &couponModel.Coupon{Id: 2, Name: "second"}

	got := couponRowsFromModels(rows)
	if len(got) != 2 {
		t.Fatalf("coupon rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy coupon rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("coupon row order changed: %#v", got)
	}
}

func TestCouponListRowsFromModelsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*couponModel.Coupon, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &couponModel.Coupon{Id: 1, Name: "first"}
	rows[len(rows)-1] = &couponModel.Coupon{Id: 2, Name: "second"}

	got := couponListRowsFromModels(rows)
	if len(got) != 2 {
		t.Fatalf("coupon list rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy coupon list rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("coupon list row order changed: %#v", got)
	}
}

func TestCouponRowsFromModelsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*couponModel.Coupon, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &couponModel.Coupon{Id: int64(index + 1), Name: "coupon"}
	}

	got := couponRowsFromModels(rows)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("coupon rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("coupon rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("coupon row order changed: %#v", got)
	}
}

func TestCouponResponseFromModelDefaultsNilEnable(t *testing.T) {
	got := couponResponseFromModel(&couponModel.Coupon{})
	if got.Enable {
		t.Fatalf("nil enable should map to false")
	}
	if got.CreatedAt != 0 || got.UpdatedAt != 0 {
		t.Fatalf("zero timestamps should be encoded as 0: %#v", got)
	}
}
