package types

import (
	"encoding/json"
	"fmt"
)

const (
	// ForthwithSendEmail forthwith send email
	ForthwithSendEmail = "forthwith:email:send"
)

const (
	EmailTypeVerify        = "verify"
	EmailTypeMaintenance   = "maintenance"
	EmailTypeExpiration    = "expiration"
	EmailTypeTrafficExceed = "traffic_exceed"
	EmailTypeCustom        = "custom"
)

type (
	SendEmailPayload struct {
		Type    string          `json:"type"`
		Email   string          `json:"to"`
		Subject string          `json:"subject"`
		Content json.RawMessage `json:"content"`
	}

	VerifyEmailContent struct {
		Type     uint8  `json:"Type"`
		SiteLogo string `json:"SiteLogo"`
		SiteName string `json:"SiteName"`
		Expire   int64  `json:"Expire"`
		Code     string `json:"Code"`
	}

	ExpirationEmailContent struct {
		SiteLogo   string `json:"SiteLogo"`
		SiteName   string `json:"SiteName"`
		ExpireDate string `json:"ExpireDate"`
	}

	SiteEmailContent struct {
		SiteLogo string `json:"SiteLogo"`
		SiteName string `json:"SiteName"`
	}

	CustomEmailContent struct {
		Content string `json:"content"`
	}
)

func EncodeEmailContent[T any](content T) (json.RawMessage, error) {
	data, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func DecodeEmailContent[T any](raw json.RawMessage) (T, error) {
	var content T
	if len(raw) == 0 {
		return content, fmt.Errorf("email content is empty")
	}
	if err := json.Unmarshal(raw, &content); err != nil {
		return content, err
	}
	return content, nil
}
