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

	r.Get("/recruiter-contacts/summaries", handlers.Query(m.service.RecruiterContactSummaries))
	r.Get("/jobs/{id}/recruiter-contacts", handlers.GetByID(m.store.ListRecruiterContacts))
	r.Post("/jobs/{id}/recruiter-contacts", handlers.Create(m.service.LogRecruiterContact))

	r.Get("/jobs/{id}/notes", handlers.GetByID(m.store.ListJobNotes))
	r.Post("/jobs/{id}/notes", handlers.Create(m.service.CreateJobNote))
	r.Patch("/notes/{id}", handlers.Update(m.service.UpdateJobNote))
	r.Delete("/notes/{id}", handlers.Delete(m.store.DeleteJobNote))

	r.Route("/applications", func(r chi.Router) {
		r.Get("/", handlers.Query(m.service.List))
		r.Post("/", handlers.Create(m.service.Create))
		r.Patch("/{id}", handlers.Update(m.service.Update))
		r.Delete("/{id}", handlers.Delete(m.store.DeleteApplication))
		r.Put("/{id}/chase", handlers.Update(m.service.SetChase))
		r.Delete("/{id}/chase", handlers.Delete(m.store.ClearChase))
		r.Get("/for-jobs", handlers.Query(m.service.ForJobs))
	})
}
