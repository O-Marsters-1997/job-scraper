package cvtailor

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func (m *Module) tailorRoutes(r chi.Router) {
	r.Route("/tailoring", func(r chi.Router) {
		r.Get("/cvs/{docId}/{tabId}/headings", m.getHeadings())
		r.Put("/cvs/{docId}/{tabId}/headings", handlers.Update(m.svc.SaveHeadings))
		r.Get("/jobs/{jobId}/suggestions", m.getSuggestions())
	})
}

type tabReq struct{ userID, docID, tabID string }

func (m *Module) getHeadings() http.HandlerFunc {
	return handlers.Handle(
		func(r *http.Request) (tabReq, error) {
			uid, err := handlers.UserID(r)
			return tabReq{uid, chi.URLParam(r, "docId"), chi.URLParam(r, "tabId")}, err
		},
		func(ctx context.Context, q tabReq) ([]dto.CVHeading, error) {
			return m.svc.Headings(ctx, q.userID, q.docID, q.tabID)
		},
		writeOK[[]dto.CVHeading],
	)
}

type suggestionsReq struct{ userID, jobID, docID, tabID string }

func (m *Module) getSuggestions() http.HandlerFunc {
	return handlers.Handle(
		func(r *http.Request) (suggestionsReq, error) {
			uid, err := handlers.UserID(r)
			return suggestionsReq{uid, chi.URLParam(r, "jobId"), r.URL.Query().Get("docId"), r.URL.Query().Get("tabId")}, err
		},
		func(ctx context.Context, q suggestionsReq) ([]dto.Suggestion, error) {
			return m.svc.Suggestions(ctx, q.userID, q.jobID, q.docID, q.tabID)
		},
		writeOK[[]dto.Suggestion],
	)
}

func writeOK[T any](w http.ResponseWriter, _ *http.Request, v T) {
	handlers.WriteJSON(w, http.StatusOK, v)
}
