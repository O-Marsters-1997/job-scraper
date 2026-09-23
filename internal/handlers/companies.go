package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

type CompaniesHandler struct {
	companies providers.CompanyProvider
	targets   providers.SourceTargetProvider
	q         queue.JobQueue
}

func NewCompaniesHandler(companies providers.CompanyProvider, targets providers.SourceTargetProvider, q queue.JobQueue) *CompaniesHandler {
	return &CompaniesHandler{companies: companies, targets: targets, q: q}
}

func (h *CompaniesHandler) List(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	cs, err := h.companies.ListCompaniesForUser(r.Context(), session.UserID)
	if err != nil {
		slog.Error("list companies failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cs)
}

// Create resolves an ATS URL into a company and optionally tracks it for the caller.
func (h *CompaniesHandler) Create(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	var body struct {
		URL       string `json:"url"`
		Track     *bool  `json:"track"`
		ScrapeNow bool   `json:"scrape_now"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	source, token, ok := detect.ResolveBoard(body.URL)
	if !ok {
		http.Error(w, "could not resolve an ATS board from that URL", http.StatusUnprocessableEntity)
		return
	}

	company, err := h.companies.UpsertCompany(r.Context(), dto.CompanyUpsert{
		Slug:      token,
		Name:      humanizeToken(token),
		ATSSource: source,
		ATSToken:  token,
	})
	if err != nil {
		slog.Error("upsert company failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	track := body.Track == nil || *body.Track
	if track {
		tracking, err := h.companies.SetCompanyTracking(r.Context(), session.UserID, company.ID, true, 360)
		if err != nil {
			slog.Error("track company failed", slog.Any("err", err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		target, err := h.targets.UpsertSourceTargetForCompany(r.Context(), session.UserID, source, token, company.ID, true, tracking.CheckIntervalMinutes)
		if err != nil {
			slog.Error("upsert source target for company failed", slog.Any("err", err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if body.ScrapeNow && h.q != nil {
			if err := h.q.EnqueueScrapeRequest(r.Context(), dto.ScrapeRequest{Target: target}); err != nil {
				slog.Error("enqueue scrape request failed", slog.Any("err", err))
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(company)
}

// SetTracking stores the caller's company interest and requested check frequency.
func (h *CompaniesHandler) SetTracking(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.SessionFromContext(r.Context())
	id := chi.URLParam(r, "id")
	var body struct {
		Enabled              *bool `json:"enabled"`
		CheckIntervalMinutes *int  `json:"check_interval_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Enabled == nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	interval := 0
	if body.CheckIntervalMinutes != nil {
		interval = *body.CheckIntervalMinutes
	}
	if body.CheckIntervalMinutes != nil && interval < 60 {
		http.Error(w, "check_interval_minutes must be at least 60", http.StatusBadRequest)
		return
	}

	company, err := h.companies.GetCompany(r.Context(), id)
	if err != nil {
		if errors.Is(err, providers.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("get company failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	tracking, err := h.companies.SetCompanyTracking(r.Context(), session.UserID, company.ID, *body.Enabled, interval)
	if err != nil {
		slog.Error("set company tracking failed", slog.Any("err", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if company.ATSSource != "" {
		_, err := h.targets.UpsertSourceTargetForCompany(r.Context(), session.UserID, company.ATSSource, company.ATSToken, company.ID, *body.Enabled, tracking.CheckIntervalMinutes)
		if err != nil {
			slog.Error("sync legacy source target failed", slog.Any("err", err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tracking)
}

// humanizeToken turns a board token like "acme-corp" into "Acme Corp".
// ponytail: name derived from the token; good enough until a source carries a real company name.
func humanizeToken(token string) string {
	words := strings.Split(token, "-")
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}
