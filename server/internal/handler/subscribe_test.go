package handler

import (
	"reflect"
	"testing"

	"github.com/neibniz/alpha/server/internal/subscribeua"
)

func TestNormalizeSubscribeUserAgentKeywords(t *testing.T) {
	got := subscribeua.NormalizeKeywords(" Clash \n\nsing-box\nCLASH\n", []string{
		"Shadowrocket",
		" sing-box ",
		"",
	})
	want := []string{"clash", "sing-box", "shadowrocket"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("keywords = %#v, want %#v", got, want)
	}
}
