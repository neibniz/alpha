package apple

import (
	"net/http"
	"time"

	"github.com/neibniz/alpha/server/pkg/logger"
)

type Config struct {
	TeamID       string
	ClientID     string
	KeyID        string
	ClientSecret string
	RedirectURI  string
}

// New creates a Client object with the default URLs and a default http client
func New(c Config) (*Client, error) {
	secret, err := GenerateClientSecret(c.ClientSecret, c.TeamID, c.ClientID, c.KeyID)
	if err != nil {
		logger.Errorw("[Apple OAuth] Generate client secret failed", logger.Field("error", err.Error()))
		return nil, err
	}
	return &Client{
		config:        c,
		validationURL: ValidationURL,
		revokeURL:     RevokeURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		secret: secret,
	}, nil
}
