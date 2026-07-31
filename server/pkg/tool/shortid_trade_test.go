package tool

import "testing"

func TestGenerateShortIDUsesFirstEightSHA256HexChars(t *testing.T) {
	if got, want := GenerateShortID("alpha"), "8ed3f6ad"; got != want {
		t.Fatalf("GenerateShortID() = %q, want %q", got, want)
	}
}

func TestGenerateTradeNoKeepsNumericShape(t *testing.T) {
	got, err := GenerateTradeNo()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 27 {
		t.Fatalf("GenerateTradeNo() length = %d, want 27: %q", len(got), got)
	}
	for _, ch := range got {
		if ch < '0' || ch > '9' {
			t.Fatalf("GenerateTradeNo() = %q, want numeric only", got)
		}
	}
}

func TestGenerateTradeNoUsesIndependentEntropy(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for index := 0; index < 1000; index++ {
		value, err := GenerateTradeNo()
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := seen[value]; exists {
			t.Fatalf("duplicate trade number: %s", value)
		}
		seen[value] = struct{}{}
	}
}
