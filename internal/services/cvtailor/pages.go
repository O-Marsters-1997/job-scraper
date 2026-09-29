package cvtailor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
)

var pageObjectRe = regexp.MustCompile(`/Type\s*/Page\b`)

var errNoPages = errors.New("no pages found in the exported PDF")

func (g *Generator) pageCount(ctx context.Context, userID, docID, tabID string) (int, error) {
	body, err := g.drive.ExportPDF(ctx, userID, docID, tabID)
	if err != nil {
		return 0, err
	}
	defer func() { _ = body.Close() }()
	pdf, err := io.ReadAll(body)
	if err != nil {
		return 0, err
	}
	n := len(pageObjectRe.FindAll(pdf, -1))
	if n == 0 {
		return 0, fmt.Errorf("%w (%d bytes)", errNoPages, len(pdf))
	}
	return n, nil
}
