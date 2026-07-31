package document

import "testing"

func TestDocumentTagsToStringNormalizesTags(t *testing.T) {
	got := documentTagsToString([]string{" alpha ", "", "beta", "alpha", " docs "})
	if want := "alpha,beta,docs"; got != want {
		t.Fatalf("documentTagsToString() = %q, want %q", got, want)
	}
}

func TestDocumentTagsToStringEmpty(t *testing.T) {
	got := documentTagsToString([]string{"", "  "})
	if got != "" {
		t.Fatalf("documentTagsToString() = %q, want empty", got)
	}
}
