package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/neibniz/alpha/server/pkg/logger"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}
type Client struct {
	*oauth2.Config
}
type UserInfo struct {
	OpenID        string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

func New(config *Config) *Client {
	return &Client{
		&oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Scopes:       []string{"openid", "profile", "email", "https://www.googleapis.com/auth/user.phonenumbers.read"},
			Endpoint:     google.Endpoint,
		},
	}
}
func (c *Client) GetUserInfo(token string) (*UserInfo, error) {
	client := c.Config.Client(context.Background(), &oauth2.Token{AccessToken: token})
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		logger.Error("[Google OAuth 2.0] Get User Info", logger.Field("error", err.Error()))
		return nil, err
	}
	defer resp.Body.Close()

	userInfo, err := decodeUserInfo(resp.Body)
	if err != nil {
		logger.Error("[Google OAuth 2.0] Decode User Info", logger.Field("error", err.Error()))
		return nil, err
	}

	return userInfo, nil
}

func decodeUserInfo(reader io.Reader) (*UserInfo, error) {
	var userInfo UserInfo
	if err := json.NewDecoder(reader).Decode(&userInfo); err != nil {
		return nil, err
	}
	if userInfo.OpenID == "" {
		return nil, fmt.Errorf("google user info missing id")
	}
	return &userInfo, nil
}
