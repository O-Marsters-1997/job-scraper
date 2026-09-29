package tailoring

import (
	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func (m *Module) Routes(r chi.Router) {
	r.Route("/experience", func(r chi.Router) {
		r.Get("/", handlers.GetAll(m.store.ListPositions))
		r.Post("/positions", handlers.Create(m.svc.CreatePosition))
		r.Put("/positions/order", handlers.Update(m.svc.ReorderPositions))
		r.Patch("/positions/{id}", handlers.Update(m.svc.UpdatePosition))
		r.Delete("/positions/{id}", handlers.Delete(m.store.DeletePosition))
		r.Post("/positions/{positionId}/achievements", handlers.Create(m.svc.CreateAchievement))
		r.Put("/positions/{positionId}/achievements/order", handlers.Update(m.svc.ReorderAchievements))
		r.Patch("/achievements/{id}", handlers.Update(m.svc.UpdateAchievement))
		r.Delete("/achievements/{id}", handlers.Delete(m.store.DeleteAchievement))
	})
}
