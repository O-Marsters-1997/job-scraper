package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/tokencrypt"
)

const (
	DriveReadonlyScope = "https://www.googleapis.com/auth/drive.readonly"
	DriveFileScope     = "https://www.googleapis.com/auth/drive.file"
)

type Tab struct {
	ID    string
	Title string
}

type FileMeta struct {
	Title      string
	ModifiedAt time.Time
}

// Client is the OAuth2 and Docs/Drive API surface, backed by a Store that
// persists encrypted tokens.
type Client struct {
	cfg   *oauth2.Config
	store Store
}

func NewClient(clientID, clientSecret, redirectURL string, store Store) *Client {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{DriveReadonlyScope},
		Endpoint:     googleoauth.Endpoint,
	}
	return &Client{cfg: cfg, store: store}
}

// Google returns a refresh token only when the consent screen is forced.
func (c *Client) AuthURL(state string, write bool) string {
	if !write {
		return c.cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
	}
	cfg := *c.cfg
	cfg.Scopes = []string{DriveFileScope}
	return cfg.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
		oauth2.SetAuthURLParam("include_granted_scopes", "true"),
	)
}

func (c *Client) HasScope(ctx context.Context, userID, scope string) (bool, error) {
	row, err := c.store.GetGoogleToken(ctx, userID)
	if err != nil {
		return false, err
	}
	return slices.Contains(strings.Fields(row.Scope), scope), nil
}

func (c *Client) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return c.cfg.Exchange(ctx, code)
}

// SaveToken encrypts and persists the OAuth token for the given user.
func (c *Client) SaveToken(ctx context.Context, userID string, tok *oauth2.Token) error {
	accessEnc, err := tokencrypt.Encrypt(tok.AccessToken)
	if err != nil {
		return fmt.Errorf("google.SaveToken encrypt access: %w", err)
	}
	refreshEnc, err := tokencrypt.Encrypt(tok.RefreshToken)
	if err != nil {
		return fmt.Errorf("google.SaveToken encrypt refresh: %w", err)
	}

	var expiry time.Time
	if !tok.Expiry.IsZero() {
		expiry = tok.Expiry
	}

	scope := c.grantedScope(ctx, userID, tok)

	return c.store.UpsertGoogleToken(ctx, dto.UpsertGoogleTokenInput{
		UserID:          userID,
		AccessTokenEnc:  accessEnc,
		RefreshTokenEnc: refreshEnc,
		TokenType:       tok.TokenType,
		Expiry:          expiry,
		Scope:           scope,
	})
}

// Google's refresh response may omit the scope field.
func (c *Client) grantedScope(ctx context.Context, userID string, tok *oauth2.Token) string {
	if scope, _ := tok.Extra("scope").(string); scope != "" {
		return scope
	}
	if row, err := c.store.GetGoogleToken(ctx, userID); err == nil && row.Scope != "" {
		return row.Scope
	}
	return DriveReadonlyScope
}

func (c *Client) DeleteToken(ctx context.Context, userID string) error {
	return c.store.DeleteGoogleToken(ctx, userID)
}

func (c *Client) getToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	row, err := c.store.GetGoogleToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	access, err := tokencrypt.Decrypt(row.AccessTokenEnc)
	if err != nil {
		return nil, fmt.Errorf("%w: decrypt access: %w", ErrTokenUnusable, err)
	}
	refresh, err := tokencrypt.Decrypt(row.RefreshTokenEnc)
	if err != nil {
		return nil, fmt.Errorf("%w: decrypt refresh: %w", ErrTokenUnusable, err)
	}

	return &oauth2.Token{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    row.TokenType,
		Expiry:       row.Expiry,
	}, nil
}

// HTTPClientForUser returns an *http.Client that automatically refreshes the
// stored OAuth token for the given user ID.
func (c *Client) HTTPClientForUser(ctx context.Context, userID string) (*http.Client, error) {
	tok, err := c.getToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("google.HTTPClientForUser: %w", err)
	}

	ts := &savingSource{
		ctx:    ctx,
		userID: userID,
		client: c,
		src:    c.cfg.TokenSource(ctx, tok),
	}

	return oauth2.NewClient(ctx, ts), nil
}

// savingSource is an oauth2.TokenSource that writes refreshed tokens back to
// the store so the new refresh token is not lost.
type savingSource struct {
	ctx    context.Context
	userID string
	client *Client
	src    oauth2.TokenSource
}

func (s *savingSource) Token() (*oauth2.Token, error) {
	tok, err := s.src.Token()
	if err != nil {
		return nil, err
	}
	_ = s.client.SaveToken(s.ctx, s.userID, tok)
	return tok, nil
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

// GetDocument returns the JSON of one Tab of the Doc, child Tabs searched
// recursively, for docparse.Parse. An empty tabID selects the first Tab.
func (c *Client) GetDocument(ctx context.Context, userID, docID, tabID string) (json.RawMessage, error) {
	hc, err := c.HTTPClientForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("google.GetDocument: %w", err)
	}

	url := fmt.Sprintf("https://docs.googleapis.com/v1/documents/%s?includeTabsContent=true&fields=tabs", docID)
	resp, err := hc.Get(url)
	if err != nil {
		return nil, fmt.Errorf("google.GetDocument request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google.GetDocument: unexpected status %d", resp.StatusCode)
	}

	var body struct {
		Tabs []json.RawMessage `json:"tabs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("google.GetDocument decode: %w", err)
	}

	tab, ok := findTab(body.Tabs, tabID)
	if !ok {
		return nil, apperr.NotFound("tab not found in document")
	}
	return tab, nil
}

func findTab(tabs []json.RawMessage, tabID string) (json.RawMessage, bool) {
	for _, raw := range tabs {
		var t struct {
			TabProperties struct {
				TabID string `json:"tabId"`
			} `json:"tabProperties"`
			ChildTabs []json.RawMessage `json:"childTabs"`
		}
		if err := json.Unmarshal(raw, &t); err != nil {
			continue
		}
		if tabID == "" || t.TabProperties.TabID == tabID {
			return raw, true
		}
		if child, ok := findTab(t.ChildTabs, tabID); ok {
			return child, true
		}
	}
	return nil, false
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

func (c *Client) CopyFile(ctx context.Context, userID, fileID, name string) (string, error) {
	body, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return "", fmt.Errorf("google.CopyFile marshal: %w", err)
	}
	resp, err := c.do(ctx, userID, http.MethodPost,
		fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s/copy", url.PathEscape(fileID)), body)
	if err != nil {
		return "", fmt.Errorf("google.CopyFile: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("google.CopyFile decode: %w", err)
	}
	return out.ID, nil
}

func (c *Client) BatchUpdate(ctx context.Context, userID, docID string, requests []json.RawMessage) error {
	body, err := json.Marshal(map[string][]json.RawMessage{"requests": requests})
	if err != nil {
		return fmt.Errorf("google.BatchUpdate marshal: %w", err)
	}
	resp, err := c.do(ctx, userID, http.MethodPost,
		fmt.Sprintf("https://docs.googleapis.com/v1/documents/%s:batchUpdate", url.PathEscape(docID)), body)
	if err != nil {
		return fmt.Errorf("google.BatchUpdate: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}

func (c *Client) DeleteFile(ctx context.Context, userID, fileID string) error {
	resp, err := c.do(ctx, userID, http.MethodDelete,
		fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s", url.PathEscape(fileID)), nil)
	if err != nil {
		return fmt.Errorf("google.DeleteFile: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}

func (c *Client) do(ctx context.Context, userID, method, target string, body []byte) (*http.Response, error) {
	hc, err := c.HTTPClientForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return resp, nil
}
