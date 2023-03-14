package auth

import (
	"context"
	"net/url"

	"golang.org/x/oauth2"
	oauth2_google "golang.org/x/oauth2/google"
)

var (
	scopes = []string{
		"https://www.googleapis.com/auth/cloud-platform.read-only",
	}
)

type ApplicationDefaultCredentialStore struct {
	tokenSource oauth2.TokenSource
}

func NewApplicationDefaultCredentialStore(context context.Context) (*ApplicationDefaultCredentialStore, error) {
	ts, err := oauth2_google.DefaultTokenSource(context, scopes...)
	if err != nil {
		return nil, err
	}
	ts = oauth2.ReuseTokenSource(nil, ts)

	// Test the token source
	if _, err := ts.Token(); err != nil {
		return nil, err
	}

	return &ApplicationDefaultCredentialStore{
		tokenSource: ts,
	}, nil
}

func (a *ApplicationDefaultCredentialStore) Basic(*url.URL) (string, string) {
	token, err := a.tokenSource.Token()
	if err != nil {
		return "", ""
	}
	return "_dcgcloud_token", token.AccessToken
}

func (a *ApplicationDefaultCredentialStore) RefreshToken(*url.URL, string) string {
	return ""
}

func (a *ApplicationDefaultCredentialStore) SetRefreshToken(*url.URL, string, string) {
}
