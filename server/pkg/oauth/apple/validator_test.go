package apple

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestVerifyAppTokenReturnsTypedValidationResponse(t *testing.T) {
	client := NewWithOptions(ClientOptions{
		validationURL: "https://apple.test/auth/token",
		client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if got := r.Form.Get("grant_type"); got != "authorization_code" {
				t.Fatalf("grant_type = %q, want authorization_code", got)
			}
			if got := r.Form.Get("code"); got != "code-1" {
				t.Fatalf("code = %q, want code-1", got)
			}
			var body strings.Builder
			if err := json.NewEncoder(&body).Encode(ValidationResponse{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				IDToken:      "id-token",
				ExpiresIn:    3600,
			}); err != nil {
				t.Fatalf("encode response: %v", err)
			}
			return jsonResponse(http.StatusOK, body.String()), nil
		})},
	})

	resp, err := client.VerifyAppToken(context.Background(), AppValidationTokenRequest{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		Code:         "code-1",
	})
	if err != nil {
		t.Fatalf("verify app token: %v", err)
	}
	if resp.AccessToken != "access-token" || resp.RefreshToken != "refresh-token" || resp.IDToken != "id-token" || resp.ExpiresIn != 3600 {
		t.Fatalf("unexpected validation response: %+v", resp)
	}
}

func TestRevokeAccessTokenAllowsEmptySuccessBody(t *testing.T) {
	client := NewWithOptions(ClientOptions{
		revokeURL: "https://apple.test/auth/revoke",
		client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if got := r.Form.Get("token_type_hint"); got != "access_token" {
				t.Fatalf("token_type_hint = %q, want access_token", got)
			}
			return jsonResponse(http.StatusOK, ""), nil
		})},
	})
	client.config = Config{ClientID: "client-id"}
	client.secret = "client-secret"

	resp, err := client.RevokeAccessToken(context.Background(), "access-token")
	if err != nil {
		t.Fatalf("revoke access token: %v", err)
	}
	if resp.Error != "" || resp.ErrorDescription != "" {
		t.Fatalf("unexpected revoke response: %+v", resp)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
