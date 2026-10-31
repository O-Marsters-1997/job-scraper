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

	r.Get("/due", handlers.GetAll(m.store.ListDue))

	r.Route("/contacts", func(r chi.Router) {
		r.Get("/", handlers.GetAll(m.store.ListContacts))
		r.Get("/companies", handlers.GetAll(m.store.ListContactCompanies))
		r.Post("/", handlers.Create(m.service.CreateContact))
		r.Delete("/{id}", handlers.Delete(m.store.DeleteContact))
		r.Post("/{id}/sent", handlers.Update(m.service.MarkContactSent))
		r.Post("/{id}/reply", handlers.Update(m.service.RecordContactReply))
	})

	r.Get("/jobs/{id}/note", handlers.GetByID(m.store.GetJobNote))
	r.Put("/jobs/{id}/note", handlers.Update(m.service.PutJobNote))

	r.Get("/jobs/{id}/comments", handlers.GetByID(m.store.ListJobComments))
	r.Post("/jobs/{id}/comments", handlers.Create(m.service.CreateJobComment))
	r.Patch("/comments/{id}", handlers.Update(m.service.UpdateJobComment))
	r.Delete("/comments/{id}", handlers.Delete(m.store.DeleteJobComment))

	r.Route("/applications", func(r chi.Router) {
		r.Get("/", handlers.Query(m.service.List))
		r.Post("/", handlers.Create(m.service.Create))
		r.Patch("/{id}", handlers.Update(m.service.Update))
		r.Delete("/{id}", handlers.Delete(m.store.DeleteApplication))
		r.Post("/delete", handlers.Update(m.service.DeleteApplications))
		r.Put("/{id}/status", handlers.Update(m.service.Move))
		r.Put("/{id}/chase", handlers.Update(m.service.SetChase))
		r.Delete("/{id}/chase", handlers.Delete(m.store.ClearChase))
		r.Get("/for-jobs", handlers.Query(m.service.ForJobs))
	})
}
