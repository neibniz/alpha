package jsonview

import (
	"encoding/json"
	"strings"
)

var (
	rawNull        = json.RawMessage("null")
	rawEmptyObject = json.RawMessage("{}")
)

func RawValueOrNull(value string) json.RawMessage {
	raw, ok := RawValue(value)
	if !ok {
		return rawNull
	}
	return raw
}

func RawValue(value string) (json.RawMessage, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, false
	}
	raw := []byte(value)
	if !json.Valid(raw) {
		return nil, false
	}
	return json.RawMessage(raw), true
}

func RawObjectOrEmpty(value string) json.RawMessage {
	raw, ok := RawObject(value)
	if !ok {
		return rawEmptyObject
	}
	return raw
}

func RawObject(value string) (json.RawMessage, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value[0] != '{' {
		return nil, false
	}
	raw := []byte(value)
	if !json.Valid(raw) {
		return nil, false
	}
	return json.RawMessage(raw), true
}
