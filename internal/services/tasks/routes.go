package tasks

import (
	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func (m *Module) Routes(r chi.Router) {
	r.Get("/task-types", handlers.GetAll(m.store.ListTaskTypes))
	r.Route("/tasks", func(r chi.Router) {
		r.Get("/", handlers.Query(m.service.List))
		r.Post("/", handlers.Create(m.service.Create))
		r.Put("/{id}/done", handlers.Update(m.service.SetDone))
	})
}
