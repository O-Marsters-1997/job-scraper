package tasks

import (
	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func (m *Module) Routes(r chi.Router) {
	r.Route("/task-types", func(r chi.Router) {
		r.Get("/", handlers.GetAll(m.store.ListTaskTypes))
		r.Post("/", handlers.Create(m.service.CreateType))
		r.Patch("/{id}", handlers.Update(m.service.UpdateType))
		r.Delete("/{id}", handlers.Delete(m.store.DeleteTaskType))
	})
	r.Route("/tasks", func(r chi.Router) {
		r.Get("/", handlers.Query(m.service.List))
		r.Post("/", handlers.Create(m.service.Create))
		r.Patch("/{id}", handlers.Update(m.service.Update))
		r.Put("/{id}/done", handlers.Update(m.service.SetDone))
	})
}
