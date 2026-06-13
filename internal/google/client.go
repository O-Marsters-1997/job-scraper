package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
)

// Tab represents a tab within a Google Doc.
type Tab struct {
	ID    string
	Title string
}

// FileMeta holds basic metadata for a Google Drive file.
type FileMeta struct {
	Title      string
	ModifiedAt time.Time
}

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

// ListTabs returns the tabs for the given Google Doc.
func (c *Client) ListTabs(ctx context.Context, userID, docID string) ([]Tab, error) {
	hc, err := c.HTTPClientForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("google.ListTabs: %w", err)
	}

	url := fmt.Sprintf("https://docs.googleapis.com/v1/documents/%s?fields=tabs.tabProperties", docID)
	resp, err := hc.Get(url)
	if err != nil {
		return nil, fmt.Errorf("google.ListTabs request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google.ListTabs: unexpected status %d", resp.StatusCode)
	}

	var body struct {
		Tabs []struct {
			TabProperties struct {
				TabID string `json:"tabId"`
				Title string `json:"title"`
			} `json:"tabProperties"`
		} `json:"tabs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("google.ListTabs decode: %w", err)
	}

	tabs := make([]Tab, len(body.Tabs))
	for i, t := range body.Tabs {
		tabs[i] = Tab{ID: t.TabProperties.TabID, Title: t.TabProperties.Title}
	}
	return tabs, nil
}

// FileMeta returns basic metadata (title and modified time) for the given Drive file.
func (c *Client) FileMeta(ctx context.Context, userID, docID string) (FileMeta, error) {
	hc, err := c.HTTPClientForUser(ctx, userID)
	if err != nil {
		return FileMeta{}, fmt.Errorf("google.FileMeta: %w", err)
	}

	url := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?fields=name,modifiedTime", docID)
	resp, err := hc.Get(url)
	if err != nil {
		return FileMeta{}, fmt.Errorf("google.FileMeta request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return FileMeta{}, fmt.Errorf("google.FileMeta: unexpected status %d", resp.StatusCode)
	}

	var body struct {
		Name         string `json:"name"`
		ModifiedTime string `json:"modifiedTime"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return FileMeta{}, fmt.Errorf("google.FileMeta decode: %w", err)
	}

	modifiedAt, err := time.Parse(time.RFC3339, body.ModifiedTime)
	if err != nil {
		return FileMeta{}, fmt.Errorf("google.FileMeta parse time: %w", err)
	}

	return FileMeta{Title: body.Name, ModifiedAt: modifiedAt}, nil
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
