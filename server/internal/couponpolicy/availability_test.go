package couponpolicy

import (
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/model/coupon"
)

func TestTimeSupportsUnixSecondsAndMilliseconds(t *testing.T) {
	want := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	if got := Time(want.Unix()); !got.Equal(want) {
		t.Fatalf("seconds time = %v, want %v", got, want)
	}
	if got := Time(want.UnixMilli()); !got.Equal(want) {
		t.Fatalf("milliseconds time = %v, want %v", got, want)
	}
	if got := NormalizeTimestamp(want.Unix()); got != want.UnixMilli() {
		t.Fatalf("normalized timestamp = %d, want %d", got, want.UnixMilli())
	}
}

func TestValidateCouponAvailability(t *testing.T) {
	now := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	enabled := true
	base := coupon.Coupon{
		Type:       1,
		Discount:   20,
		StartTime:  now.Add(-time.Hour).UnixMilli(),
		ExpireTime: now.Add(time.Hour).UnixMilli(),
		Count:      10,
		UsedCount:  2,
		Subscribe:  "7,8",
		Enable:     &enabled,
	}
	if err := Validate(&base, 7, now); err != nil {
		t.Fatalf("valid coupon rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*coupon.Coupon)
	}{
		{name: "disabled", mutate: func(item *coupon.Coupon) { off := false; item.Enable = &off }},
		{name: "not started", mutate: func(item *coupon.Coupon) { item.StartTime = now.Add(time.Minute).UnixMilli() }},
		{name: "expired seconds", mutate: func(item *coupon.Coupon) { item.ExpireTime = now.Add(-time.Minute).Unix() }},
		{name: "usage exhausted", mutate: func(item *coupon.Coupon) { item.UsedCount = item.Count }},
		{name: "invalid percentage", mutate: func(item *coupon.Coupon) { item.Discount = 101 }},
		{name: "wrong subscription", mutate: func(item *coupon.Coupon) { item.Subscribe = "9" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := base
			test.mutate(&item)
			if err := Validate(&item, 7, now); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}
