package cvtailortest

import (
	"context"
	"encoding/json"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/lineedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/profiledraft"
)

type RecordingProfileWriter struct {
	result profiledraft.Result
	err    error
	Inputs []profiledraft.Input
}

func Redrafting(res profiledraft.Result, err error) *RecordingProfileWriter {
	return &RecordingProfileWriter{result: res, err: err}
}

func (w *RecordingProfileWriter) Redraft(_ context.Context, _ string, in profiledraft.Input) (profiledraft.Result, error) {
	w.Inputs = append(w.Inputs, in)
	return w.result, w.err
}

var _ cvtailor.ProfileWriter = (*RecordingProfileWriter)(nil)

type Key string

func (k Key) Get(context.Context, string, string) (string, error) { return string(k), nil }

type NoKey struct{}

func (NoKey) Get(context.Context, string, string) (string, error) { return "", data.ErrNotFound }

var (
	_ cvtailor.Credentials = Key("")
	_ cvtailor.Credentials = NoKey{}
)

type RecordingEditor struct {
	explanation   string
	ExplainInputs []cvedit.ExplainInput

	insightLines  []cvedit.InsightLine
	InsightInputs []cvedit.InsightsInput
}

func Insighting(lines ...cvedit.InsightLine) *RecordingEditor {
	return &RecordingEditor{insightLines: lines}
}

func (e *RecordingEditor) Insights(_ context.Context, _ string, in cvedit.InsightsInput) (cvedit.InsightsResult, error) {
	e.InsightInputs = append(e.InsightInputs, in)
	return cvedit.InsightsResult{Lines: slices.Clone(e.insightLines), Cost: 0.01}, nil
}

// Explaining is an Editor whose explanations reply with text.
func Explaining(text string) *RecordingEditor {
	return &RecordingEditor{explanation: text}
}

func (e *RecordingEditor) Explain(_ context.Context, _ string, in cvedit.ExplainInput) (cvedit.ExplainResult, error) {
	e.ExplainInputs = append(e.ExplainInputs, in)
	return cvedit.ExplainResult{Text: e.explanation, Cost: 0.001}, nil
}

type RecordingModel struct {
	deltas  []string
	err     error
	pace    time.Duration
	Prompts []lineedit.Prompt
}

func Suggesting(deltas ...string) *RecordingModel {
	return &RecordingModel{deltas: deltas}
}

func SuggestingEvery(pace time.Duration, deltas ...string) *RecordingModel {
	return &RecordingModel{deltas: deltas, pace: pace}
}

// SuggestFailing streams the deltas, then fails with err; with no deltas the
// stream never opens.
func SuggestFailing(err error, deltas ...string) *RecordingModel {
	return &RecordingModel{deltas: deltas, err: err}
}

func (m *RecordingModel) Complete(_ context.Context, _ string, p lineedit.Prompt, onDelta func(string)) (lineedit.Completion, error) {
	m.Prompts = append(m.Prompts, p)
	for _, d := range m.deltas {
		time.Sleep(m.pace)
		onDelta(d)
	}
	return lineedit.Completion{Text: strings.Join(m.deltas, ""), Cost: 0.001}, m.err
}

var _ lineedit.Model = (*RecordingModel)(nil)

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
