package xraytemplate

import "encoding/json"

func JSONToObject(value string) ConfigObject {
	if value == "" {
		return ConfigObject{}
	}
	var data ConfigObject
	if err := json.Unmarshal([]byte(value), &data); err != nil || data == nil {
		return ConfigObject{}
	}
	return data
}

func MergeRouting(items []ConfigObject) ConfigObject {
	if len(items) == 0 {
		return nil
	}
	if len(items) == 1 {
		return items[0]
	}
	merged := ConfigObject{}
	for _, item := range items {
		if item == nil {
			continue
		}
		for key, value := range item {
			if key == "rules" || key == "balancers" {
				merged[key] = AppendUniqueConfigList(merged[key], value)
				continue
			}
			merged[key] = value
		}
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func MergeDNS(items []ConfigObject) ConfigObject {
	if len(items) == 0 {
		return nil
	}
	if len(items) == 1 {
		return items[0]
	}
	merged := ConfigObject{}
	for _, item := range items {
		if item == nil {
			continue
		}
		for key, value := range item {
			switch key {
			case "servers":
				merged[key] = AppendConfigList(merged[key], value)
			case "hosts":
				merged[key] = MergeMap(merged[key], value)
			default:
				merged[key] = value
			}
		}
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func MergeGeodata(items []ConfigObject) ConfigObject {
	if len(items) == 0 {
		return nil
	}
	if len(items) == 1 {
		return items[0]
	}
	merged := ConfigObject{}
	for _, item := range items {
		if item == nil {
			continue
		}
		for key, value := range item {
			if key == "assets" {
				merged[key] = AppendConfigList(merged[key], value)
				continue
			}
			merged[key] = value
		}
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func AppendConfigList(dst any, src any) []any {
	items := appendConfigList(nil, dst)
	return appendConfigList(items, src)
}

func MergeMap(dst any, src any) ConfigObject {
	out := ConfigObject{}
	if data, ok := dst.(ConfigObject); ok {
		for key, value := range data {
			out[key] = value
		}
	}
	if data, ok := src.(ConfigObject); ok {
		for key, value := range data {
			out[key] = value
		}
	}
	return out
}
