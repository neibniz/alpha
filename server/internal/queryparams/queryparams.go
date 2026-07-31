package queryparams

import (
	"net/http"
	"net/url"
)

func FirstValues(r *http.Request) map[string]string {
	return FromValues(r.URL.Query())
}

func ParseFirstValues(rawQuery string) map[string]string {
	values, _ := url.ParseQuery(rawQuery)
	return FromValues(values)
}

func FromValues(values url.Values) map[string]string {
	if len(values) == 0 {
		return nil
	}

	var result map[string]string
	for key, value := range values {
		if len(value) > 0 {
			if result == nil {
				result = make(map[string]string, len(values))
			}
			result[key] = value[0]
		}
	}
	return result
}
