package applications

import (
	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func (m *Module) Routes(r chi.Router) {
	r.Route("/application-statuses", func(r chi.Router) {
		r.Get("/", handlers.GetAll(m.listStatuses))
		r.Post("/", handlers.Create(m.statuses.Create))
		r.Patch("/{id}", handlers.Update(m.statuses.Update))
		r.Delete("/{id}", handlers.Delete(m.statuses.Delete))
	})

	r.Route("/applications", func(r chi.Router) {
		r.Get("/", handlers.Query(m.applications.List))
		r.Post("/", handlers.Create(m.applications.Create))
		r.Patch("/{id}", handlers.Update(m.applications.Update))
		r.Delete("/{id}", handlers.Delete(m.applications.Delete))
		r.Get("/for-jobs", handlers.Query(m.applications.ForJobs))
	})
}
