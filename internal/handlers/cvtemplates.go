package handlers

import (
	"context"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// cvExporter is the interface handlers.ExportCV needs from
// services/cvtemplates.Service; declared here, not imported.
type cvExporter interface {
	ExportPDF(ctx context.Context, userID, docID, tabID string) (io.ReadCloser, error)
}

// ExportCV stays a misfit per ADR 0020: it streams a PDF response
// rather than returning JSON.
func ExportCV(svc cvExporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := caller(w, r)
		if !ok {
			return
		}
		body, err := svc.ExportPDF(r.Context(), userID, chi.URLParam(r, "docId"), chi.URLParam(r, "tabId"))
		if err != nil {
			writeError(w, r, err)
			return
		}
		defer func() { _ = body.Close() }()

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `inline; filename="cv.pdf"`)
		_, _ = io.Copy(w, body)
	}
}
