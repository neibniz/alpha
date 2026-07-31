package coupon

import (
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/types"
)

func validCreateCouponRequest() *types.CreateCouponRequest {
	start := time.Date(2026, time.July, 12, 0, 0, 0, 0, time.UTC)
	return &types.CreateCouponRequest{
		Name:       " Summer Sale ",
		Code:       " SAVE-20 ",
		Count:      100,
		Type:       1,
		Discount:   20,
		StartTime:  start.Unix(),
		ExpireTime: start.Add(24 * time.Hour).Unix(),
		UserLimit:  1,
		UsedCount:  99,
	}
}

func TestNormalizeCreateCoupon(t *testing.T) {
	req := validCreateCouponRequest()
	if err := normalizeCreateCoupon(req); err != nil {
		t.Fatal(err)
	}
	if req.Name != "Summer Sale" || req.Code != "SAVE-20" || req.UsedCount != 0 {
		t.Fatalf("normalized request = %#v", req)
	}
	if req.StartTime < 100_000_000_000 || req.ExpireTime < 100_000_000_000 {
		t.Fatalf("timestamps were not normalized to milliseconds: %#v", req)
	}
}

func TestNormalizeCouponRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*types.CreateCouponRequest)
	}{
		{name: "blank name", mutate: func(req *types.CreateCouponRequest) { req.Name = " " }},
		{name: "invalid code", mutate: func(req *types.CreateCouponRequest) { req.Code = "bad code" }},
		{name: "invalid type", mutate: func(req *types.CreateCouponRequest) { req.Type = 3 }},
		{name: "percentage over 100", mutate: func(req *types.CreateCouponRequest) { req.Discount = 101 }},
		{name: "expiry before start", mutate: func(req *types.CreateCouponRequest) { req.ExpireTime = req.StartTime }},
		{name: "negative count", mutate: func(req *types.CreateCouponRequest) { req.Count = -1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := validCreateCouponRequest()
			test.mutate(req)
			if err := normalizeCreateCoupon(req); err == nil {
				t.Fatal("normalizeCreateCoupon() error = nil")
			}
		})
	}
}

func TestNormalizeUpdateCouponPreservesUsedCount(t *testing.T) {
	base := validCreateCouponRequest()
	enabled := true
	req := &types.UpdateCouponRequest{
		Id:         7,
		Name:       base.Name,
		Code:       base.Code,
		Count:      10,
		Type:       base.Type,
		Discount:   base.Discount,
		StartTime:  base.StartTime,
		ExpireTime: base.ExpireTime,
		UsedCount:  0,
		Enable:     &enabled,
	}
	if err := normalizeUpdateCoupon(req, 4); err != nil {
		t.Fatal(err)
	}
	if req.UsedCount != 4 {
		t.Fatalf("used count = %d, want 4", req.UsedCount)
	}
}
