package telegram

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// ParseAuthDataJson parses provided json content for AuthData
func ParseAuthDataJson(content []byte) (*AuthData, error) {
	data := &AuthData{}
	err := json.Unmarshal(content, data)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling error: %w", err)
	}
	return data, nil
}

// ParseAuthDataBase64 decodes provided content from base64 and parses result for AuthData
func ParseAuthDataBase64(content []byte) (*AuthData, error) {

	decodedBytes, err := base64.RawStdEncoding.DecodeString(string(content))
	if err != nil && len(decodedBytes) == 0 {
		return nil, fmt.Errorf("base64 decoding error: %w", err)
	}
	return ParseAuthDataJson(decodedBytes)
}

// ParseAndValidateBase64 parses base64 content for AuthData and validates it
func ParseAndValidateBase64(content []byte, botToken string) (*AuthData, error) {
	authData, err := ParseAuthDataBase64(content)
	if err != nil {
		return nil, err
	}
	err = authData.Validate([]byte(botToken))
	return authData, err
}

// ParseAndValidateJson parses json content for AuthData and validates it
func ParseAndValidateJson(content []byte, botToken []byte) (*AuthData, error) {
	authData, err := ParseAuthDataJson(content)
	if err != nil {
		return nil, err
	}
	err = authData.Validate(botToken)
	return authData, err
}

// GenerateTelegramOAuthURL generates a URL for Telegram OAuth
func GenerateTelegramOAuthURL(botToken, embed, redirect string) string {
	bot := strings.SplitN(botToken, ":", 2)
	if len(bot) == 0 || bot[0] == "" {
		return ""
	}
	parsedURL, err := url.Parse(redirect)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return ""
	}
	query := url.Values{}
	query.Set("bot_id", bot[0])
	query.Set("origin", fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host))
	query.Set("embed", embed)
	query.Set("request_access", "write")
	query.Set("return_to", redirect)
	return (&url.URL{
		Scheme:   "https",
		Host:     "oauth.telegram.org",
		Path:     "/auth",
		RawQuery: query.Encode(),
	}).String()
}
