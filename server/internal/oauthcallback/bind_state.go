package oauthcallback

import (
	"encoding/json"
	"errors"
	"strings"
)

type BindState struct {
	UserID   int64  `json:"user_id"`
	Redirect string `json:"redirect"`
}

func EncodeBindState(userID int64, redirect string) (string, error) {
	if userID <= 0 || strings.TrimSpace(redirect) == "" {
		return "", errors.New("invalid oauth bind state")
	}
	data, err := json.Marshal(BindState{UserID: userID, Redirect: redirect})
	return string(data), err
}

func DecodeBindState(raw string) (BindState, error) {
	var state BindState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return BindState{}, errors.New("invalid oauth bind state")
	}
	if state.UserID <= 0 || strings.TrimSpace(state.Redirect) == "" {
		return BindState{}, errors.New("invalid oauth bind state")
	}
	return state, nil
}
