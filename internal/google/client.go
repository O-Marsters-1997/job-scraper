package google

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
)

// TokenStore persists OAuth tokens keyed by user ID.
type TokenStore interface {
	GetToken(ctx context.Context, userID string) (*oauth2.Token, error)
	SaveToken(ctx context.Context, userID string, tok *oauth2.Token) error
}

// Client wraps an oauth2.Config and a TokenStore for Google OAuth.
type Client struct {
	cfg   *oauth2.Config
	store TokenStore
}

// NewClient constructs a Client with the given credentials and redirect URL.
func NewClient(clientID, clientSecret, redirectURL string, store TokenStore) *Client {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"https://www.googleapis.com/auth/drive.readonly"},
		Endpoint:     googleoauth.Endpoint,
	}
	return &Client{cfg: cfg, store: store}
}

// AuthURL returns the Google OAuth consent page URL for the given state token.
func (c *Client) AuthURL(state string) string {
	return c.cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// Exchange converts an authorization code into an OAuth token.
func (c *Client) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return c.cfg.Exchange(ctx, code)
}

// SaveToken persists a token for the given user.
func (c *Client) SaveToken(ctx context.Context, userID string, tok *oauth2.Token) error {
	return c.store.SaveToken(ctx, userID, tok)
}

// DeleteToken removes the stored token for the given user.
func (c *Client) DeleteToken(ctx context.Context, userID string) error {
	type deleter interface {
		DeleteToken(ctx context.Context, userID string) error
	}
	if d, ok := c.store.(deleter); ok {
		return d.DeleteToken(ctx, userID)
	}
	return nil
}

// HTTPClientForUser returns an *http.Client that automatically refreshes the
// stored OAuth token for the given user ID.
func (c *Client) HTTPClientForUser(ctx context.Context, userID string) (*http.Client, error) {
	tok, err := c.store.GetToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	// savingSource wraps the token source and persists refreshed tokens.
	ts := &savingSource{
		ctx:    ctx,
		userID: userID,
		store:  c.store,
		src:    c.cfg.TokenSource(ctx, tok),
	}

	return oauth2.NewClient(ctx, ts), nil
}

// savingSource is an oauth2.TokenSource that writes refreshed tokens back to
// the store so the new refresh token is not lost.
type savingSource struct {
	ctx    context.Context
	userID string
	store  TokenStore
	src    oauth2.TokenSource
}

func (s *savingSource) Token() (*oauth2.Token, error) {
	tok, err := s.src.Token()
	if err != nil {
		return nil, err
	}
	// Best-effort persist; ignore the error so the caller still gets a client.
	_ = s.store.SaveToken(s.ctx, s.userID, tok)
	return tok, nil
}
