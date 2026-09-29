package cvtailortest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
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
	Updates [][]json.RawMessage
	// BasePages is the page count of the base CV, one when zero.
	BasePages int
	// DraftPages holds the page count of each successive export of a copy;
	// the last repeats, and one is assumed when it is empty.
	DraftPages  []int
	copyExports int
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

func (d *Drive) ExportPDF(_ context.Context, _, docID, _ string) (io.ReadCloser, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	pages := max(d.BasePages, 1)
	if slices.Contains(d.Copies, docID) {
		pages = 1
		if len(d.DraftPages) > 0 {
			pages = d.DraftPages[min(d.copyExports, len(d.DraftPages)-1)]
		}
		d.copyExports++
	}
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	for i := range pages {
		fmt.Fprintf(&b, "%d 0 obj\n<< /Type /Page /Parent 1 0 R >>\nendobj\n", i+2)
	}
	fmt.Fprintf(&b, "1 0 obj\n<< /Type /Pages /Count %d >>\nendobj\n", pages)
	return io.NopCloser(strings.NewReader(b.String())), nil
}

func (d *Drive) DeleteFile(_ context.Context, _, fileID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Deleted = append(d.Deleted, fileID)
	return nil
}

var _ cvtailor.Drive = (*Drive)(nil)
