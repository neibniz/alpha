package userview

import (
	"reflect"
	"testing"
)

func TestRulesFromRawNormalizesJSONStringArray(t *testing.T) {
	got := RulesFromRaw(`[" block ", "", "direct", "block"]`)
	want := []string{"block", "direct"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RulesFromRaw() = %#v, want %#v", got, want)
	}
}

func TestRulesFromRawKeepsEmptyArrayShape(t *testing.T) {
	for _, raw := range []string{"", " [] ", "null", " null ", `"invalid"`, `{"rule":"bad"}`} {
		got := RulesFromRaw(raw)
		if got == nil || len(got) != 0 {
			t.Fatalf("RulesFromRaw(%q) = %#v, want non-nil empty slice", raw, got)
		}
	}
}

func TestRulesCacheReusesNormalizedRules(t *testing.T) {
	cache := NewRulesCache(2)
	first := cache.Rules(`[" block ", "", "direct", "block"]`)
	second := cache.Rules(` [" block ", "", "direct", "block"] `)
	want := []string{"block", "direct"}
	if !reflect.DeepEqual(first, want) || !reflect.DeepEqual(second, want) {
		t.Fatalf("cached rules = %#v/%#v, want %#v", first, second, want)
	}
	if len(first) == 0 || &first[0] != &second[0] {
		t.Fatalf("duplicate rules should reuse normalized slice storage")
	}
	if len(cache.rows) != 1 || cache.index != nil {
		t.Fatalf("small rules cache rows/index = %d/%#v, want one row and no map", len(cache.rows), cache.index)
	}
}

func TestRulesCacheCachesInvalidAsEmptyRules(t *testing.T) {
	cache := NewRulesCache(2)
	first := cache.Rules(`{"rule":"bad"}`)
	second := cache.Rules(` {"rule":"bad"} `)
	if first == nil || len(first) != 0 || second == nil || len(second) != 0 {
		t.Fatalf("invalid cached rules = %#v/%#v, want empty non-nil", first, second)
	}
	if len(cache.rows) != 1 {
		t.Fatalf("invalid rules cache rows = %d, want 1", len(cache.rows))
	}
}

func TestRulesCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := NewRulesCache(smallRulesCacheThreshold + 1)
	if cache == nil || cache.rows != nil || cache.index != nil {
		t.Fatalf("large rules cache should not preallocate storage from hint alone: %#v", cache)
	}
	first := cache.Rules(`[" block ", "", "direct", "block"]`)
	second := cache.Rules(` [" block ", "", "direct", "block"] `)
	if !reflect.DeepEqual(first, second) || len(cache.rows) != 1 || cache.index != nil {
		t.Fatalf("duplicate rules should stay on linear path: rows=%#v index=%#v", cache.rows, cache.index)
	}

	cache = NewRulesCache(smallRulesCacheThreshold + 1)
	for i := 0; i <= smallRulesCacheThreshold; i++ {
		cache.Rules(`["rule-` + string(rune('a'+i)) + `"]`)
	}
	if len(cache.rows) != smallRulesCacheThreshold+1 || len(cache.index) != smallRulesCacheThreshold+1 {
		t.Fatalf("rules cache rows/index = %d/%d", len(cache.rows), len(cache.index))
	}
}

func TestRulesCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := NewRulesCache(smallRulesCacheThreshold + 20)
	for i := 0; i <= smallRulesCacheThreshold; i++ {
		cache.Rules(`["rule-` + string(rune('a'+i)) + `"]`)
	}
	if cache.index == nil {
		t.Fatal("rules cache should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("rules cache row cap = %d, want at most %d", cap(cache.rows), maxCap)
	}
	for i := 0; i < smallRulesCacheThreshold+20; i++ {
		cache.Rules(`["rule-a"]`)
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("rules cache row cap after duplicates = %d, want at most %d", cap(cache.rows), maxCap)
	}
}
