package handlers

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/google"
)

// CVTemplatesHandler holds ExportCV, which stays a misfit per ADR 0020: it
// streams a PDF response rather than returning JSON.
type CVTemplatesHandler struct {
	google *google.Client
}

func NewCVTemplatesHandler(gc *google.Client) *CVTemplatesHandler {
	return &CVTemplatesHandler{google: gc}
}

func (h *CVTemplatesHandler) ExportCV(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	docID := chi.URLParam(r, "docId")
	tabID := chi.URLParam(r, "tabId")

	body, err := h.google.ExportPDF(r.Context(), session.UserID, docID, tabID)
	if err != nil {
		WriteError(w, r, apperr.Upstream("failed to export PDF"))
		return
	}
	defer func() { _ = body.Close() }()

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="cv.pdf"`)
	_, _ = io.Copy(w, body)
}
