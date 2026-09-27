package scoring

import (
	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func (m *Module) Routes(r chi.Router) {
	r.Get("/scoring-config", handlers.GetAll(m.scoringConfig.Get))
	r.Put("/scoring-config", handlers.Update(m.scoringConfig.Update))
	r.Get("/scoring-options", handlers.GetAll(m.scoringConfig.Options))
	r.Get("/scores/status", handlers.GetAll(m.store.GetScoringStatus))
	r.Post("/scores/recompute", handlers.GetAll(m.scoring.Recompute))
}
