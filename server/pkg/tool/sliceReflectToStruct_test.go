package tool

import (
	"reflect"
	"sync"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/system"
)

func TestSystemConfigSliceReflectToStruct(t *testing.T) {
	type config struct {
		Name     string
		Enabled  bool
		Count    int
		Total    int64
		Optional *bool
		Items    []string
	}

	var got config
	SystemConfigSliceReflectToStruct([]*system.System{
		{Key: "Name", Value: "Alpha", Type: "string"},
		{Key: "Enabled", Value: "true", Type: "bool"},
		{Key: "Count", Value: "7", Type: "int"},
		{Key: "Total", Value: "42", Type: "int64"},
		{Key: "Optional", Value: "true", Type: "bool"},
		{Key: "Items", Value: `["a","b"]`, Type: "interface"},
	}, &got)

	if got.Name != "Alpha" || !got.Enabled || got.Count != 7 || got.Total != 42 {
		t.Fatalf("scalar fields not mapped: %+v", got)
	}
	if got.Optional == nil || !*got.Optional {
		t.Fatalf("optional bool not mapped: %+v", got.Optional)
	}
	if !reflect.DeepEqual(got.Items, []string{"a", "b"}) {
		t.Fatalf("interface field not mapped: %#v", got.Items)
	}
}

func TestSystemConfigSliceReflectToStructIgnoresInvalidFields(t *testing.T) {
	type config struct {
		Name  string
		Count int
	}

	got := config{Name: "existing", Count: 3}
	SystemConfigSliceReflectToStruct([]*system.System{
		{Key: "Missing", Value: "ignored", Type: "string"},
		{Key: "Name", Value: "not-bool", Type: "bool"},
		{Key: "Count", Value: "overflow", Type: "int"},
	}, &got)

	if got.Name != "existing" || got.Count != 3 {
		t.Fatalf("invalid fields should be ignored: %+v", got)
	}
}

func TestSystemConfigSliceReflectToStructConcurrentCache(t *testing.T) {
	type config struct {
		Name    string
		Enabled bool
		Count   int
	}

	rows := []*system.System{
		{Key: "Name", Value: "Alpha", Type: "string"},
		{Key: "Enabled", Value: "true", Type: "bool"},
		{Key: "Count", Value: "7", Type: "int"},
	}

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var got config
			SystemConfigSliceReflectToStruct(rows, &got)
			if got.Name != "Alpha" || !got.Enabled || got.Count != 7 {
				t.Errorf("mapped config = %+v", got)
			}
		}()
	}
	wg.Wait()
}
