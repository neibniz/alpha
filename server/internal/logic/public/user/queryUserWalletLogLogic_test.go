package user

import (
	"testing"

	modelLog "github.com/neibniz/alpha/server/internal/model/log"
)

func TestWalletLogRowsMergeAssetsAndSignDeductions(t *testing.T) {
	rows := []*modelLog.SystemLog{
		{Id: 3, Type: modelLog.TypeGift.Uint8(), Content: `{"type":342,"amount":2800,"balance":2200,"order_no":"O-1","timestamp":30}`},
		{Id: 2, Type: modelLog.TypeCommission.Uint8(), Content: `{"type":331,"amount":500,"order_no":"O-2","timestamp":20}`},
		{Id: 1, Type: modelLog.TypeBalance.Uint8(), Content: `{"type":323,"amount":1000,"balance":9000,"order_no":"O-3","timestamp":10}`},
	}

	got := walletLogRows(rows)
	if len(got) != 3 {
		t.Fatalf("wallet rows len = %d, want 3", len(got))
	}
	if got[0].Asset != "gift" || got[0].Amount != -2800 || got[0].Balance == nil || *got[0].Balance != 2200 {
		t.Fatalf("gift row = %#v", got[0])
	}
	if got[1].Asset != "commission" || got[1].Amount != 500 || got[1].Balance != nil {
		t.Fatalf("commission row = %#v", got[1])
	}
	if got[2].Asset != "balance" || got[2].Amount != -1000 {
		t.Fatalf("balance row = %#v", got[2])
	}
}

func TestWalletLogRowsSkipMalformedContent(t *testing.T) {
	got := walletLogRows([]*modelLog.SystemLog{{Id: 1, Type: modelLog.TypeGift.Uint8(), Content: "not-json"}, nil})
	if len(got) != 0 {
		t.Fatalf("wallet rows = %#v, want empty", got)
	}
}

func TestWalletLogRowsSignsCommissionPaymentAndRefund(t *testing.T) {
	rows := []*modelLog.SystemLog{
		{Id: 2, Type: modelLog.TypeCommission.Uint8(), Content: `{"type":337,"amount":400,"balance":600,"order_no":"O-4","timestamp":20}`},
		{Id: 1, Type: modelLog.TypeCommission.Uint8(), Content: `{"type":338,"amount":250,"balance":850,"order_no":"O-4","timestamp":10}`},
	}

	got := walletLogRows(rows)
	if len(got) != 2 {
		t.Fatalf("wallet rows len = %d, want 2", len(got))
	}
	if got[0].Amount != -400 || got[0].Balance == nil || *got[0].Balance != 600 {
		t.Fatalf("commission payment row = %#v", got[0])
	}
	if got[1].Amount != 250 || got[1].Balance == nil || *got[1].Balance != 850 {
		t.Fatalf("commission refund row = %#v", got[1])
	}
}
