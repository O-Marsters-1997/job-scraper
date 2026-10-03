package jobsearch

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func (m *Module) Routes(r chi.Router) {
	r.Get("/jobs", handlers.Query(m.jobs.List))
	r.Get("/jobs/all", handlers.GetAll(m.jobs.ListScored))
	r.Post("/jobs/seen", handlers.Create(m.jobs.MarkSeen))
	r.Get("/jobs/{id}", handlers.GetByID(m.jobs.Get))

	r.Get("/sources", handlers.GetAll(m.jobs.ListSources))
	r.Get("/sources/resolve", handlers.Query(m.jobs.ResolveBoard))

	r.Route("/source-targets", func(r chi.Router) {
		r.Get("/", handlers.GetAll(m.sourceTargets.List))
		r.Post("/", handlers.Create(m.sourceTargets.Create))
		r.Patch("/{id}", handlers.Update(m.sourceTargets.Update))
		r.Post("/{id}/scrape", handlers.GetByID(m.sourceTargets.Scrape))
		r.Delete("/{id}", handlers.Delete(m.sourceTargets.Delete))
	})

	r.Route("/companies", func(r chi.Router) {
		r.Get("/", handlers.Query(m.jobs.ListCompanies))
		r.Post("/", handlers.Create(m.jobs.CreateCompany))
		r.Get("/{id}", handlers.GetByID(m.jobs.GetCompany))
		r.Get("/new", handlers.GetAll(m.ListNewCompanies))
		r.Get("/tracked", handlers.GetAll(m.jobs.ListTrackedCompanies))
		r.Delete("/{id}/tracking", handlers.Delete(m.jobs.UntrackCompany))
		r.Put("/{id}/tracking", handlers.Update(m.jobs.SetCompanyTracking))
		r.Put("/{id}/review", handlers.Update(m.jobs.SetCompanyReview))
		r.Get("/{id}/boards", handlers.GetByID(m.jobs.ListCompanyBoards))
		r.Post("/{id}/boards", handlers.Create(m.jobs.AddCompanyBoard))
	})
}

// PublicRoutes registers /ingest, authenticated by service token. The
// router applies ServiceTokenMiddleware itself: depguard's "shared" rule
// keeps internal/api out of internal/services/** (ADR 0011).
func (m *Module) PublicRoutes(r chi.Router) {
	r.Post("/ingest", handlers.Handle(
		handlers.DecodeBody[dto.Job],
		func(ctx context.Context, job dto.Job) (IngestResult, error) {
			results, err := m.jobs.IngestJobs(ctx, []dto.Job{job})
			if err != nil {
				return IngestResult{}, err
			}
			return results[0], nil
		},
		func(w http.ResponseWriter, _ *http.Request, res IngestResult) {
			handlers.WriteJSON(w, http.StatusOK, res)
		},
	))

	r.With(chimw.RequestSize(2<<20)).Post("/ingest/batch", handlers.Handle(
		decodeIngestBatch,
		func(ctx context.Context, in ingestBatchInput) (ingestBatchView, error) {
			results, err := m.jobs.IngestJobs(ctx, in.Jobs)
			if err != nil {
				return ingestBatchView{}, err
			}
			return ingestBatchView{Results: results}, nil
		},
		func(w http.ResponseWriter, _ *http.Request, res ingestBatchView) {
			handlers.WriteJSON(w, http.StatusOK, res)
		},
	))
}

type ingestBatchInput struct {
	Jobs []dto.Job `json:"jobs"`
}

type ingestBatchView struct {
	Results []IngestResult `json:"results"`
}

func decodeIngestBatch(r *http.Request) (ingestBatchInput, error) {
	var in ingestBatchInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.Jobs) == 0 || len(in.Jobs) > 100 {
		return ingestBatchInput{}, apperr.Invalid("invalid batch")
	}
	return in, nil
}
