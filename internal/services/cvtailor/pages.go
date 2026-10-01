package cvtailor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"

	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/pdftext"
)

var pageObjectRe = regexp.MustCompile(`/Type\s*/Page\b`)

var errNoPages = errors.New("no pages found in the exported PDF")

func (m *Module) exportPDF(ctx context.Context, userID, docID, tabID string) ([]byte, error) {
	body, err := m.drive.ExportPDF(ctx, userID, docID, tabID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()
	return io.ReadAll(body)
}

func pageCount(pdf []byte) (int, error) {
	n := len(pageObjectRe.FindAll(pdf, -1))
	if n == 0 {
		return 0, fmt.Errorf("%w (%d bytes)", errNoPages, len(pdf))
	}
	return n, nil
}

func (m *Module) measure(ctx context.Context, userID, docID, tabID string) ([]byte, int, error) {
	pdf, err := m.exportPDF(ctx, userID, docID, tabID)
	if err != nil {
		return nil, 0, err
	}
	n, err := pageCount(pdf)
	return pdf, n, err
}

func parseInput(ctx context.Context, pdf []byte, headings []docparse.Heading) *checks.ParseInput {
	lines, err := pdftext.Lines(pdf)
	if err != nil {
		slog.WarnContext(ctx, "extract draft PDF text failed", slog.Any(logger.KeyErr, err))
		return nil
	}
	if len(lines) == 0 {
		return nil
	}
	in := &checks.ParseInput{Lines: lines}
	for _, h := range headings {
		in.Headings = append(in.Headings, h.Text)
	}
	return in
}
