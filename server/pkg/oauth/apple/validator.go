package apple

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// ValidationURL is the endpoint for verifying tokens
	ValidationURL string = "https://appleid.apple.com/auth/token"
	// RevokeURL is the endpoint for revoking tokens
	RevokeURL string = "https://appleid.apple.com/auth/revoke"
	// ContentType is the one expected by Apple
	ContentType string = "application/x-www-form-urlencoded"
	// UserAgent is required by Apple or the request will fail
	UserAgent string = "go-signin-with-apple"
	// AcceptHeader is the content that we are willing to accept
	AcceptHeader string = "application/json"
)

// ValidationClient is an interface to call the validation API
type ValidationClient interface {
	VerifyWebToken(ctx context.Context, code string) (ValidationResponse, error)
	VerifyAppToken(ctx context.Context, reqBody AppValidationTokenRequest) (ValidationResponse, error)
	VerifyRefreshToken(ctx context.Context, reqBody ValidationRefreshRequest) (RefreshResponse, error)
	RevokeAccessToken(ctx context.Context, token string) (RevokeResponse, error)
	RevokeRefreshToken(ctx context.Context, token string) (RevokeResponse, error)
}

// Client implements ValidationClient
type Client struct {
	config        Config
	validationURL string
	revokeURL     string
	secret        string
	client        *http.Client
}

// ClientOptions is a struct to hold the options for the client
type ClientOptions struct {
	validationURL string
	revokeURL     string
	//nolint:unused
	secret string
	client *http.Client
}

// NewWithURL creates a Client object with a custom URL provided
//
// Deprecated: This function is deprecated and will be removed in a future version. Use NewWithOptions instead.
func NewWithURL(validationURL string, revokeURL string) *Client {
	return NewWithOptions(ClientOptions{
		validationURL: validationURL,
		revokeURL:     revokeURL,
	})
}

// NewWithOptions creates a Client object with custom options. It will default to the standard options if not provided
func NewWithOptions(options ClientOptions) *Client {
	if options.client == nil {
		options.client = &http.Client{
			Timeout: 5 * time.Second,
		}
	}
	if options.validationURL == "" {
		options.validationURL = ValidationURL
	}
	if options.revokeURL == "" {
		options.revokeURL = RevokeURL
	}
	client := &Client{
		validationURL: options.validationURL,
		revokeURL:     options.revokeURL,
		client:        options.client,
	}
	return client
}

// VerifyWebToken sends the WebValidationTokenRequest and gets validation result
func (c *Client) VerifyWebToken(ctx context.Context, code string) (ValidationResponse, error) {
	data := url.Values{
		"client_id":     {c.config.ClientID},
		"client_secret": {c.secret},
		"code":          {code},
		"redirect_uri":  {c.config.RedirectURI},
		"grant_type":    {"authorization_code"},
	}
	return doRequest[ValidationResponse](ctx, c.client, c.validationURL, data)
}

// VerifyAppToken sends the AppValidationTokenRequest and gets validation result
func (c *Client) VerifyAppToken(ctx context.Context, reqBody AppValidationTokenRequest) (ValidationResponse, error) {
	data := url.Values{
		"client_id":     {reqBody.ClientID},
		"client_secret": {reqBody.ClientSecret},
		"code":          {reqBody.Code},
		"grant_type":    {"authorization_code"},
	}

	return doRequest[ValidationResponse](ctx, c.client, c.validationURL, data)
}

// VerifyRefreshToken sends the WebValidationTokenRequest and gets validation result
func (c *Client) VerifyRefreshToken(ctx context.Context, reqBody ValidationRefreshRequest) (RefreshResponse, error) {
	data := url.Values{
		"client_id":     {reqBody.ClientID},
		"client_secret": {reqBody.ClientSecret},
		"refresh_token": {reqBody.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	return doRequest[RefreshResponse](ctx, c.client, c.validationURL, data)
}

// RevokeRefreshToken revokes the Refresh Token and gets the revoke result
func (c *Client) RevokeRefreshToken(ctx context.Context, token string) (RevokeResponse, error) {
	data := url.Values{
		"client_id":       {c.config.ClientID},
		"client_secret":   {c.secret},
		"token":           {token},
		"token_type_hint": {"refresh_token"},
	}

	return doRequest[RevokeResponse](ctx, c.client, c.revokeURL, data)
}

// RevokeAccessToken revokes the Access Token and gets the revoke result
func (c *Client) RevokeAccessToken(ctx context.Context, token string) (RevokeResponse, error) {
	data := url.Values{
		"client_id":       {c.config.ClientID},
		"client_secret":   {c.secret},
		"token":           {token},
		"token_type_hint": {"access_token"},
	}
	logger.Debugw("[Apple OAuth] Revoke access token request", logger.Field("client_id", c.config.ClientID))
	return doRequest[RevokeResponse](ctx, c.client, c.revokeURL, data)
}

// GetUniqueID decodes the id_token response and returns the unique subject ID to identify the user
func GetUniqueID(idToken string) (string, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	return fmt.Sprintf("%v", claims["sub"]), nil
}

// GetClaims decodes the id_token response and returns the JWT claims to identify the user
func GetClaims(idToken string) (*jwt.MapClaims, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return &claims, nil
}

func doRequest[T any](ctx context.Context, client *http.Client, url string, data url.Values) (T, error) {
	var result T
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return result, err
	}
	req.Header.Add("content-type", ContentType)
	req.Header.Add("accept", AcceptHeader)
	req.Header.Add("user-agent", UserAgent) // apple requires a user agent

	res, err := client.Do(req)
	if err != nil {
		return result, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		logger.Errorw("[Apple OAuth] Error response", logger.Field("status", res.Status))
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		if errors.Is(err, io.EOF) {
			return result, nil
		}
		return result, err
	}
	return result, nil
}
