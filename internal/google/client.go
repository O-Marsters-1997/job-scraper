package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
)

type Tab struct {
	ID    string
	Title string
}

type FileMeta struct {
	Title      string
	ModifiedAt time.Time
}

type TokenStore interface {
	GetToken(ctx context.Context, userID string) (*oauth2.Token, error)
	SaveToken(ctx context.Context, userID string, tok *oauth2.Token) error
	DeleteToken(ctx context.Context, userID string) error
}

type Client struct {
	cfg   *oauth2.Config
	store TokenStore
}

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

func (c *Client) AuthURL(state string) string {
	return c.cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (c *Client) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return c.cfg.Exchange(ctx, code)
}

func (c *Client) SaveToken(ctx context.Context, userID string, tok *oauth2.Token) error {
	return c.store.SaveToken(ctx, userID, tok)
}

func (c *Client) DeleteToken(ctx context.Context, userID string) error {
	return c.store.DeleteToken(ctx, userID)
}

// HTTPClientForUser returns an *http.Client that automatically refreshes the
// stored OAuth token for the given user ID.
func (c *Client) HTTPClientForUser(ctx context.Context, userID string) (*http.Client, error) {
	tok, err := c.store.GetToken(ctx, userID)
	if err != nil {
		return nil, err
	}

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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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

// The caller must close the returned body.
// tabID is the Google Docs tab id (e.g. "t.0"); the "t." prefix is normalised
// defensively. Pass an empty string to export the whole document.
func (c *Client) ExportPDF(ctx context.Context, userID, docID, tabID string) (io.ReadCloser, error) {
	httpClient, err := c.HTTPClientForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("google.ExportPDF: %w", err)
	}

	exportURL := fmt.Sprintf("https://docs.google.com/document/d/%s/export?format=pdf", docID)
	if tabID != "" {
		tab := tabID
		if !strings.HasPrefix(tab, "t.") {
			tab = "t." + tab
		}
		exportURL += "&tab=" + tab
	}
	resp, err := httpClient.Get(exportURL)
	if err != nil {
		return nil, fmt.Errorf("google.ExportPDF request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("export failed: %s", resp.Status)
	}

	return resp.Body, nil
}

func (s *savingSource) Token() (*oauth2.Token, error) {
	tok, err := s.src.Token()
	if err != nil {
		return nil, err
	}
	_ = s.store.SaveToken(s.ctx, s.userID, tok)
	return tok, nil
}
