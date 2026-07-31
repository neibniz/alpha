package subscribeua

import (
	"reflect"
	"testing"
)

func TestNormalizeKeywordsTrimsLowercasesAndDeduplicates(t *testing.T) {
	got := NormalizeKeywords(" Clash \n\nsing-box\nCLASH\n", []string{
		"Shadowrocket",
		" sing-box ",
		"",
	})
	want := []string{"clash", "sing-box", "shadowrocket"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeKeywords() = %#v, want %#v", got, want)
	}
}

func TestNormalizeKeywordsBoundsSparseCapacity(t *testing.T) {
	dbValues := make([]string, 0, smallKeywordUniqueThreshold+40)
	for i := 0; i <= smallKeywordUniqueThreshold; i++ {
		dbValues = append(dbValues, "kw-"+string(rune('a'+i)))
	}
	for len(dbValues) < smallKeywordUniqueThreshold+40 {
		dbValues = append(dbValues, "kw-a")
	}

	got := NormalizeKeywords("", dbValues)
	if len(got) != smallKeywordUniqueThreshold+1 {
		t.Fatalf("NormalizeKeywords() len = %d, want %d", len(got), smallKeywordUniqueThreshold+1)
	}
	if maxCap := len(got) * 2; cap(got) > maxCap {
		t.Fatalf("NormalizeKeywords() cap = %d, want <= %d", cap(got), maxCap)
	}
}

func TestAppendUniqueKeywordUsesLinearDedupForSmallInput(t *testing.T) {
	var keywords []string
	var seen map[string]struct{}
	keywords, seen = appendUniqueKeyword(keywords, "clash", seen)
	keywords, seen = appendUniqueKeyword(keywords, "sing-box", seen)
	keywords, seen = appendUniqueKeyword(keywords, "clash", seen)
	keywords, seen = appendUniqueKeyword(keywords, "", seen)

	if seen != nil {
		t.Fatal("small keyword set should not allocate map")
	}
	if want := []string{"clash", "sing-box"}; !reflect.DeepEqual(keywords, want) {
		t.Fatalf("keywords = %#v, want %#v", keywords, want)
	}
}

func TestAppendUniqueKeywordUpgradesLargeInputToMap(t *testing.T) {
	var keywords []string
	var seen map[string]struct{}
	for _, keyword := range []string{
		"kw-1", "kw-2", "kw-3", "kw-4", "kw-5",
		"kw-6", "kw-7", "kw-8", "kw-9",
	} {
		keywords, seen = appendUniqueKeyword(keywords, keyword, seen)
	}
	keywords, seen = appendUniqueKeyword(keywords, "kw-1", seen)

	if seen == nil {
		t.Fatal("large keyword set should upgrade to map")
	}
	if len(keywords) != smallKeywordUniqueThreshold+1 {
		t.Fatalf("keywords = %#v, want %d unique values", keywords, smallKeywordUniqueThreshold+1)
	}
}

func TestAppendUniqueKeywordUsesCapacityHintForLargeInput(t *testing.T) {
	keywords := make([]string, 0, smallKeywordUniqueThreshold+1)
	var seen map[string]struct{}
	keywords, seen = appendUniqueKeyword(keywords, "clash", seen)
	if seen != nil {
		t.Fatal("large hinted keyword set should not allocate map for one unique keyword")
	}
	keywords, seen = appendUniqueKeyword(keywords, "clash", seen)
	if len(keywords) != 1 || seen != nil {
		t.Fatalf("keywords = %#v seen=%#v, want one unique keyword and no map", keywords, seen)
	}
	for _, keyword := range []string{"kw-2", "kw-3", "kw-4", "kw-5", "kw-6", "kw-7", "kw-8", "kw-9"} {
		keywords, seen = appendUniqueKeyword(keywords, keyword, seen)
	}
	if seen == nil {
		t.Fatal("keyword set should allocate map after actual unique values exceed threshold")
	}
	if len(keywords) != smallKeywordUniqueThreshold+1 {
		t.Fatalf("keywords = %#v, want %d unique values", keywords, smallKeywordUniqueThreshold+1)
	}
}

func TestAppendUniqueKeywordSmallInputAvoidsMapAllocation(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		keywords := make([]string, 0, 3)
		var seen map[string]struct{}
		keywords, seen = appendUniqueKeyword(keywords, "clash", seen)
		keywords, seen = appendUniqueKeyword(keywords, "sing-box", seen)
		keywords, seen = appendUniqueKeyword(keywords, "clash", seen)
		_ = keywords
		_ = seen
	})
	if allocs > 0 {
		t.Fatalf("appendUniqueKeyword small input allocations = %f, want 0", allocs)
	}
}

func TestAllowsUserAgentMatchesNormalizedKeywords(t *testing.T) {
	keywords := []string{"clash", "sing-box"}
	if !AllowsUserAgent("ClashX/1.118", keywords) {
		t.Fatal("AllowsUserAgent() should match case-insensitively")
	}
	if AllowsUserAgent("Mozilla/5.0", keywords) {
		t.Fatal("AllowsUserAgent() should reject unknown user agents")
	}
	if AllowsUserAgent("", keywords) {
		t.Fatal("AllowsUserAgent() should reject empty user agent")
	}
}
