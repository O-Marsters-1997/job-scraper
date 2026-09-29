package cvtailortest

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

type Key string

func (k Key) Get(context.Context, string, string) (string, error) { return string(k), nil }

type NoKey struct{}

func (NoKey) Get(context.Context, string, string) (string, error) { return "", data.ErrNotFound }

var (
	_ cvtailor.Credentials = Key("")
	_ cvtailor.Credentials = NoKey{}
)

type Reply struct {
	Result cvedit.Result
	Err    error
}

type RecordingEditor struct {
	replies []Reply
	Inputs  []cvedit.Input
}

func ReplyingWith(replies ...Reply) *RecordingEditor {
	return &RecordingEditor{replies: replies}
}

func Editing(results ...cvedit.Result) *RecordingEditor {
	replies := make([]Reply, len(results))
	for i, r := range results {
		replies[i] = Reply{Result: r}
	}
	return ReplyingWith(replies...)
}

func (e *RecordingEditor) Edit(_ context.Context, _ string, in cvedit.Input) (cvedit.Result, error) {
	e.Inputs = append(e.Inputs, in)
	r := e.replies[min(len(e.Inputs), len(e.replies))-1]
	return r.Result, r.Err
}

var _ cvtailor.Editor = (*RecordingEditor)(nil)

type failingBatchUpdate struct {
	cvtailor.Drive
	err error
}

func (f failingBatchUpdate) BatchUpdate(context.Context, string, string, []json.RawMessage) error {
	return f.err
}

func FailsBatchUpdate(d cvtailor.Drive, err error) cvtailor.Drive {
	return failingBatchUpdate{Drive: d, err: err}
}

type failingExport struct {
	cvtailor.Drive
	err error
}

func (f failingExport) ExportPDF(context.Context, string, string, string) (io.ReadCloser, error) {
	return nil, f.err
}

func FailsExport(d cvtailor.Drive, err error) cvtailor.Drive {
	return failingExport{Drive: d, err: err}
}

type unreadableExport struct{ cvtailor.Drive }

func (unreadableExport) ExportPDF(context.Context, string, string, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("not a pdf")), nil
}

func ExportsNoPages(d cvtailor.Drive) cvtailor.Drive { return unreadableExport{Drive: d} }

type countedExports struct {
	cvtailor.Drive
	pages []int
	calls int
}

func (c *countedExports) ExportPDF(context.Context, string, string, string) (io.ReadCloser, error) {
	n := c.pages[min(c.calls, len(c.pages)-1)]
	c.calls++
	return pdfWithPages(n), nil
}

func ExportsPages(d cvtailor.Drive, pages ...int) cvtailor.Drive {
	return &countedExports{Drive: d, pages: pages}
}

type exportsPDF struct {
	cvtailor.Drive
	body string
}

func (e exportsPDF) ExportPDF(context.Context, string, string, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(e.body)), nil
}

func ExportsPDF(d cvtailor.Drive, body string) cvtailor.Drive {
	return exportsPDF{Drive: d, body: body}
}
