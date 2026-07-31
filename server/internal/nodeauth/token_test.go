package nodeauth

import "testing"

func TestDerivedTokenIsBoundToServer(t *testing.T) {
	first, err := DeriveToken("0123456789abcdef0123456789abcdef", 1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := DeriveToken("0123456789abcdef0123456789abcdef", 2)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || !VerifyToken("0123456789abcdef0123456789abcdef", 1, first) {
		t.Fatal("derived node token is not correctly bound to server id")
	}
	if VerifyToken("0123456789abcdef0123456789abcdef", 2, first) {
		t.Fatal("server 1 token authenticated as server 2")
	}
}
