package identitytest

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/services/google"
)

type DocsClient struct {
	mu         sync.Mutex
	connectErr error
	docs       map[string]docFixture
}

type docFixture struct {
	tabs []google.Tab
	meta google.FileMeta
}

func NewDocsClient() *DocsClient {
	return &DocsClient{docs: map[string]docFixture{}}
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

func (d *DocsClient) HTTPClientForUser(context.Context, string) (*http.Client, error) {
	if d.connectErr != nil {
		return nil, d.connectErr
	}
	return &http.Client{}, nil
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
