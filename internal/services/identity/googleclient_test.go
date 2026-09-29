package identity_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/oauth2"

	"github.com/ollymarsters/job-scraper/internal/services/google"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type fakeGoogleClient struct {
	mu          sync.Mutex
	connected   map[string]bool
	exchangeErr error
	tabs        []google.Tab
	meta        google.FileMeta
}

func newFakeGoogleClient() *fakeGoogleClient {
	return &fakeGoogleClient{connected: map[string]bool{}}
}

func (f *fakeGoogleClient) AuthURL(state string) string {
	return "https://accounts.google.com/o?state=" + state
}

func (f *fakeGoogleClient) Exchange(context.Context, string) (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "tok"}, f.exchangeErr
}

func (f *fakeGoogleClient) SaveToken(_ context.Context, userID string, _ *oauth2.Token) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.connected[userID] = true
	return nil
}

func (f *fakeGoogleClient) DeleteToken(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.connected, userID)
	return nil
}

func (f *fakeGoogleClient) HTTPClientForUser(_ context.Context, userID string) (*http.Client, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.connected[userID] {
		return nil, google.ErrTokenNotFound
	}
	return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		body := fmt.Sprintf(`{"email":"%s@example.com"}`, userID)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}, nil
}

func (f *fakeGoogleClient) ListTabs(context.Context, string, string) ([]google.Tab, error) {
	return f.tabs, nil
}

func (f *fakeGoogleClient) FileMeta(context.Context, string, string) (google.FileMeta, error) {
	return f.meta, nil
}

func (f *fakeGoogleClient) ExportPDF(context.Context, string, string, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("pdf-bytes")), nil
}

func (f *fakeGoogleClient) GetDocument(context.Context, string, string, string) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}
