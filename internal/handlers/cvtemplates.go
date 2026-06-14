package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/cvtemplates"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/google"
)

type CVTemplatesHandler struct {
	svc    *cvtemplates.Service
	google *google.Client
}

func NewCVTemplatesHandler(svc *cvtemplates.Service, gc *google.Client) *CVTemplatesHandler {
	return &CVTemplatesHandler{svc: svc, google: gc}
}

func (h *CVTemplatesHandler) ListCVTemplates(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	cvs, err := h.svc.List(r.Context(), session.UserID)
	if err != nil {
		if strings.Contains(err.Error(), "not connected") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"google account not connected"}`))
			return
		}
		slog.Error("list cv templates failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cvs)
}

func (h *CVTemplatesHandler) AddTrackedDoc(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	session, _ := auth.SessionFromContext(r.Context())
	if err := h.svc.AddDoc(r.Context(), session.UserID, body.URL); err != nil {
		// Parse errors and access errors are client faults.
		if isClientError(err) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		slog.Error("add tracked doc failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *CVTemplatesHandler) RemoveTrackedDoc(w http.ResponseWriter, r *http.Request) {
	docID := chi.URLParam(r, "docId")
	session, _ := auth.SessionFromContext(r.Context())
	if err := h.svc.RemoveDoc(r.Context(), session.UserID, docID); err != nil {
		if errors.Is(err, providers.ErrTrackedDocNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("remove tracked doc failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CVTemplatesHandler) ExportCV(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	docID := chi.URLParam(r, "docId")
	tabID := chi.URLParam(r, "tabId")

	body, err := h.google.ExportPDF(r.Context(), session.UserID, docID, tabID)
	if err != nil {
		slog.Error("ExportCV", slog.Any("err", err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to export CV"})
		return
	}
	defer func() { _ = body.Close() }()

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="cv.pdf"`)
	_, _ = io.Copy(w, body)
}

func (h *CVTemplatesHandler) HideTab(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	docID := chi.URLParam(r, "docId")
	tabID := chi.URLParam(r, "tabId")

	if err := h.svc.HideTab(r.Context(), session.UserID, docID, tabID); err != nil {
		if errors.Is(err, providers.ErrTabNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("hide tab failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CVTemplatesHandler) ShowTab(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	docID := chi.URLParam(r, "docId")
	tabID := chi.URLParam(r, "tabId")

	if err := h.svc.ShowTab(r.Context(), session.UserID, docID, tabID); err != nil {
		if errors.Is(err, providers.ErrTabNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("show tab failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func isClientError(err error) bool {
	msg := err.Error()
	return strings.HasPrefix(msg, "invalid Google Docs") ||
		strings.HasPrefix(msg, "cannot access document")
}
