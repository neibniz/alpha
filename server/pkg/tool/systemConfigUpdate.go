package tool

import (
	"reflect"
	"strings"

	"github.com/neibniz/alpha/server/internal/model/system"
	"gorm.io/gorm"
)

type SystemConfigUpdateValue struct {
	Key   string
	Value string
}

const smallSystemConfigUpdateIndexThreshold = 8

func SystemConfigUpdateValuesFromStruct(data any) []SystemConfigUpdateValue {
	v := reflect.ValueOf(data)
	if !v.IsValid() {
		return []SystemConfigUpdateValue{}
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return []SystemConfigUpdateValue{}
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return []SystemConfigUpdateValue{}
	}

	t := v.Type()
	values := make([]SystemConfigUpdateValue, 0, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		values = append(values, SystemConfigUpdateValue{
			Key:   field.Name,
			Value: ConvertValueToString(v.Field(i)),
		})
	}
	return values
}

func UpdateSystemConfigValues(db *gorm.DB, category string, values []SystemConfigUpdateValue) error {
	values, keys := normalizeSystemConfigUpdateValues(values)
	return updateNormalizedSystemConfigValues(db, category, values, keys)
}

func updateNormalizedSystemConfigValues(db *gorm.DB, category string, values []SystemConfigUpdateValue, keys []string) error {
	if len(values) == 0 {
		return nil
	}

	var caseSQL strings.Builder
	caseSQL.Grow(systemConfigUpdateCaseCapacity(len(values)))
	caseSQL.WriteString("CASE key")
	args := make([]any, 0, len(values)*2)
	for _, value := range values {
		caseSQL.WriteString(" WHEN ? THEN ?")
		args = append(args, value.Key, value.Value)
	}
	caseSQL.WriteString(" ELSE value END")

	return db.Model(&system.System{}).
		Where("category = ? AND key IN ?", category, keys).
		Update("value", gorm.Expr(caseSQL.String(), args...)).Error
}

func systemConfigUpdateCaseCapacity(valueCount int) int {
	if valueCount <= 0 {
		return len("CASE key") + len(" ELSE value END")
	}
	return len("CASE key") + valueCount*len(" WHEN ? THEN ?") + len(" ELSE value END")
}

func UpdateOrCreateSystemConfigValues(db *gorm.DB, category string, values []SystemConfigUpdateValue, defaultType string) error {
	values, keys := normalizeSystemConfigUpdateValues(values)
	if len(values) == 0 {
		return nil
	}

	var existingKeys []string
	if err := db.Model(&system.System{}).
		Where("category = ? AND key IN ?", category, keys).
		Pluck("key", &existingKeys).Error; err != nil {
		return err
	}

	existingKeySet := systemConfigExistingKeySet(existingKeys)

	existingCap := min(len(existingKeys), len(values))
	existingValues := make([]SystemConfigUpdateValue, 0, existingCap)
	existingKeysForUpdate := make([]string, 0, existingCap)
	missingRows := make([]system.System, 0, max(len(values)-len(existingKeys), 0))
	for _, value := range values {
		if systemConfigKeyExists(existingKeys, existingKeySet, value.Key) {
			existingValues = append(existingValues, value)
			existingKeysForUpdate = append(existingKeysForUpdate, value.Key)
			continue
		}
		missingRows = append(missingRows, system.System{
			Category: category,
			Key:      value.Key,
			Value:    value.Value,
			Type:     defaultType,
			Desc:     value.Key,
		})
	}

	if err := updateNormalizedSystemConfigValues(db, category, existingValues, existingKeysForUpdate); err != nil {
		return err
	}
	if len(missingRows) == 0 {
		return nil
	}
	return db.Create(&missingRows).Error
}

func systemConfigExistingKeySet(keys []string) map[string]struct{} {
	if len(keys) <= smallSystemConfigUpdateIndexThreshold {
		return nil
	}
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[key] = struct{}{}
	}
	return keySet
}

func systemConfigKeyExists(keys []string, keySet map[string]struct{}, key string) bool {
	if keySet != nil {
		_, ok := keySet[key]
		return ok
	}
	for _, existingKey := range keys {
		if existingKey == key {
			return true
		}
	}
	return false
}

func normalizeSystemConfigUpdateValues(values []SystemConfigUpdateValue) ([]SystemConfigUpdateValue, []string) {
	if len(values) == 0 {
		return nil, nil
	}
	collector := newSystemConfigUpdateValueCollector(len(values))
	for _, value := range values {
		collector.Add(value)
	}
	if collector.Len() == 0 {
		return nil, nil
	}
	return collector.Values(), collector.Keys()
}

type systemConfigUpdateValueCollector struct {
	values  []SystemConfigUpdateValue
	index   map[string]int
	capHint int
}

func newSystemConfigUpdateValueCollector(capHint int) systemConfigUpdateValueCollector {
	return systemConfigUpdateValueCollector{capHint: capHint}
}

func (c *systemConfigUpdateValueCollector) Add(value SystemConfigUpdateValue) {
	if value.Key == "" {
		return
	}
	if c.index != nil {
		if index, ok := c.index[value.Key]; ok {
			c.values[index].Value = value.Value
			return
		}
		c.index[value.Key] = len(c.values)
		c.values = append(c.values, value)
		return
	}
	for index, existing := range c.values {
		if existing.Key == value.Key {
			c.values[index].Value = value.Value
			return
		}
	}
	if c.values == nil {
		c.values = make([]SystemConfigUpdateValue, 0, systemConfigUpdateInitialCap(c.capHint))
	}
	c.values = append(c.values, value)
	if len(c.values) > smallSystemConfigUpdateIndexThreshold {
		c.upgradeIndex()
	}
}

func (c systemConfigUpdateValueCollector) Len() int {
	return len(c.values)
}

func (c systemConfigUpdateValueCollector) Values() []SystemConfigUpdateValue {
	return c.values
}

func (c systemConfigUpdateValueCollector) Keys() []string {
	keys := make([]string, 0, len(c.values))
	for _, value := range c.values {
		keys = append(keys, value.Key)
	}
	return keys
}

func (c *systemConfigUpdateValueCollector) upgradeIndex() {
	if c.index != nil {
		return
	}
	nextCap := systemConfigUpdateLookupCap(len(c.values), c.capHint)
	if nextCap > cap(c.values) {
		values := make([]SystemConfigUpdateValue, len(c.values), nextCap)
		copy(values, c.values)
		c.values = values
	}
	c.index = make(map[string]int, nextCap)
	for index, value := range c.values {
		c.index[value.Key] = index
	}
}

func systemConfigUpdateInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallSystemConfigUpdateIndexThreshold+1 {
		return smallSystemConfigUpdateIndexThreshold + 1
	}
	return capHint
}

func systemConfigUpdateLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}
