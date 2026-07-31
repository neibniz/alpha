package oauthcallback

import (
	"encoding/json"
	"fmt"
)

type StateCode struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type Telegram struct {
	State        string `json:"state"`
	TgAuthResult string `json:"tgAuthResult"`
}

func NewStateCode(code, state string) (StateCode, error) {
	if code == "" {
		return StateCode{}, fmt.Errorf("callback code is empty")
	}
	if state == "" {
		return StateCode{}, fmt.Errorf("callback state is empty")
	}
	return StateCode{
		Code:  code,
		State: state,
	}, nil
}

func NewTelegram(state, tgAuthResult string) (Telegram, error) {
	if state == "" {
		return Telegram{}, fmt.Errorf("callback state is empty")
	}
	if tgAuthResult == "" {
		return Telegram{}, fmt.Errorf("telegram auth result is empty")
	}
	return Telegram{
		State:        state,
		TgAuthResult: tgAuthResult,
	}, nil
}

func Decode[T any](raw json.RawMessage) (T, error) {
	var dest T
	if len(raw) == 0 {
		return dest, fmt.Errorf("callback is empty")
	}
	if err := json.Unmarshal(raw, &dest); err != nil {
		return dest, err
	}
	return dest, nil
}
