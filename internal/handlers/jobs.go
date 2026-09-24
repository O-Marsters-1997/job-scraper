package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

type JobHandler struct {
	jobs providers.JobProvider
}

func NewJobHandler(jobs providers.JobProvider) *JobHandler {
	return &JobHandler{jobs: jobs}
}

func newSessionCookie(id string, maxAge int) *http.Cookie {
	secure := os.Getenv("COOKIE_SECURE") == "true"
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}
	return &http.Cookie{
		Name:     "session_id",
		Value:    id,
		HttpOnly: true,
		SameSite: sameSite,
		Secure:   secure,
		Path:     "/",
		MaxAge:   maxAge,
	}
}

func (h *JobHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	session, _ := auth.SessionFromContext(r.Context())
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		var err error
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			http.Error(w, "limit must be between 1 and 100", http.StatusBadRequest)
			return
		}
	}
	availability := r.URL.Query().Get("availability")
	if availability != "" && availability != "open" && availability != "closed" && availability != "all" {
		http.Error(w, "invalid availability", http.StatusBadRequest)
		return
	}
	options := providers.JobPageOptions{Limit: int32(limit + 1), Availability: availability, CompanyID: r.URL.Query().Get("company_id")}
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		data, err := base64.RawURLEncoding.DecodeString(raw)
		var cursor struct {
			Time time.Time `json:"time"`
			ID   string    `json:"id"`
		}
		if err == nil {
			err = json.Unmarshal(data, &cursor)
		}
		if err != nil || cursor.Time.IsZero() || cursor.ID == "" {
			http.Error(w, "invalid cursor", http.StatusBadRequest)
			return
		}
		options.CursorTime, options.CursorID = cursor.Time, cursor.ID
	}
	jobs, err := h.jobs.Page(r.Context(), session.UserID, options)
	if err != nil {
		if errors.Is(err, providers.ErrInvalidID) {
			http.Error(w, "invalid company or cursor ID", http.StatusBadRequest)
			return
		}
		slog.Error("list jobs failed",
			slog.Any("err", err),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if len(jobs.Items) > limit {
		jobs.Items = jobs.Items[:limit]
		last := jobs.Items[len(jobs.Items)-1]
		cursor, _ := json.Marshal(struct {
			Time time.Time `json:"time"`
			ID   string    `json:"id"`
		}{last.ScrapedAt, last.ID})
		jobs.NextCursor = base64.RawURLEncoding.EncodeToString(cursor)
	}
	if jobs.Items == nil {
		jobs.Items = []dto.Job{}
	}
	slog.Info("list jobs", slog.Duration("latency", time.Since(started)), slog.Int("count", len(jobs.Items)))
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jobs); err != nil {
		slog.Error("encode jobs failed",
			slog.Any("err", err),
		)
	}
}

func (h *JobHandler) ListAllJobs(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	jobs, err := h.jobs.List(r.Context(), session.UserID)
	if err != nil {
		slog.Error("list all jobs failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if jobs == nil {
		jobs = []dto.Job{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jobs); err != nil {
		slog.Error("encode jobs failed", slog.Any("err", err))
	}
}

func (h *JobHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	job, err := h.jobs.GetJob(r.Context(), chi.URLParam(r, "id"), session.UserID)
	if errors.Is(err, providers.ErrNotFound) {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, providers.ErrInvalidID) {
		http.Error(w, "invalid job ID", http.StatusBadRequest)
		return
	}
	if err != nil {
		slog.Error("get job failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(job); err != nil {
		slog.Error("encode job failed", slog.Any("err", err))
	}
}
