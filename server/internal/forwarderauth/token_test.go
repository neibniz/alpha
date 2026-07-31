package forwarderauth

import "testing"

func TestDeriveAndVerifyToken(t *testing.T) {
	secret := "forwarder-bootstrap-secret-0123456789abcdef"
	token, err := DeriveToken(secret, "0198a9bc-59dc-7e5f-a3e1-9f225e6ff2b2")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyToken(secret, "0198a9bc-59dc-7e5f-a3e1-9f225e6ff2b2", token) {
		t.Fatal("derived token was rejected")
	}
	if VerifyToken(secret, "different-instance", token) {
		t.Fatal("token must be bound to one instance")
	}
}
