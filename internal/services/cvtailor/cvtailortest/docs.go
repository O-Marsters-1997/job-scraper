package cvtailortest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/google"
)

type Docs struct {
	TabJSON json.RawMessage
	Err     error
}

func (d Docs) GetDocument(context.Context, string, string, string) (json.RawMessage, error) {
	return d.TabJSON, d.Err
}

var _ cvtailor.DocFetcher = Docs{}

// Drive records the Drive and Docs writes a Draft makes and can fail on
// demand.
type Drive struct {
	mu      sync.Mutex
	Tabs    []google.Tab
	Copies  []string
	Deleted []string
	// PDF is the body ExportPDF serves; PDFErr fails it.
	PDF     string
	PDFErr  error
	Updates [][]json.RawMessage
	// BatchUpdateErr fails every BatchUpdate on a copy.
	BatchUpdateErr error
}

func (d *Drive) ListTabs(context.Context, string, string) ([]google.Tab, error) {
	return d.Tabs, nil
}

func (d *Drive) CopyFile(_ context.Context, _, _, _ string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	id := fmt.Sprintf("copy-%d", len(d.Copies)+1)
	d.Copies = append(d.Copies, id)
	return id, nil
}

func (d *Drive) BatchUpdate(_ context.Context, _, _ string, reqs []json.RawMessage) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.BatchUpdateErr != nil {
		return d.BatchUpdateErr
	}
	d.Updates = append(d.Updates, reqs)
	return nil
}

func (d *Drive) DeleteFile(_ context.Context, _, fileID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Deleted = append(d.Deleted, fileID)
	return nil
}

func (d *Drive) ExportPDF(_ context.Context, _, _, _ string) (io.ReadCloser, error) {
	if d.PDFErr != nil {
		return nil, d.PDFErr
	}
	return io.NopCloser(strings.NewReader(d.PDF)), nil
}

var _ cvtailor.Drive = (*Drive)(nil)
