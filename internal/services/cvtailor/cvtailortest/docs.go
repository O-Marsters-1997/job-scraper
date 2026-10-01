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

type EditedCopyDocs struct {
	BaseDocID  string
	Base, Copy json.RawMessage
}

func (d EditedCopyDocs) GetDocument(_ context.Context, _, docID, _ string) (json.RawMessage, error) {
	if docID == d.BaseDocID {
		return d.Base, nil
	}
	return d.Copy, nil
}

var (
	_ cvtailor.DocFetcher = Docs{}
	_ cvtailor.DocFetcher = EditedCopyDocs{}
)

type Drive struct {
	mu        sync.Mutex
	Tabs      []google.Tab
	Copies    []string
	Deleted   []string
	Renamed   map[string]string
	RenameErr error
	Updates   [][]json.RawMessage
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
	d.Updates = append(d.Updates, reqs)
	return nil
}

func (d *Drive) ExportPDF(_ context.Context, _, _, _ string) (io.ReadCloser, error) {
	return pdfWithPages(1), nil
}

func (d *Drive) DeleteFile(_ context.Context, _, fileID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Deleted = append(d.Deleted, fileID)
	return nil
}

func (d *Drive) RenameFile(_ context.Context, _, fileID, name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.RenameErr != nil {
		return d.RenameErr
	}
	if d.Renamed == nil {
		d.Renamed = map[string]string{}
	}
	d.Renamed[fileID] = name
	return nil
}

func pdfWithPages(pages int) io.ReadCloser {
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	for i := range pages {
		fmt.Fprintf(&b, "%d 0 obj\n<< /Type /Page /Parent 1 0 R >>\nendobj\n", i+2)
	}
	fmt.Fprintf(&b, "1 0 obj\n<< /Type /Pages /Count %d >>\nendobj\n", pages)
	return io.NopCloser(strings.NewReader(b.String()))
}

var _ cvtailor.Drive = (*Drive)(nil)
