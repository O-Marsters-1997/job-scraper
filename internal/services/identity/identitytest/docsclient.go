package identitytest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/oauth2"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/services/google"
)

type DocsClient struct {
	mu           sync.Mutex
	connectErr   error
	docs         map[string]docFixture
	linked       map[string]bool
	writeGranted bool
	scopes       map[string]string
	ExchangeErr  error
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type docFixture struct {
	tabs []google.Tab
	meta google.FileMeta
}

func NewDocsClient() *DocsClient {
	return &DocsClient{docs: map[string]docFixture{}}
}

// Unlinked returns a DocsClient that starts with no Google Link: users are
// connected by SaveToken and disconnected by DeleteToken, and a granted
// SaveToken records the drive.file scope when writeGranted is true.
func Unlinked(writeGranted bool) *DocsClient {
	return &DocsClient{
		docs:         map[string]docFixture{},
		linked:       map[string]bool{},
		scopes:       map[string]string{},
		writeGranted: writeGranted,
	}
}

// Disconnected returns a DocsClient whose HTTPClientForUser always fails with err.
func Disconnected(err error) *DocsClient {
	return &DocsClient{connectErr: err, docs: map[string]docFixture{}}
}

// WithDoc registers docID's tabs and metadata; returns the receiver for chaining.
func (d *DocsClient) WithDoc(docID string, tabs []google.Tab, meta google.FileMeta) *DocsClient {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.docs[docID] = docFixture{tabs: tabs, meta: meta}
	return d
}

func (d *DocsClient) HTTPClientForUser(_ context.Context, userID string) (*http.Client, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.connectErr != nil {
		return nil, d.connectErr
	}
	if d.linked != nil && !d.linked[userID] {
		return nil, google.ErrTokenNotFound
	}
	return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		body := fmt.Sprintf(`{"email":"%s@example.com"}`, userID)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}, nil
}

func (d *DocsClient) AuthURL(state string, write bool) string {
	u := "https://accounts.google.com/o?state=" + state
	if write {
		u += "&scope=drive.file&include_granted_scopes=true"
	}
	return u
}

func (d *DocsClient) Exchange(context.Context, string) (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "tok"}, d.ExchangeErr
}

func (d *DocsClient) SaveToken(_ context.Context, userID string, _ *oauth2.Token) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.linked != nil {
		d.linked[userID] = true
	}
	if d.writeGranted {
		d.scopes[userID] = google.DriveFileScope
	}
	return nil
}

func (d *DocsClient) DeleteToken(_ context.Context, userID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.linked, userID)
	delete(d.scopes, userID)
	return nil
}

func (d *DocsClient) ListTabs(_ context.Context, _, docID string) ([]google.Tab, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	doc, ok := d.docs[docID]
	if !ok {
		return nil, apperr.NotFound("doc not registered with stub")
	}
	return doc.tabs, nil
}

func (d *DocsClient) FileMeta(_ context.Context, _, docID string) (google.FileMeta, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	doc, ok := d.docs[docID]
	if !ok {
		return google.FileMeta{}, apperr.NotFound("doc not registered with stub")
	}
	return doc.meta, nil
}

func (d *DocsClient) ExportPDF(context.Context, string, string, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("pdf-bytes")), nil
}

func (d *DocsClient) GetDocument(_ context.Context, _, docID, _ string) (json.RawMessage, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.docs[docID]; !ok {
		return nil, apperr.NotFound("doc not registered with stub")
	}
	return json.RawMessage(`{}`), nil
}

func (d *DocsClient) HasScope(_ context.Context, userID, scope string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.scopes[userID] == scope, nil
}

func (d *DocsClient) CopyFile(context.Context, string, string, string) (string, error) {
	return "copy-id", nil
}

func (d *DocsClient) BatchUpdate(context.Context, string, string, []json.RawMessage) error {
	return nil
}

func (d *DocsClient) DeleteFile(context.Context, string, string) error { return nil }
