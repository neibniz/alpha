package types

import (
	"encoding/base64"
	"strings"
	"testing"

	privacycodec "github.com/neibniz/alpha/server/internal/privacy"
)

func TestPrivatePayloadDoesNotExposeRecipient(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	indexKey := base64.StdEncoding.EncodeToString([]byte("abcdef0123456789abcdef0123456789"))
	codec, err := privacycodec.NewCodec(privacycodec.Keyring{
		ActiveKeyID:   "k1",
		EncryptionKey: map[string]string{"k1": key},
		IndexKey:      indexKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := SendEmailPayload{Type: EmailTypeVerify, Email: "user@example.com"}
	encoded, err := EncodePrivatePayload(codec, EmailPayloadScope, payload)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), payload.Email) {
		t.Fatalf("queue payload exposes recipient: %s", encoded)
	}
	var decoded SendEmailPayload
	if err := DecodePrivatePayload(codec, EmailPayloadScope, encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Email != payload.Email || decoded.Type != payload.Type {
		t.Fatalf("decoded payload = %#v", decoded)
	}
}
