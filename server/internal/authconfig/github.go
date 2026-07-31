package authconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/neibniz/alpha/server/internal/model/auth"
	"github.com/neibniz/alpha/server/internal/oauthcallback"
	"github.com/neibniz/alpha/server/internal/privacy"
)

const gitHubClientSecretScope = "auth_method.github.client_secret" // #nosec G101 -- encryption scope label, not a credential

type GitHubAdminConfig struct {
	ClientID               string `json:"client_id"`
	ClientSecretConfigured bool   `json:"client_secret_configured"`
	RedirectURL            string `json:"redirect_url"`
}

func LoadGitHub(raw string, codec *privacy.Codec) (auth.GithubAuthConfig, error) {
	config, err := ParseGitHub(raw)
	if err != nil {
		return config, err
	}
	if codec != nil {
		secret, err := codec.Open(gitHubClientSecretScope, config.ClientSecret)
		if err != nil {
			return config, fmt.Errorf("decrypt github client secret: %w", err)
		}
		config.ClientSecret = secret
	}
	return config, nil
}

func ParseGitHub(raw string) (auth.GithubAuthConfig, error) {
	var config auth.GithubAuthConfig
	if strings.TrimSpace(raw) == "" {
		raw = "{}"
	}
	if err := config.Unmarshal(raw); err != nil {
		return config, fmt.Errorf("decode github oauth config: %w", err)
	}
	config.ClientId = strings.TrimSpace(config.ClientId)
	config.ClientSecret = strings.TrimSpace(config.ClientSecret)
	config.RedirectURL = strings.TrimSpace(config.RedirectURL)
	return config, nil
}

func StoreGitHub(config auth.GithubAuthConfig, codec *privacy.Codec) (string, error) {
	config.ClientId = strings.TrimSpace(config.ClientId)
	config.ClientSecret = strings.TrimSpace(config.ClientSecret)
	config.RedirectURL = strings.TrimSpace(config.RedirectURL)
	if codec != nil {
		secret, err := codec.Seal(gitHubClientSecretScope, config.ClientSecret)
		if err != nil {
			return "", fmt.Errorf("encrypt github client secret: %w", err)
		}
		config.ClientSecret = secret
	}
	return config.Marshal(), nil
}

func ValidateGitHub(config auth.GithubAuthConfig) error {
	if strings.TrimSpace(config.ClientId) == "" {
		return errors.New("github client id is required")
	}
	if strings.TrimSpace(config.ClientSecret) == "" {
		return errors.New("github client secret is required")
	}
	callback, err := oauthcallback.ValidateBaseURL(config.RedirectURL)
	if err != nil {
		return fmt.Errorf("invalid github callback URL: %w", err)
	}
	if callback.Path != "" && path.Clean(callback.Path) != "/" {
		return errors.New("github callback URL must use the user frontend origin root so login and account binding callbacks are both allowed")
	}
	return nil
}

func RedactGitHub(raw string) (json.RawMessage, error) {
	var config auth.GithubAuthConfig
	if strings.TrimSpace(raw) == "" {
		raw = "{}"
	}
	if err := config.Unmarshal(raw); err != nil {
		return nil, fmt.Errorf("decode github oauth config: %w", err)
	}
	view := GitHubAdminConfig{
		ClientID:               strings.TrimSpace(config.ClientId),
		ClientSecretConfigured: strings.TrimSpace(config.ClientSecret) != "",
		RedirectURL:            strings.TrimSpace(config.RedirectURL),
	}
	payload, err := json.Marshal(view)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}
