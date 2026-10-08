package cvtailor

import (
	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func (m *Module) Routes(r chi.Router) {
	r.Route("/experience", func(r chi.Router) {
		r.Get("/", handlers.GetAll(m.store.ListPositions))
		r.Post("/import/preview", handlers.Update(m.svc.PreviewImport))
		r.Post("/import", handlers.Create(m.svc.ImportPositions))
		r.Post("/positions", handlers.Create(m.svc.CreatePosition))
		r.Put("/positions/order", handlers.Update(m.svc.ReorderPositions))
		r.Patch("/positions/{id}", handlers.Update(m.svc.UpdatePosition))
		r.Delete("/positions/{id}", handlers.Delete(m.store.DeletePosition))
		r.Get("/skills", handlers.GetAll(m.store.ListBankSkills))
		r.Post("/skills", handlers.Create(m.svc.CreateBankSkill))
		r.Put("/skills/order", handlers.Update(m.svc.ReorderBankSkills))
		r.Patch("/skills/{id}", handlers.Update(m.svc.UpdateBankSkill))
		r.Delete("/skills/{id}", handlers.Delete(m.store.DeleteBankSkill))
		r.Post("/positions/{positionId}/achievements", handlers.Create(m.svc.CreateAchievement))
		r.Put("/positions/{positionId}/achievements/order", handlers.Update(m.svc.ReorderAchievements))
		r.Patch("/achievements/{id}", handlers.Update(m.svc.UpdateAchievement))
		r.Delete("/achievements/{id}", handlers.Delete(m.store.DeleteAchievement))
	})
	m.tailorRoutes(r)
}
