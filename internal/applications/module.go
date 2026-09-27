// Package applications is the applications context: Applications and their
// Statuses (ADR 0011).
package applications

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/applications/internal/application"
	"github.com/ollymarsters/job-scraper/internal/applications/internal/applicationstatuses"
	"github.com/ollymarsters/job-scraper/internal/applications/internal/store"
)

type Module struct {
	store        *store.Store
	applications *application.Service
	statuses     *applicationstatuses.Service
}

func New(pool *pgxpool.Pool) *Module {
	st := store.New(pool)
	return &Module{
		store:        st,
		applications: application.New(st),
		statuses:     applicationstatuses.New(st),
	}
}

// SeedDefaults seeds userID's default Statuses; legacy signup calls this
// through its own local StatusSeeder interface (ADR 0011).
func (m *Module) SeedDefaults(ctx context.Context, userID string) error {
	return m.store.SeedDefaultStatuses(ctx, userID)
}
