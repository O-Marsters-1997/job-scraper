// Package applications is the applications context: Applications and their
// Statuses (ADR 0011).
package applications

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/applications/store"
	"github.com/ollymarsters/job-scraper/internal/services/applicationstatuses"
)

// Deps are the stores Build wires into the Module; New builds the real ones
// and calls Build. Tests call Build directly with fakes (ADR 0012).
type Deps struct {
	Applications Store
	Statuses     applicationstatuses.Store
}

type Module struct {
	applications *Service
	statuses     *applicationstatuses.Service
	listStatuses func(ctx context.Context, userID string) ([]dto.ApplicationStatus, error)
	seedDefaults func(ctx context.Context, tx pgx.Tx, userID string) error
}

func Build(deps Deps) *Module {
	return &Module{
		applications: NewService(deps.Applications),
		statuses:     applicationstatuses.New(deps.Statuses),
		listStatuses: deps.Statuses.ListApplicationStatusesByUser,
		seedDefaults: deps.Statuses.SeedDefaultStatuses,
	}
}

func New(pool *pgxpool.Pool) *Module {
	st := store.New(pool)
	return Build(Deps{Applications: st, Statuses: st})
}

// SeedDefaults seeds userID's default Statuses inside tx; identity's signup
// calls this through its own local StatusSeeder interface (ADR 0011).
func (m *Module) SeedDefaults(ctx context.Context, tx pgx.Tx, userID string) error {
	return m.seedDefaults(ctx, tx, userID)
}
