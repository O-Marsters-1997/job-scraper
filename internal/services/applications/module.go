// Package applications is the applications context: Applications and their
// Statuses (ADR 0011).
package applications

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/services/applications/store"
)

type Deps struct {
	Store Store
}

type Module struct {
	service *Service
	store   Store
}

func Build(deps Deps) *Module {
	return &Module{service: NewService(deps.Store), store: deps.Store}
}

func New(pool *pgxpool.Pool, events store.EventRecorder) *Module {
	return Build(Deps{Store: store.New(pool, events)})
}

// SeedDefaults seeds userID's default Statuses inside tx; identity's signup
// calls this through its own local StatusSeeder interface (ADR 0011).
func (m *Module) SeedDefaults(ctx context.Context, tx pgx.Tx, userID string) error {
	return m.store.SeedDefaultStatuses(ctx, tx, userID)
}
