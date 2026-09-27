package handlers

import (
	"context"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	kernel "github.com/ollymarsters/job-scraper/internal/handlers"
)

type cvExporter interface {
	ExportPDF(ctx context.Context, userID, docID, tabID string) (io.ReadCloser, error)
}

type exportRequest struct {
	userID, docID, tabID string
}

// ExportCV streams a PDF response rather than returning JSON, so it goes
// through Handle directly rather than a CRUD generic.
func ExportCV(svc cvExporter) http.HandlerFunc {
	return kernel.Handle(
		func(r *http.Request) (exportRequest, error) {
			uid, err := kernel.UserID(r)
			if err != nil {
				return exportRequest{}, err
			}
			return exportRequest{userID: uid, docID: chi.URLParam(r, "docId"), tabID: chi.URLParam(r, "tabId")}, nil
		},
		func(ctx context.Context, in exportRequest) (io.ReadCloser, error) {
			return svc.ExportPDF(ctx, in.userID, in.docID, in.tabID)
		},
		func(w http.ResponseWriter, _ *http.Request, body io.ReadCloser) {
			defer func() { _ = body.Close() }()
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", `inline; filename="cv.pdf"`)
			_, _ = io.Copy(w, body)
		},
	)
}
