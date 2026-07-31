package subscribeua

import "strings"

const smallKeywordUniqueThreshold = 8

func NormalizeKeywords(configValue string, dbValues []string) []string {
	keywords := make([]string, 0, keywordInitialCap(len(dbValues)+1))
	var seen map[string]struct{}
	addKeyword := func(value string) {
		value = strings.ToLower(strings.TrimSpace(value))
		keywords, seen = appendUniqueKeyword(keywords, value, seen)
	}
	for len(configValue) > 0 {
		part := configValue
		if index := strings.IndexByte(configValue, '\n'); index >= 0 {
			part = configValue[:index]
			configValue = configValue[index+1:]
		} else {
			configValue = ""
		}
		addKeyword(part)
	}
	for _, value := range dbValues {
		addKeyword(value)
	}
	return keywords
}

func appendUniqueKeyword(values []string, value string, seen map[string]struct{}) ([]string, map[string]struct{}) {
	if value == "" {
		return values, seen
	}
	if seen != nil {
		if _, ok := seen[value]; ok {
			return values, seen
		}
		seen[value] = struct{}{}
		return append(values, value), seen
	}
	for _, existing := range values {
		if existing == value {
			return values, seen
		}
	}
	values = append(values, value)
	if len(values) > smallKeywordUniqueThreshold {
		capHint := cap(values)
		seen = make(map[string]struct{}, keywordLookupCap(len(values), capHint))
		for _, existing := range values {
			seen[existing] = struct{}{}
		}
	}
	return values, seen
}

func keywordInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallKeywordUniqueThreshold+1 {
		return smallKeywordUniqueThreshold + 1
	}
	return capHint
}

func keywordLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func AllowsUserAgent(userAgent string, keywords []string) bool {
	if userAgent == "" {
		return false
	}
	userAgent = strings.ToLower(userAgent)
	for _, keyword := range keywords {
		if keyword == "" {
			continue
		}
		if strings.Contains(userAgent, keyword) {
			return true
		}
	}
	return false
}
