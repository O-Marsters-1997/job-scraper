package cvtailortest

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

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

type RecordingEditor struct {
	deltas        []string
	suggestErr    error
	pace          time.Duration
	SuggestInputs []cvedit.SuggestInput

	explanation   string
	ExplainInputs []cvedit.ExplainInput
}

// Explaining is an Editor whose explanations reply with text.
func Explaining(text string) *RecordingEditor {
	return &RecordingEditor{explanation: text}
}

func (e *RecordingEditor) Explain(_ context.Context, _ string, in cvedit.ExplainInput) (cvedit.ExplainResult, error) {
	e.ExplainInputs = append(e.ExplainInputs, in)
	return cvedit.ExplainResult{Text: e.explanation, Cost: 0.001}, nil
}

// Suggesting is an Editor whose suggestions stream the given deltas.
func Suggesting(deltas ...string) *RecordingEditor {
	return &RecordingEditor{deltas: deltas}
}

func SuggestingEvery(pace time.Duration, deltas ...string) *RecordingEditor {
	return &RecordingEditor{deltas: deltas, pace: pace}
}

// SuggestFailing streams the deltas, then fails with err; with no deltas the
// stream never opens.
func SuggestFailing(err error, deltas ...string) *RecordingEditor {
	return &RecordingEditor{deltas: deltas, suggestErr: err}
}

func (e *RecordingEditor) Suggest(_ context.Context, _ string, in cvedit.SuggestInput, onDelta func(string)) (cvedit.SuggestResult, error) {
	e.SuggestInputs = append(e.SuggestInputs, in)
	for _, d := range e.deltas {
		time.Sleep(e.pace)
		onDelta(d)
	}
	return cvedit.SuggestResult{Text: strings.Join(e.deltas, ""), Cost: 0.001}, e.suggestErr
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

type failsFirstBatchUpdate struct {
	cvtailor.Drive
	err    error
	failed bool
}

func (f *failsFirstBatchUpdate) BatchUpdate(ctx context.Context, userID, docID string, reqs []json.RawMessage) error {
	if !f.failed {
		f.failed = true
		return f.err
	}
	return f.Drive.BatchUpdate(ctx, userID, docID, reqs)
}

func FailsFirstBatchUpdate(d cvtailor.Drive, err error) cvtailor.Drive {
	return &failsFirstBatchUpdate{Drive: d, err: err}
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
