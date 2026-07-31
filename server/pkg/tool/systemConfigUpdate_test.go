package tool

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/system"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSystemConfigUpdateValuesFromStruct(t *testing.T) {
	type config struct {
		Name     string
		Enabled  bool
		Count    int64
		Optional *bool
		hidden   string
	}

	enabled := true
	got := SystemConfigUpdateValuesFromStruct(&config{
		Name:     "Alpha",
		Enabled:  true,
		Count:    9,
		Optional: &enabled,
		hidden:   "ignore",
	})
	want := []SystemConfigUpdateValue{
		{Key: "Name", Value: "Alpha"},
		{Key: "Enabled", Value: "true"},
		{Key: "Count", Value: "9"},
		{Key: "Optional", Value: "true"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("values = %#v, want %#v", got, want)
	}
}

func TestSystemConfigUpdateValuesFromInvalidInput(t *testing.T) {
	if got := SystemConfigUpdateValuesFromStruct(nil); len(got) != 0 {
		t.Fatalf("nil input values = %#v, want empty", got)
	}
	if got := SystemConfigUpdateValuesFromStruct(7); len(got) != 0 {
		t.Fatalf("non-struct input values = %#v, want empty", got)
	}
}

func TestConvertValueToStringNilPointer(t *testing.T) {
	type config struct {
		Optional *bool
	}
	got := SystemConfigUpdateValuesFromStruct(config{})
	want := []SystemConfigUpdateValue{{Key: "Optional", Value: ""}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("values = %#v, want %#v", got, want)
	}
}

func TestUpdateSystemConfigValuesUpdatesOnlyCategoryKeys(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&system.System{}); err != nil {
		t.Fatal(err)
	}

	rows := []system.System{
		{Category: "site", Key: "Host", Value: "old-host", Type: "string"},
		{Category: "site", Key: "SiteName", Value: "old-name", Type: "string"},
		{Category: "register", Key: "RegisterOnly", Value: "register-value", Type: "string"},
		{Category: "site", Key: "Untouched", Value: "keep", Type: "string"},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	err = UpdateSystemConfigValues(db, "site", []SystemConfigUpdateValue{
		{Key: "Host", Value: "new-host"},
		{Key: "SiteName", Value: "new-name"},
		{Key: "RegisterOnly", Value: "wrong-category"},
		{Key: "Missing", Value: "ignored"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var got []system.System
	if err := db.Order("category, key").Find(&got).Error; err != nil {
		t.Fatal(err)
	}
	values := make(map[string]string, len(got))
	for _, row := range got {
		values[row.Category+"/"+row.Key] = row.Value
	}
	want := map[string]string{
		"register/RegisterOnly": "register-value",
		"site/Host":             "new-host",
		"site/SiteName":         "new-name",
		"site/Untouched":        "keep",
	}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("values = %#v, want %#v", values, want)
	}
}

func TestUpdateOrCreateSystemConfigValuesCreatesMissingAndUpdatesExisting(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&system.System{}); err != nil {
		t.Fatal(err)
	}

	rows := []system.System{
		{Category: "server", Key: "Existing", Value: "old", Type: "string"},
		{Category: "site", Key: "OtherCategory", Value: "keep", Type: "string"},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	err = UpdateOrCreateSystemConfigValues(db, "server", []SystemConfigUpdateValue{
		{Key: "Existing", Value: "new"},
		{Key: "Missing", Value: "created"},
	}, "string")
	if err != nil {
		t.Fatal(err)
	}

	var got []system.System
	if err := db.Order("category, key").Find(&got).Error; err != nil {
		t.Fatal(err)
	}
	values := make(map[string]system.System, len(got))
	for _, row := range got {
		values[row.Category+"/"+row.Key] = row
	}

	if got := values["server/Existing"].Value; got != "new" {
		t.Fatalf("existing value = %q, want new", got)
	}
	missing := values["server/Missing"]
	if missing.Value != "created" || missing.Type != "string" || missing.Desc != "Missing" {
		t.Fatalf("missing row = %#v, want created string row", missing)
	}
	if got := values["site/OtherCategory"].Value; got != "keep" {
		t.Fatalf("other category value = %q, want keep", got)
	}
}

func TestNormalizeSystemConfigUpdateValuesDeduplicatesLastValue(t *testing.T) {
	values, keys := normalizeSystemConfigUpdateValues([]SystemConfigUpdateValue{
		{Key: "", Value: "ignored"},
		{Key: "A", Value: "old"},
		{Key: "B", Value: "two"},
		{Key: "A", Value: "new"},
	})
	wantValues := []SystemConfigUpdateValue{
		{Key: "A", Value: "new"},
		{Key: "B", Value: "two"},
	}
	wantKeys := []string{"A", "B"}
	if !reflect.DeepEqual(values, wantValues) {
		t.Fatalf("values = %#v, want %#v", values, wantValues)
	}
	if !reflect.DeepEqual(keys, wantKeys) {
		t.Fatalf("keys = %#v, want %#v", keys, wantKeys)
	}
}

func TestNormalizeSystemConfigUpdateValuesLargeDuplicateAvoidsMapAllocationAndFullCap(t *testing.T) {
	input := make([]SystemConfigUpdateValue, smallSystemConfigUpdateIndexThreshold+20)
	for index := range input {
		input[index] = SystemConfigUpdateValue{Key: "A", Value: strconv.Itoa(index)}
	}
	values, keys := normalizeSystemConfigUpdateValues(input)
	wantValues := []SystemConfigUpdateValue{{Key: "A", Value: strconv.Itoa(len(input) - 1)}}
	if !reflect.DeepEqual(values, wantValues) {
		t.Fatalf("values = %#v, want %#v", values, wantValues)
	}
	if !reflect.DeepEqual(keys, []string{"A"}) {
		t.Fatalf("keys = %#v, want [A]", keys)
	}
	if cap(values) != smallSystemConfigUpdateIndexThreshold+1 {
		t.Fatalf("values cap = %d, want %d", cap(values), smallSystemConfigUpdateIndexThreshold+1)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_, _ = normalizeSystemConfigUpdateValues(input)
	})
	if allocs > 2 {
		t.Fatalf("normalize duplicate config values allocations = %f, want at most 2", allocs)
	}
}

func TestNormalizeSystemConfigUpdateValuesEmptyKeysAllocFree(t *testing.T) {
	input := make([]SystemConfigUpdateValue, smallSystemConfigUpdateIndexThreshold+20)
	allocs := testing.AllocsPerRun(1000, func() {
		_, _ = normalizeSystemConfigUpdateValues(input)
	})
	if allocs != 0 {
		t.Fatalf("normalize empty-key config values allocations = %f, want 0", allocs)
	}
	values, keys := normalizeSystemConfigUpdateValues(input)
	if values != nil || keys != nil {
		t.Fatalf("empty-key normalize = %#v/%#v, want nil/nil", values, keys)
	}
}

func TestSystemConfigExistingKeySetSmallInputUsesLinearLookup(t *testing.T) {
	keys := []string{"A", "B", "C"}
	keySet := systemConfigExistingKeySet(keys)
	if keySet != nil {
		t.Fatalf("small key set = %#v, want nil map for linear lookup", keySet)
	}
	if !systemConfigKeyExists(keys, keySet, "B") {
		t.Fatal("systemConfigKeyExists() = false, want true")
	}
	if systemConfigKeyExists(keys, keySet, "Missing") {
		t.Fatal("systemConfigKeyExists() = true, want false")
	}
}

func TestSystemConfigExistingKeySetLargeInputBuildsMap(t *testing.T) {
	keys := make([]string, smallSystemConfigUpdateIndexThreshold+1)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
	}
	keySet := systemConfigExistingKeySet(keys)
	if keySet == nil {
		t.Fatal("large key set = nil, want map index")
	}
	if !systemConfigKeyExists(keys, keySet, keys[len(keys)-1]) {
		t.Fatal("systemConfigKeyExists() = false, want true")
	}
	if systemConfigKeyExists(keys, keySet, "Missing") {
		t.Fatal("systemConfigKeyExists() = true, want false")
	}
}

