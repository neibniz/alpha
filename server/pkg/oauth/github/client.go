package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

const (
	defaultAPIBaseURL = "https://api.github.com"
	maxResponseBytes  = 1 << 20
	apiVersion        = "2022-11-28"
)

var ErrVerifiedEmailRequired = errors.New("github account has no verified email address")

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type Client struct {
	*oauth2.Config
	apiBaseURL string
}

type UserInfo struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type emailAddress struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func New(config Config) *Client {
	return &Client{
		Config: &oauth2.Config{
			ClientID:     strings.TrimSpace(config.ClientID),
			ClientSecret: strings.TrimSpace(config.ClientSecret),
			RedirectURL:  strings.TrimSpace(config.RedirectURL),
			Scopes:       []string{"read:user", "user:email"},
			Endpoint:     githuboauth.Endpoint,
		},
		apiBaseURL: defaultAPIBaseURL,
	}
}

func (c *Client) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, errors.New("github access token is empty")
	}

	httpClient := c.Config.Client(ctx, &oauth2.Token{AccessToken: accessToken, TokenType: "Bearer"})
	var user UserInfo
	if err := c.getJSON(ctx, httpClient, "/user", &user); err != nil {
		return nil, fmt.Errorf("get github user: %w", err)
	}
	if user.ID <= 0 {
		return nil, errors.New("github user response is missing id")
	}

	var emails []emailAddress
	if err := c.getJSON(ctx, httpClient, "/user/emails?per_page=100", &emails); err != nil {
		return nil, fmt.Errorf("get github emails: %w", err)
	}
	user.Email = verifiedEmail(emails)
	if user.Email == "" {
		return nil, ErrVerifiedEmailRequired
	}
	return &user, nil
}

func verifiedEmail(emails []emailAddress) string {
	for _, item := range emails {
		if item.Primary && item.Verified && strings.TrimSpace(item.Email) != "" {
			return strings.ToLower(strings.TrimSpace(item.Email))
		}
	}
	for _, item := range emails {
		if item.Verified && strings.TrimSpace(item.Email) != "" {
			return strings.ToLower(strings.TrimSpace(item.Email))
		}
	}
	return ""
}

func (c *Client) getJSON(ctx context.Context, client *http.Client, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.apiBaseURL, "/")+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", apiVersion)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body := io.LimitReader(resp.Body, maxResponseBytes)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		payload, _ := io.ReadAll(body)
		return fmt.Errorf("github API returned %s: %s", resp.Status, strings.TrimSpace(string(payload)))
	}
	if err := json.NewDecoder(body).Decode(dest); err != nil {
		return fmt.Errorf("decode github API response: %w", err)
	}
	return nil
}
