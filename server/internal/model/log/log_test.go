package log

import (
	"encoding/json"
	"strings"
	"testing"
	"unsafe"
)

func TestTokenPrefixClonesLongTokenStorage(t *testing.T) {
	token := strings.Repeat("a", 64)
	prefix := TokenPrefix(token)
	if prefix != "aaaaaaaa" {
		t.Fatalf("TokenPrefix() = %q, want first 8 bytes", prefix)
	}
	if unsafe.StringData(prefix) == unsafe.StringData(token) {
		t.Fatal("TokenPrefix should not retain the full token string storage")
	}
}

func TestTokenPrefixKeepsShortTokenShape(t *testing.T) {
	if got := TokenPrefix("short"); got != "short" {
		t.Fatalf("TokenPrefix(short) = %q", got)
	}
	if got := TokenPrefix(""); got != "" {
		t.Fatalf("TokenPrefix(empty) = %q", got)
	}
}

type logContentMarshaler interface {
	Marshal() ([]byte, error)
}

func TestLogContentMarshalShapes(t *testing.T) {
	tests := []struct {
		name  string
		value logContentMarshaler
		want  string
	}{
		{
			name: "message",
			value: &Message{
				To:       "user@example.test",
				Subject:  "Subject",
				Content:  json.RawMessage(`{"code":"123456"}`),
				Platform: "email",
				Template: "verify",
				Status:   1,
			},
			want: `{"to":"user@example.test","subject":"Subject","content":{"code":"123456"},"platform":"email","template":"verify","status":1}`,
		},
		{
			name:  "traffic",
			value: &Traffic{Download: 10, Upload: 20},
			want:  `{"download":10,"upload":20}`,
		},
		{
			name:  "login",
			value: &Login{Method: "password", LoginIP: "203.0.113.1", UserAgent: "UA", Success: true, Timestamp: 100},
			want:  `{"method":"password","login_ip":"203.0.113.1","user_agent":"UA","success":true,"timestamp":100}`,
		},
		{
			name:  "logout",
			value: &Logout{LogoutIP: "203.0.113.2", UserAgent: "UA", Timestamp: 101},
			want:  `{"logout_ip":"203.0.113.2","user_agent":"UA","timestamp":101}`,
		},
		{
			name:  "register",
			value: &Register{AuthMethod: "email", Identifier: "user@example.test", RegisterIP: "203.0.113.3", UserAgent: "UA", Timestamp: 102},
			want:  `{"auth_method":"email","identifier":"user@example.test","register_ip":"203.0.113.3","user_agent":"UA","timestamp":102}`,
		},
		{
			name:  "subscribe",
			value: &Subscribe{Token: "token", UserAgent: "UA", ClientIP: "203.0.113.4", UserSubscribeId: 9},
			want:  `{"token":"token","user_agent":"UA","client_ip":"203.0.113.4","user_subscribe_id":9}`,
		},
		{
			name:  "reset subscribe",
			value: &ResetSubscribe{Type: ResetSubscribeTypeQuota, UserId: 7, OrderNo: "order-1", Timestamp: 103},
			want:  `{"type":234,"user_id":7,"order_no":"order-1","timestamp":103}`,
		},
		{
			name: "adjust subscribe",
			value: &SubscribeAdjustment{
				UserId:         7,
				OperatorUserId: 2,
				Before:         SubscribeAdjustmentState{SubscribeId: 1, Traffic: 100, Download: 80, Upload: 20, ExpireAt: 1000, Status: 2},
				After:          SubscribeAdjustmentState{SubscribeId: 1, Traffic: 200, Download: 0, Upload: 0, ExpireAt: 2000, Status: 1},
				Timestamp:      103,
			},
			want: `{"user_id":7,"operator_user_id":2,"before":{"subscribe_id":1,"traffic":100,"download":80,"upload":20,"expire_at":1000,"status":2},"after":{"subscribe_id":1,"traffic":200,"download":0,"upload":0,"expire_at":2000,"status":1},"timestamp":103}`,
		},
		{
			name:  "balance",
			value: &Balance{Type: BalanceTypeAdjust, Amount: 100, OrderNo: "order-2", Balance: 900, Timestamp: 104},
			want:  `{"type":326,"amount":100,"order_no":"order-2","balance":900,"timestamp":104}`,
		},
		{
			name:  "commission",
			value: &Commission{Type: CommissionTypePurchase, Amount: 20, OrderNo: "order-3", Timestamp: 105},
			want:  `{"type":331,"amount":20,"order_no":"order-3","timestamp":105}`,
		},
		{
			name:  "gift",
			value: &Gift{Type: GiftTypeIncrease, OrderNo: "order-4", SubscribeId: 11, Amount: 30, Balance: 300, Remark: "quota", Timestamp: 106},
			want:  `{"type":341,"order_no":"order-4","subscribe_id":11,"amount":30,"balance":300,"remark":"quota","timestamp":106}`,
		},
		{
			name:  "user traffic",
			value: &UserTraffic{SubscribeId: 12, UserId: 8, Upload: 40, Download: 50, Total: 90},
			want:  `{"subscribe_id":12,"user_id":8,"upload":40,"download":50,"total":90}`,
		},
		{
			name:  "user traffic rank",
			value: &UserTrafficRank{Rank: map[uint8]UserTraffic{1: {SubscribeId: 12, UserId: 8, Upload: 40, Download: 50, Total: 90}}},
			want:  `{"rank":{"1":{"subscribe_id":12,"user_id":8,"upload":40,"download":50,"total":90}}}`,
		},
		{
			name:  "server traffic",
			value: &ServerTraffic{ServerId: 13, Upload: 60, Download: 70, Total: 130},
			want:  `{"server_id":13,"upload":60,"download":70,"total":130}`,
		},
		{
			name:  "server traffic rank",
			value: &ServerTrafficRank{Rank: map[uint8]ServerTraffic{1: {ServerId: 13, Upload: 60, Download: 70, Total: 130}}},
			want:  `{"rank":{"1":{"server_id":13,"upload":60,"download":70,"total":130}}}`,
		},
		{
			name:  "traffic stat",
			value: &TrafficStat{Upload: 80, Download: 90, Total: 170},
			want:  `{"upload":80,"download":90,"total":170}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.value.Marshal()
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("Marshal() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestLogContentMarshalNilReceiverKeepsEmptyObjectShape(t *testing.T) {
	var login *Login
	got, err := login.Marshal()
	if err != nil {
		t.Fatalf("nil Login.Marshal() error = %v", err)
	}
	if string(got) != "{}" {
		t.Fatalf("nil Login.Marshal() = %s, want {}", got)
	}
}

func TestLogContentUnmarshalDirectAlias(t *testing.T) {
	var gift Gift
	if err := gift.Unmarshal([]byte(`{"type":341,"order_no":"order-4","subscribe_id":11,"amount":30,"balance":300,"remark":"quota","timestamp":106}`)); err != nil {
		t.Fatalf("Gift.Unmarshal() error = %v", err)
	}
	if gift.Type != GiftTypeIncrease || gift.OrderNo != "order-4" || gift.SubscribeId != 11 || gift.Amount != 30 || gift.Balance != 300 || gift.Remark != "quota" || gift.Timestamp != 106 {
		t.Fatalf("Gift.Unmarshal() = %#v", gift)
	}
}
