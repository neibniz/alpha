package queryparams

import (
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestFirstValuesReturnsNilForEmptyQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/subscribe", nil)
	if got := FirstValues(req); got != nil {
		t.Fatalf("FirstValues() = %#v, want nil", got)
	}
}

func TestFirstValuesUsesFirstQueryValue(t *testing.T) {
	req := httptest.NewRequest("GET", "/subscribe?flag=clash&flag=singbox&type=&preset=mobile", nil)
	got := FirstValues(req)
	want := map[string]string{
		"flag":   "clash",
		"type":   "",
		"preset": "mobile",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FirstValues() = %#v, want %#v", got, want)
	}
}

func TestParseFirstValues(t *testing.T) {
	got := ParseFirstValues("pid=1000&name=Alpha+Plan&name=Beta&sign=")
	want := map[string]string{
		"pid":  "1000",
		"name": "Alpha Plan",
		"sign": "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseFirstValues() = %#v, want %#v", got, want)
	}
}

func TestParseFirstValuesReturnsNilForInvalidQuery(t *testing.T) {
	if got := ParseFirstValues("bad=%zz"); got != nil {
		t.Fatalf("ParseFirstValues(invalid) = %#v, want nil", got)
	}
}

func TestFromValuesReturnsNilWhenAllKeysHaveNoValues(t *testing.T) {
	values := url.Values{"empty": nil}
	if got := FromValues(values); got != nil {
		t.Fatalf("FromValues() = %#v, want nil", got)
	}
}

func TestFromValuesAllEmptyValuesAvoidsMapAllocation(t *testing.T) {
	values := url.Values{
		"empty": nil,
		"blank": []string{},
	}
	allocs := testing.AllocsPerRun(1000, func() {
		if got := FromValues(values); got != nil {
			t.Fatalf("FromValues() = %#v, want nil", got)
		}
	})
	if allocs != 0 {
		t.Fatalf("FromValues() allocations = %v, want 0", allocs)
	}
}
