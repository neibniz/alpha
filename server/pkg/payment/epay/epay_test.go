package epay

import (
	"crypto/md5"
	"fmt"
	"testing"
)

func TestEpay(t *testing.T) {
	client := NewClient("", "http://127.0.0.1", "", "")
	order := Order{
		Name:      "测试",
		OrderNo:   "123456789",
		Amount:    1000,
		SignType:  "md5",
		NotifyUrl: "http://127.0.0.1",
		ReturnUrl: "http://127.0.0.1",
	}
	url := client.CreatePayUrl(order)
	t.Logf("PayUrl: %s\n", url)

}

func TestQueryOrderStatus(t *testing.T) {
	t.Skipf("Skip TestQueryOrderStatus test")
	client := NewClient("Pid", "Url", "Key", "Type")
	orderNo := "123456789"
	status := client.QueryOrderStatus(orderNo)
	t.Logf("OrderNo: %s, Status: %v\n", orderNo, status)
}

func TestVerifySign(t *testing.T) {
	t.Skipf("Skip TestVerifySign test")
	params := map[string]string{
		"pid":          "1654",
		"trade_no":     "2024121521150860990",
		"out_trade_no": "202412152115078262977262254",
		"type":         "alipay",
		"name":         "product",
		"money":        "10",
		"trade_status": "TRADE_SUCCESS",
		"sign":         "d3181f18ebdf9821f0ab6ee93faa82d1",
		"sign_type":    "MD5",
	}

	key := "LbTabbB580zWyhXhyyww7wwvy5u8k0wl"
	c := NewClient("Pid", "Url", key, "Type")
	if c.VerifySign(params) {
		t.Logf("Sign verification success!")
	} else {
		t.Error("Sign verification failed!")
	}
}

func TestStructToMapUsesExpectedPaymentShape(t *testing.T) {
	client := NewClient("pid", "https://pay.example", "key", "alipay")
	got := client.structToMap(Order{
		Name:      "product",
		OrderNo:   "order-1",
		Amount:    10,
		NotifyUrl: "https://example.com/notify",
		ReturnUrl: "https://example.com/return",
	})

	if len(got) != epayOrderParamCount {
		t.Fatalf("structToMap() len = %d, want %d", len(got), epayOrderParamCount)
	}
	if got["pid"] != "pid" || got["type"] != "alipay" || got["money"] != "10" || got["out_trade_no"] != "order-1" {
		t.Fatalf("structToMap() = %#v", got)
	}
}

func TestCreateSignSkipsSignatureAndEmptyFields(t *testing.T) {
	client := NewClient("pid", "https://pay.example", "key", "alipay")
	params := map[string]string{
		"pid":          "pid",
		"type":         "alipay",
		"name":         "product",
		"money":        "10",
		"out_trade_no": "order",
		"return_url":   "",
		"sign":         "ignored",
		"sign_type":    "MD5",
	}

	sum := md5.Sum([]byte("money=10&name=product&out_trade_no=order&pid=pid&type=alipaykey"))
	want := fmt.Sprintf("%x", sum)
	if got := client.createSign(params); got != want {
		t.Fatalf("createSign() = %q, want %q", got, want)
	}
}

func TestVerifySignRejectsMissingOrModifiedSignature(t *testing.T) {
	client := NewClient("1001", "https://pay.example.com", "secret", "alipay")
	params := map[string]string{
		"pid":          "1001",
		"out_trade_no": "order-1",
		"money":        "10.00",
	}
	params["sign"] = client.createSign(params)
	if !client.VerifySign(params) {
		t.Fatal("valid signature rejected")
	}
	params["money"] = "100.00"
	if client.VerifySign(params) {
		t.Fatal("modified payment notification accepted")
	}
	delete(params, "sign")
	if client.VerifySign(params) {
		t.Fatal("missing signature accepted")
	}
}
