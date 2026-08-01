package oauthcallback

import "testing"

func TestBindStateRoundTrip(t *testing.T) {
	raw, err := EncodeBindState(7, "https://panel.example/bind/github/callback")
	if err != nil {
		t.Fatal(err)
	}
	state, err := DecodeBindState(raw)
	if err != nil {
		t.Fatal(err)
	}
	if state.UserID != 7 || state.Redirect != "https://panel.example/bind/github/callback" {
		t.Fatalf("state = %#v", state)
	}
	if _, err := DecodeBindState(`{"user_id":8}`); err == nil {
		t.Fatal("incomplete state should be rejected")
	}
}
