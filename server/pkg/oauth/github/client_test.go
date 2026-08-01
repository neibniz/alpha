package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestClientAuthCodeURLAndUserInfo(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("authorization header = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Fatalf("accept header = %q", got)
		}
		_, _ = w.Write([]byte(`{"id":42,"login":"octocat","avatar_url":"https://avatars.example/42","email":"public@example.test"}`))
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("per_page"); got != "100" {
			t.Fatalf("per_page = %q", got)
		}
		_, _ = w.Write([]byte(`[{"email":"other@example.test","verified":true,"primary":false},{"email":"Primary@Example.Test","verified":true,"primary":true}]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := New(Config{ClientID: "client", ClientSecret: "secret", RedirectURL: "https://panel.example/oauth/github/callback"})
	client.apiBaseURL = server.URL

	authURL, err := url.Parse(client.AuthCodeURL("state-value"))
	if err != nil {
		t.Fatal(err)
	}
	if authURL.Host != "github.com" || authURL.Query().Get("state") != "state-value" {
		t.Fatalf("unexpected auth URL: %s", authURL)
	}
	if got := authURL.Query().Get("scope"); got != "read:user user:email" {
		t.Fatalf("scope = %q", got)
	}

	info, err := client.GetUserInfo(context.Background(), "token")
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != 42 || info.Login != "octocat" || info.Email != "primary@example.test" {
		t.Fatalf("unexpected user info: %+v", info)
	}
}

func TestClientRequiresVerifiedEmail(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":42,"login":"octocat"}`))
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"email":"private@example.test","verified":false,"primary":true}]`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := New(Config{})
	client.apiBaseURL = server.URL
	_, err := client.GetUserInfo(context.Background(), "token")
	if !errors.Is(err, ErrVerifiedEmailRequired) {
		t.Fatalf("error = %v, want ErrVerifiedEmailRequired", err)
	}
}

func TestClientRejectsGitHubAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"bad credentials"}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	client := New(Config{})
	client.apiBaseURL = server.URL
	if _, err := client.GetUserInfo(context.Background(), "token"); err == nil {
		t.Fatal("GitHub API error should be returned")
	}
}
