package applications

import (
	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func (m *Module) Routes(r chi.Router) {
	r.Route("/application-statuses", func(r chi.Router) {
		r.Get("/", handlers.GetAll(m.store.ListApplicationStatusesByUser))
		r.Post("/", handlers.Create(m.service.CreateStatus))
		r.Patch("/{id}", handlers.Update(m.service.UpdateStatus))
		r.Delete("/{id}", handlers.Delete(m.service.DeleteStatus))
	})

	r.Route("/applications", func(r chi.Router) {
		r.Get("/", handlers.Query(m.service.List))
		r.Post("/", handlers.Create(m.service.Create))
		r.Patch("/{id}", handlers.Update(m.service.Update))
		r.Delete("/{id}", handlers.Delete(m.service.Delete))
		r.Get("/for-jobs", handlers.Query(m.service.ForJobs))
	})
}
