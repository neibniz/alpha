package payment

import (
	"encoding/json"
	"testing"

	modelPayment "github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestPaymentMethodListItemRowsSkipsNilRowsAndKeepsEmptySlice(t *testing.T) {
	got := paymentMethodListItemRows([]*modelPayment.Payment{
		nil,
		{
			Id:       7,
			Name:     "Stripe",
			Platform: "stripe",
			Token:    "tok",
		},
	}, "https://alpha.example.com", false)

	if len(got) != 1 {
		t.Fatalf("detail rows len = %d, want 1", len(got))
	}
	if got[0].Id != 7 || got[0].NotifyURL != "https://alpha.example.com/v1/notify/stripe/tok" {
		t.Fatalf("unexpected detail row: %#v", got[0])
	}

	empty := paymentMethodListItemRows(nil, "https://alpha.example.com", false)
	if empty == nil {
		t.Fatal("empty detail rows must return an empty slice, not nil")
	}
	payload, err := json.Marshal(struct {
		List []types.PaymentMethodListItem `json:"list"`
	}{List: empty})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"list":[]}` {
		t.Fatalf("empty detail rows marshal = %s, want list []", payload)
	}
}

func TestPaymentMethodListItemRowsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*modelPayment.Payment, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &modelPayment.Payment{Id: 1, Platform: "stripe", Token: "a"}
	rows[len(rows)-1] = &modelPayment.Payment{Id: 2, Platform: "epay", Token: "b"}

	got := paymentMethodListItemRows(rows, "https://alpha.example.com", false)
	if len(got) != 2 {
		t.Fatalf("detail rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy detail rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("detail row order changed: %#v", got)
	}
}

func TestPaymentMethodListItemRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*modelPayment.Payment, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &modelPayment.Payment{Id: int64(index + 1), Platform: "stripe", Token: "tok"}
	}

	got := paymentMethodListItemRows(rows, "https://alpha.example.com", true)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("detail rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("detail rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("detail row order changed: %#v", got)
	}
}

func TestPaymentNotifyURL(t *testing.T) {
	if got := paymentNotifyURL(&modelPayment.Payment{Platform: "balance"}, "alpha.example.com", false); got != "" {
		t.Fatalf("balance notify URL = %q, want empty", got)
	}
	if got := paymentNotifyURL(&modelPayment.Payment{Platform: "stripe", Token: "tok"}, "https://alpha.example.com", true); got != "https://alpha.example.com/api/v1/notify/stripe/tok" {
		t.Fatalf("gateway notify URL = %q", got)
	}
	if got := paymentNotifyURL(&modelPayment.Payment{Platform: "epay", Domain: "https://pay.example.com", Token: "tok"}, "https://alpha.example.com", false); got != "https://pay.example.com/v1/notify/epay/tok" {
		t.Fatalf("domain notify URL = %q", got)
	}
	if got := paymentNotifyURL(&modelPayment.Payment{Platform: "epay", Token: "tok"}, "alpha.example.com", false); got != "" {
		t.Fatalf("invalid site URL should not produce a callback URL: %q", got)
	}
}
