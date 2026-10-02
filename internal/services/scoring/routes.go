package scoring

import (
	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func (m *Module) Routes(r chi.Router) {
	r.Get("/scoring-config", handlers.GetAll(m.svc.GetConfig))
	r.Put("/scoring-config", handlers.Update(m.svc.UpdateConfig))
	r.Get("/scoring-options", handlers.GetAll(m.svc.Options))
	r.Get("/scores/status", handlers.GetAll(m.store.GetScoringStatus))
	r.Post("/scores/recompute", handlers.GetAll(m.svc.Recompute))
	r.Get("/push/vapid-public-key", handlers.GetAll(m.svc.VAPIDPublicKey))
	r.Post("/push/subscriptions", handlers.Create(m.svc.Subscribe))
	r.Delete("/push/subscriptions", handlers.Update(m.svc.Unsubscribe))
	r.Post("/push/test", handlers.Create(m.svc.TestPush))
	r.Post("/scoring-feedback/job", handlers.Create(m.svc.AppendJobFeedback))
	r.Post("/scoring-feedback/collection", handlers.Create(m.svc.AppendCollectionFeedback))
	r.Post("/scoring-feedback/overall", handlers.Create(m.svc.AppendOverallFeedback))
	r.Get("/scoring-feedback", handlers.Query(m.svc.ListFeedback))
	r.Delete("/scoring-feedback/{id}", handlers.Delete(m.svc.DeleteFeedback))
}
