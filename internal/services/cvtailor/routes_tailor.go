package cvtailor

import (
	"context"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func (m *Module) tailorRoutes(r chi.Router) {
	r.Route("/tailoring", func(r chi.Router) {
		r.Get("/cvs/{docId}/{tabId}/headings", handlers.Query(m.svc.Headings))
		r.Put("/cvs/{docId}/{tabId}/headings", handlers.Update(m.svc.SaveHeadings))
		r.Get("/jobs/{jobId}/suggestions", handlers.Query(m.svc.Suggestions))
		r.Post("/drafts", m.createDraft())
		r.Get("/jobs/{jobId}/drafts", handlers.Query(m.svc.ListJobDrafts))
		r.Get("/drafts/{id}", handlers.Query(m.svc.GetDraft))
		r.Get("/drafts/{id}/pdf", m.draftPDF())
		r.Post("/drafts/{id}/keep", handlers.Update(m.svc.KeepDraft))
		r.Post("/drafts/{id}/discard", handlers.Update(m.svc.DiscardDraft))
	})
}

func (m *Module) createDraft() http.HandlerFunc {
	type req struct {
		userID string
		in     dto.DraftInput
	}
	return handlers.Handle(
		func(r *http.Request) (req, error) {
			uid, err := handlers.UserID(r)
			if err != nil {
				return req{}, err
			}
			in, err := handlers.DecodeBody[dto.DraftInput](r)
			return req{userID: uid, in: in}, err
		},
		func(ctx context.Context, in req) (dto.DraftRef, error) {
			return m.svc.CreateDraft(ctx, in.userID, in.in)
		},
		func(w http.ResponseWriter, _ *http.Request, ref dto.DraftRef) {
			handlers.WriteJSON(w, http.StatusAccepted, ref)
		},
	)
}

func (m *Module) draftPDF() http.HandlerFunc {
	type req struct {
		userID string
		q      dto.DraftQuery
	}
	return handlers.Handle(
		func(r *http.Request) (req, error) {
			uid, err := handlers.UserID(r)
			return req{userID: uid, q: dto.DraftQuery{ID: chi.URLParam(r, "id")}}, err
		},
		func(ctx context.Context, in req) (io.ReadCloser, error) {
			return m.svc.DraftPDF(ctx, in.userID, in.q)
		},
		func(w http.ResponseWriter, _ *http.Request, body io.ReadCloser) {
			defer func() { _ = body.Close() }()
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", `inline; filename="cv-draft.pdf"`)
			_, _ = io.Copy(w, body)
		},
	)
}
