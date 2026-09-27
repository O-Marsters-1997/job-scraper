// Package applications is the applications context: Applications and their
// Statuses (ADR 0011).
package applications

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/services/applications/store"
	"github.com/ollymarsters/job-scraper/internal/services/applicationstatuses"
)

type Module struct {
	store        *store.Store
	applications *Service
	statuses     *applicationstatuses.Service
}

func New(pool *pgxpool.Pool) *Module {
	st := store.New(pool)
	return &Module{
		store:        st,
		applications: NewService(st),
		statuses:     applicationstatuses.New(st),
	}
}

// SeedDefaults seeds userID's default Statuses inside tx; identity's signup
// calls this through its own local StatusSeeder interface (ADR 0011).
func (m *Module) SeedDefaults(ctx context.Context, tx pgx.Tx, userID string) error {
	return m.store.SeedDefaultStatuses(ctx, tx, userID)
}