func TestSystemConfigUpdateValueCollectorUpgradesOnlyAfterActualUniqueThreshold(t *testing.T) {
	collector := newSystemConfigUpdateValueCollector(smallSystemConfigUpdateIndexThreshold + 20)
	for index := 0; index < smallSystemConfigUpdateIndexThreshold+20; index++ {
		collector.Add(SystemConfigUpdateValue{Key: "A", Value: strconv.Itoa(index)})
	}
	if collector.index != nil {
		t.Fatal("duplicate-heavy config update collector should not allocate map")
	}
	if len(collector.values) != 1 || cap(collector.values) != smallSystemConfigUpdateIndexThreshold+1 {
		t.Fatalf("collector len/cap = %d/%d, want 1/%d", len(collector.values), cap(collector.values), smallSystemConfigUpdateIndexThreshold+1)
	}
	if got := collector.values[0].Value; got != strconv.Itoa(smallSystemConfigUpdateIndexThreshold+19) {
		t.Fatalf("collector last value = %q, want latest duplicate value", got)
	}

	collector = newSystemConfigUpdateValueCollector(smallSystemConfigUpdateIndexThreshold + 20)
	for index := 0; index <= smallSystemConfigUpdateIndexThreshold; index++ {
		collector.Add(SystemConfigUpdateValue{Key: "K" + strconv.Itoa(index), Value: strconv.Itoa(index)})
	}
	if collector.index == nil {
		t.Fatal("config update collector should allocate map after actual unique keys exceed threshold")
	}
	if len(collector.values) != smallSystemConfigUpdateIndexThreshold+1 {
		t.Fatalf("collector len = %d, want %d", len(collector.values), smallSystemConfigUpdateIndexThreshold+1)
	}
	if maxCap := len(collector.values) * 2; cap(collector.values) > maxCap {
		t.Fatalf("collector cap = %d, want <= %d", cap(collector.values), maxCap)
	}
}

func TestSystemConfigUpdateCaseCapacity(t *testing.T) {
	values := []SystemConfigUpdateValue{
		{Key: "A", Value: "one"},
		{Key: "B", Value: "two"},
	}
	got := systemConfigUpdateCaseCapacity(len(values))
	want := len("CASE key") + len(values)*len(" WHEN ? THEN ?") + len(" ELSE value END")
	if got != want {
		t.Fatalf("systemConfigUpdateCaseCapacity() = %d, want %d", got, want)
	}
}
