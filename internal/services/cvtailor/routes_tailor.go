package cvtailor

import (
	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func (m *Module) tailorRoutes(r chi.Router) {
	r.Route("/tailoring", func(r chi.Router) {
		r.Get("/cvs/{docId}/{tabId}/headings", handlers.Query(m.svc.Headings))
		r.Put("/cvs/{docId}/{tabId}/headings", handlers.Update(m.svc.SaveHeadings))
		r.Get("/jobs/{jobId}/suggestions", handlers.Query(m.svc.Suggestions))
	})
}
