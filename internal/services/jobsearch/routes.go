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
	r.Get("/jobs/all", handlers.GetAll(m.jobStore.ListJobs))
	r.Get("/jobs/{id}", handlers.GetByID(m.jobs.Get))

	r.Get("/sources", handlers.GetAll(m.sources.List))
	r.Get("/sources/resolve", handlers.Query(m.sources.Resolve))

	r.Route("/source-targets", func(r chi.Router) {
		r.Get("/", handlers.GetAll(m.targetLister.ListSourceTargetsByUser))
		r.Post("/", handlers.Create(m.sourceTargets.Create))
		r.Patch("/{id}", handlers.Update(m.sourceTargets.Update))
		r.Post("/{id}/scrape", handlers.GetByID(m.sourceTargets.Scrape))
		r.Delete("/{id}", handlers.Delete(m.sourceTargets.Delete))
	})

	r.Route("/companies", func(r chi.Router) {
		r.Get("/", handlers.GetAll(m.jobStore.ListCompaniesForUser))
		r.Post("/", handlers.Create(m.companies.Create))
		r.Put("/{id}/tracking", handlers.Update(m.companies.SetTracking))
		r.Get("/{id}/boards", handlers.GetByID(m.companies.ListBoards))
		r.Post("/{id}/boards", handlers.Create(m.companies.AddBoard))
	})
}

// PublicRoutes registers /ingest, authenticated by service token. The
// router applies ServiceTokenMiddleware itself: depguard's "shared" rule
// keeps internal/api out of internal/services/** (ADR 0011).
func (m *Module) PublicRoutes(r chi.Router) {
	r.Post("/ingest", handlers.Handle(
		handlers.DecodeBody[dto.Job],
		func(ctx context.Context, job dto.Job) (IngestResult, error) {
			results, err := m.ingest.IngestJobs(ctx, []dto.Job{job})
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
			results, err := m.ingest.IngestJobs(ctx, in.Jobs)
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
