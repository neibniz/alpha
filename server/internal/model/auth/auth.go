package auth

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/neibniz/alpha/server/pkg/email"
)

type Auth struct {
	Id        int64     `gorm:"primaryKey"`
	Method    string    `gorm:"unique;type:varchar(255);not null;default:'';comment:platform"`
	Config    string    `gorm:"type:text;not null;comment:Auth Configuration"`
	Enabled   *bool     `gorm:"type:boolean;not null;default:false;comment:Is Enabled"`
	CreatedAt time.Time `gorm:"<-:create;comment:Create Time"`
	UpdatedAt time.Time `gorm:"comment:Update Time"`
}

func (Auth) TableName() string {
	return "auth_method"
}

var authConfigRawNull = []byte("null")

func marshalAuthString[T any](value *T) string {
	data, err := json.Marshal(value)
	if err != nil {
		data, _ = json.Marshal(new(T))
	}
	return string(data)
}

func isEmptyOrNullRaw(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	return bytes.Equal(bytes.TrimSpace(raw), authConfigRawNull)
}

type AppleAuthConfig struct {
	TeamID       string `json:"team_id"`
	KeyID        string `json:"key_id"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
}

func (l *AppleAuthConfig) Marshal() string {
	return marshalAuthString(l)
}

func (l *AppleAuthConfig) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), l)
}

type GoogleAuthConfig struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
}

func (l *GoogleAuthConfig) Marshal() string {
	return marshalAuthString(l)
}

func (l *GoogleAuthConfig) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), l)
}

type GithubAuthConfig struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
}

func (l *GithubAuthConfig) Marshal() string {
	return marshalAuthString(l)
}

func (l *GithubAuthConfig) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), l)
}

type FacebookAuthConfig struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
}

func (l *FacebookAuthConfig) Marshal() string {
	return marshalAuthString(l)
}

func (l *FacebookAuthConfig) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), l)
}

type TelegramAuthConfig struct {
	BotToken      string `json:"bot_token"`
	EnableNotify  bool   `json:"enable_notify"`
	WebHookDomain string `json:"webhook_domain"`
}

func (l *TelegramAuthConfig) Marshal() string {
	return marshalAuthString(l)
}

func (l *TelegramAuthConfig) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), l)
}

type EmailAuthConfig struct {
	Platform                   string          `json:"platform"`
	PlatformConfig             json.RawMessage `json:"platform_config"`
	EnableVerify               bool            `json:"enable_verify"`
	EnableNotify               bool            `json:"enable_notify"`
	EnableDomainSuffix         bool            `json:"enable_domain_suffix"`
	DomainSuffixList           string          `json:"domain_suffix_list"`
	VerifyEmailTemplate        string          `json:"verify_email_template"`
	ExpirationEmailTemplate    string          `json:"expiration_email_template"`
	MaintenanceEmailTemplate   string          `json:"maintenance_email_template"`
	TrafficExceedEmailTemplate string          `json:"traffic_exceed_email_template"`
}

func (l *EmailAuthConfig) Marshal() string {
	l.applyDefaults()
	bytes, err := json.Marshal(l)
	if err != nil {
		config := defaultEmailAuthConfig()
		bytes, _ = json.Marshal(config)
	}
	return string(bytes)
}

func (l *EmailAuthConfig) Unmarshal(data string) {
	if err := json.Unmarshal([]byte(data), l); err != nil {
		*l = *defaultEmailAuthConfig()
		return
	}
	l.applyDefaults()
}

func (l *EmailAuthConfig) applyDefaults() {
	if l.Platform == "" {
		l.Platform = "smtp"
	}
	if isEmptyOrNullRaw(l.PlatformConfig) {
		l.PlatformConfig = json.RawMessage(new(SMTPConfig).Marshal())
	}
	if l.ExpirationEmailTemplate == "" {
		l.ExpirationEmailTemplate = email.DefaultExpirationEmailTemplate
	}
	if l.MaintenanceEmailTemplate == "" {
		l.MaintenanceEmailTemplate = email.DefaultMaintenanceEmailTemplate
	}
	if l.TrafficExceedEmailTemplate == "" {
		l.TrafficExceedEmailTemplate = email.DefaultTrafficExceedEmailTemplate
	}
	if l.VerifyEmailTemplate == "" {
		l.VerifyEmailTemplate = email.DefaultEmailVerifyTemplate
	}
}

func defaultEmailAuthConfig() *EmailAuthConfig {
	return &EmailAuthConfig{
		Platform:                   "smtp",
		PlatformConfig:             json.RawMessage(new(SMTPConfig).Marshal()),
		EnableVerify:               true,
		EnableNotify:               true,
		EnableDomainSuffix:         false,
		DomainSuffixList:           "",
		VerifyEmailTemplate:        email.DefaultEmailVerifyTemplate,
		ExpirationEmailTemplate:    email.DefaultExpirationEmailTemplate,
		MaintenanceEmailTemplate:   email.DefaultMaintenanceEmailTemplate,
		TrafficExceedEmailTemplate: email.DefaultTrafficExceedEmailTemplate,
	}
}

// SMTPConfig Email SMTP configuration
type SMTPConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	User string `json:"user"`
	Pass string `json:"pass"`
	From string `json:"from"`
	SSL  bool   `json:"ssl"`
}

func (l *SMTPConfig) Marshal() string {
	return marshalAuthString(l)
}

func (l *SMTPConfig) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), l)
}
