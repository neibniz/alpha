package privacy

import (
	"encoding/base64"
	"strings"
	"testing"
)

func testCodec(t *testing.T) *Codec {
	t.Helper()
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	indexKey := base64.StdEncoding.EncodeToString([]byte("abcdef0123456789abcdef0123456789"))
	codec, err := NewCodec(Keyring{
		ActiveKeyID:   "k1",
		EncryptionKey: map[string]string{"k1": key},
		IndexKey:      indexKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	return codec
}

func TestCodecRoundTripUsesRandomAuthenticatedEncryption(t *testing.T) {
	codec := testCodec(t)
	first, err := codec.Seal("user:7:email", "user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	second, err := codec.Seal("user:7:email", "user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if first == second || !strings.HasPrefix(first, envelopePrefix+"k1:") {
		t.Fatalf("envelopes must be versioned and randomized: %q / %q", first, second)
	}
	got, err := codec.Open("user:7:email", first)
	if err != nil || got != "user@example.com" {
		t.Fatalf("Open() = %q, %v", got, err)
	}
	if _, err := codec.Open("user:8:email", first); err == nil {
		t.Fatal("expected AAD mismatch to fail")
	}

	payloadOffset := strings.LastIndexByte(first, ':') + 1
	sealed, err := base64.RawURLEncoding.DecodeString(first[payloadOffset:])
	if err != nil || len(sealed) == 0 {
		t.Fatalf("decode sealed payload for tampering: %v", err)
	}
	sealed[len(sealed)-1] ^= 1
	tampered := first[:payloadOffset] + base64.RawURLEncoding.EncodeToString(sealed)
	if _, err := codec.Open("user:7:email", tampered); err == nil {
		t.Fatal("expected tampered ciphertext to fail")
	}
}

func TestIsSealedRecognizesOnlyPrivacyEnvelopes(t *testing.T) {
	codec := testCodec(t)
	sealed, err := codec.Seal("cache:7", "sensitive")
	if err != nil {
		t.Fatal(err)
	}
	if !IsSealed(sealed) {
		t.Fatalf("IsSealed(%q) = false", sealed)
	}
	for _, value := range []string{"", "sensitive", "enc:v2:k1:value"} {
		if IsSealed(value) {
			t.Fatalf("IsSealed(%q) = true", value)
		}
	}
}

func TestBlindIndexNormalizesEmailAndDoesNotExposePlaintext(t *testing.T) {
	codec := testCodec(t)
	first := codec.BlindIndex("email", " User@Example.COM ")
	second := codec.BlindIndex("email", "user@example.com")
	if first != second || len(first) != 64 || strings.Contains(first, "user") {
		t.Fatalf("unexpected blind index %q / %q", first, second)
	}
}

func TestProtectIdentityProducesMaskedProjection(t *testing.T) {
	codec := testCodec(t)
	ciphertext, blindIndex, display, err := codec.ProtectIdentity(7, "email", "User@Example.com")
	if err != nil {
		t.Fatal(err)
	}
	if ciphertext == "user@example.com" || blindIndex == "user@example.com" || display != "u**r@example.com" {
		t.Fatalf("protected identity = %q / %q / %q", ciphertext, blindIndex, display)
	}
	plaintext, err := codec.OpenIdentity(7, "email", ciphertext)
	if err != nil || plaintext != "user@example.com" {
		t.Fatalf("OpenIdentity() = %q, %v", plaintext, err)
	}
}
