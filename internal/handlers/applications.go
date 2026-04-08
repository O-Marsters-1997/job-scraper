package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
)

type ApplicationHandler struct {
	applications providers.ApplicationProvider
}

func NewApplicationHandler(applications providers.ApplicationProvider) *ApplicationHandler {
	return &ApplicationHandler{applications: applications}
}

func (h *ApplicationHandler) ListApplications(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	statusID := r.URL.Query().Get("status_id")

	var (
		apps any
		err  error
	)
	if statusID != "" {
		apps, err = h.applications.ListApplicationsByUserAndStatus(r.Context(), session.UserID, statusID)
	} else {
		apps, err = h.applications.ListApplicationsByUser(r.Context(), session.UserID)
	}
	if err != nil {
		slog.Error("list applications failed",
			slog.Any("err", err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(apps)
}

func (h *ApplicationHandler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	var body struct {
		JobID      string  `json:"job_id"`
		StatusID   string  `json:"status_id"`
		Notes      string  `json:"notes"`
		AppliedAt  *string `json:"applied_at"`
		SalaryInfo string  `json:"salary_info"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.JobID == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	app, err := h.applications.CreateApplication(r.Context(), session.UserID, body.JobID, body.StatusID, body.Notes, body.SalaryInfo, body.AppliedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"application already exists for this job"}`))
			return
		}
		slog.Error("create application failed",
			slog.Any("err", err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(app)
}

func (h *ApplicationHandler) UpdateApplication(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	id := chi.URLParam(r, "id")
	var body struct {
		StatusID   string  `json:"status_id"`
		Notes      string  `json:"notes"`
		AppliedAt  *string `json:"applied_at"`
		SalaryInfo string  `json:"salary_info"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	app, err := h.applications.UpdateApplication(r.Context(), id, session.UserID, body.StatusID, body.Notes, body.SalaryInfo, body.AppliedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("update application failed",
			slog.Any("err", err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(app)
}

func (h *ApplicationHandler) DeleteApplication(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.applications.DeleteApplication(r.Context(), id, session.UserID); err != nil {
		slog.Error("delete application failed",
			slog.Any("err", err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ApplicationHandler) GetApplicationsForJobs(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	raw := r.URL.Query().Get("job_ids")
	if raw == "" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{})
		return
	}
	jobIDs := strings.Split(raw, ",")
	m, err := h.applications.GetApplicationsForJobs(r.Context(), session.UserID, jobIDs)
	if err != nil {
		slog.Error("get applications for jobs failed",
			slog.Any("err", err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m)
}
