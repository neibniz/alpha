package tool

import (
	"errors"
	"testing"
)

func TestEncodePassWord(t *testing.T) {
	hash, err := EncodePassWord("correct-horse-battery-staple")
	if err != nil {
		t.Fatal(err)
	}
	if !MultiPasswordVerify(PasswordAlgo, "", "correct-horse-battery-staple", hash) {
		t.Fatal("bcrypt password verification failed")
	}
	if _, err := EncodePassWord("short"); !errors.Is(err, ErrPasswordLength) {
		t.Fatalf("short password error = %v, want ErrPasswordLength", err)
	}
}

func TestMultiPasswordVerify(t *testing.T) {
	pwd := "$2y$10$WFO17pdtohfeBILjEChoGeVxpDG.u9kVCKhjDAeEeNmCjIlj3tDRy"
	status := MultiPasswordVerify("bcrypt", "", "admin1", pwd)
	t.Logf("MultiPasswordVerify: %v", status)
}

func TestVerifyPassWordRejectsMalformedLegacyHash(t *testing.T) {
	for _, hash := range []string{"", "$pbkdf2-sha512$", "$wrong$salt$hash"} {
		if VerifyPassWord("password", hash) {
			t.Fatalf("malformed legacy hash %q was accepted", hash)
		}
	}
}
