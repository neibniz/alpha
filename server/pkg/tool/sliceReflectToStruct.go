package tool

import (
	"reflect"
	"strconv"
	"sync"

	"github.com/neibniz/alpha/server/internal/model/system"
	"github.com/goccy/go-json"
)

var systemConfigFieldCache = struct {
	sync.RWMutex
	fields map[reflect.Type]map[string][]int
}{
	fields: make(map[reflect.Type]map[string][]int),
}

func SystemConfigSliceReflectToStruct(slice []*system.System, structType any) {
	v := reflect.ValueOf(structType)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return
	}

	fields := cachedSystemConfigFields(v.Type())
	for _, config := range slice {
		index, ok := fields[config.Key]
		if !ok {
			continue
		}
		setSystemConfigField(v.FieldByIndex(index), config)
	}
}

func cachedSystemConfigFields(t reflect.Type) map[string][]int {
	systemConfigFieldCache.RLock()
	fields, ok := systemConfigFieldCache.fields[t]
	systemConfigFieldCache.RUnlock()
	if ok {
		return fields
	}

	fields = make(map[string][]int, t.NumField())
	for _, field := range reflect.VisibleFields(t) {
		if field.PkgPath != "" {
			continue
		}
		fields[field.Name] = field.Index
	}

	systemConfigFieldCache.Lock()
	if cached, ok := systemConfigFieldCache.fields[t]; ok {
		systemConfigFieldCache.Unlock()
		return cached
	}
	systemConfigFieldCache.fields[t] = fields
	systemConfigFieldCache.Unlock()
	return fields
}

func setSystemConfigField(field reflect.Value, config *system.System) {
	if !field.IsValid() || !field.CanSet() {
		return
	}

	if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Bool {
		if config.Value == "" {
			field.Set(reflect.Zero(field.Type()))
			return
		}
		boolValue, err := strconv.ParseBool(config.Value)
		if err != nil {
			return
		}
		field.Set(reflect.ValueOf(&boolValue))
		return
	}

	switch config.Type {
	case "string":
		if field.Kind() == reflect.String {
			field.SetString(config.Value)
		}
	case "bool":
		if field.Kind() == reflect.Bool {
			boolValue, err := strconv.ParseBool(config.Value)
			if err == nil {
				field.SetBool(boolValue)
			}
		}
	case "int":
		setSystemConfigIntField(field, config.Value, 0)
	case "int64":
		setSystemConfigIntField(field, config.Value, 64)
	case "interface":
		if field.CanAddr() {
			_ = json.Unmarshal([]byte(config.Value), field.Addr().Interface())
		}
	}
}

func setSystemConfigIntField(field reflect.Value, value string, bitSize int) {
	if field.Kind() < reflect.Int || field.Kind() > reflect.Int64 {
		return
	}
	intValue, err := strconv.ParseInt(value, 10, bitSize)
	if err != nil || field.OverflowInt(intValue) {
		return
	}
	field.SetInt(intValue)
}
